package handler

import (
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
	"logo_api/auth"
	"logo_api/dao/mysql"
	"logo_api/model"
	"logo_api/service"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func GetUserList(svc *service.ResourceService) gin.HandlerFunc {
	return func(c *gin.Context) {
		pageStr := c.DefaultQuery("page", "1")
		pageSizeStr := c.DefaultQuery("pageSize", "10")
		keyword := c.DefaultQuery("keyword", "")
		sortBy := c.DefaultQuery("sortBy", "id")
		sortOrder := c.DefaultQuery("sortOrder", "asc")

		// 统一处理空值和默认值(调 apifox 时，即使字段为空，也会传入空字符串，因此需要手动处理空字符串)
		if pageStr == "" {
			pageStr = "1"
		}
		if pageSizeStr == "" {
			pageSizeStr = "10"
		}
		if sortBy == "" {
			sortBy = "id"
		}
		if sortOrder == "" {
			sortOrder = "asc"
		}

		// 将字符串转换为整数 (int)
		page, err := strconv.Atoi(pageStr)
		if err != nil {
			zap.L().Error("strconv.Atoi(pageStr) Error", zap.String("pageStr", pageStr), zap.Error(err), zap.String("keyword", keyword))
			// 处理错误：如果转换失败，返回 400 错误给客户端
			c.JSON(400, model.Response{
				Code:    400,
				Message: "Invalid page parameter",
				Data:    nil,
			})
			return
		}

		pageSize, err := strconv.Atoi(pageSizeStr)
		if err != nil {
			zap.L().Error("strconv.Atoi(pageSizeStr) Error", zap.String("pageSizeStr", pageSizeStr), zap.Error(err), zap.String("keyword", keyword))
			// 处理错误：如果转换失败，返回 400 错误给客户端
			c.JSON(400, model.Response{
				Code:    400,
				Message: "Invalid page parameter",
				Data:    nil,
			})
			return
		}

		// 参数有效性检查（推荐）
		// 确保 page 和 pageSize 是正数
		if page <= 0 || pageSize <= 0 {
			c.JSON(400, model.Response{
				Code:    400,
				Message: "Invalid page parameter, page and pageSize must be greater than 0.",
				Data:    nil,
			})
			return
		}

		users, totalCount, err := svc.GetUserList(page, pageSize, keyword, sortBy, sortOrder)
		// 处理服务层错误
		if err != nil {
			// 记录日志，并返回 500（内部错误）或 400（如果确定是客户端输入导致的服务错误）
			zap.L().Error("svc.GetUserList failed", zap.Error(err),
				zap.Int("page", page), zap.Int("pageSize", pageSize), zap.String("keyword", keyword))
			c.JSON(500, model.Response{
				Code:    500,
				Message: "Failed to retrieve user list",
				Data:    nil,
			})
			return // 确保错误后退出
		}
		// 根据 totalCount 进行判断
		if totalCount == 0 && keyword != "" {
			// 如果 totalCount 为 0 且用户使用了 keyword 进行搜索

			// 保持 200，但修改 Message
			// 客户端看到 200 状态码知道接口运行正常，但可以根据 Message 提示用户
			message := fmt.Sprintf("No users found matching keyword '%s'", keyword)
			zap.L().Info(message, zap.Int("page", page), zap.Int("pageSize", pageSize), zap.Int64("totalCount", totalCount), zap.String("keyword", keyword))

			c.JSON(http.StatusOK, model.Response{
				Code:    http.StatusOK,
				Message: message,
				Data: gin.H{
					"list":       users, // list 为空 []
					"page":       page,
					"pageSize":   pageSize,
					"totalCount": totalCount, // totalCount 为 0
				},
			})
			return
		}

		// 成功返回响应
		// 成功的 API 响应通常返回 200 OK，并包含数据。
		// 为了完整性，一个列表接口通常还需要返回总记录数（Total Count），但这里代码中没有体现，
		// 我们只返回用户列表。
		zap.L().Info("success get user list", zap.Int("page", page), zap.Int("pageSize", pageSize), zap.Int64("totalCount", totalCount), zap.String("keyword", keyword))
		c.JSON(200, model.Response{
			Code:    200,
			Message: "Success get user list",
			Data: gin.H{
				"list":       users,
				"page":       page,
				"pageSize":   pageSize,
				"totalCount": totalCount,
			},
		})
	}
}

func RegisterFunc(svc *service.ResourceService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. 处理请求参数
		var req model.ReqUser
		if err := c.ShouldBind(&req); err != nil {
			zap.L().Error("Register() failed, Invalid request body or missing fields.")
			c.JSON(http.StatusBadRequest, model.Response{
				Code:    http.StatusBadRequest,
				Message: "Invalid request body or missing fields.",
			})
			return
		}
		// 2. 校验参数
		username := strings.TrimSpace(req.Username)
		password := strings.TrimSpace(req.Password) // 注意：password 必须是明文才能加密

		// 2.1 校验用户名是否为空（TrimSpace后）
		if username == "" {
			zap.L().Error("Register() failed, Username cannot be empty or consist only of spaces.", zap.String("username", username))
			c.JSON(http.StatusBadRequest, model.Response{
				Code:    http.StatusBadRequest,
				Message: "Username cannot be empty or consist only of spaces.",
			})
			return
		}

		// 2.2 校验密码是否为空（TrimSpace后）
		if password == "" {
			zap.L().Error("Register() failed, Password cannot be empty or consist only of spaces.", zap.String("username", username))
			c.JSON(http.StatusBadRequest, model.Response{
				Code:    http.StatusBadRequest,
				Message: "Password cannot be empty or consist only of spaces.",
			})
			return
		}

		// 2.3 校验长度（长度最短为6）
		const minLength = 6
		if len(password) < minLength {
			zap.L().Error("Register() failed, Password must be at least 6 characters.", zap.String("username", username))
			c.JSON(http.StatusBadRequest, model.Response{
				Code:    http.StatusBadRequest,
				Message: fmt.Sprintf("Password must be at least %d characters long.", minLength),
			})
			return
		}

		// 3. 检查用户是否存在
		_, err := svc.GetUserFromName(username)
		if err == nil {
			// 用户已存在，返回 409 Conflict
			c.JSON(http.StatusConflict, model.Response{
				Code:    http.StatusConflict,
				Message: "Username already exists.",
			})
			return

		}
		// 4. 匹配 "用户不存在" 错误（流程应该继续）
		// 需要导入 mysql 包
		if errors.Is(err, mysql.ErrUserNotFound) {
			// 用户不存在，这是注册流程的预期行为，继续执行注册 (步骤 5)
			// 注意：这里不需要 return，流程会自然向下执行到密码哈希和插入
		} else {
			// 匹配到其他数据库错误 (非 nil, 非 ErrUserNotFound)
			zap.L().Error("get user error", zap.String("username", username), zap.Error(err))
			c.JSON(http.StatusInternalServerError, model.Response{
				Code:    http.StatusInternalServerError,
				Message: "Server database error.",
			})
			return
		}

		// 5. 对密码进行哈希处理 (关键安全步骤)
		hashedPassword, hashErr := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if hashErr != nil {
			zap.L().Error("bcrypt hashing failed", zap.Error(hashErr))
			c.JSON(http.StatusInternalServerError, model.Response{
				Code:    http.StatusInternalServerError,
				Message: "Failed to process password.",
			})
			return
		}
		// 6. 插入新用户 (存储哈希后的密码)
		newUser := model.User{
			Status:   model.StatusActive, // 1 启用 0 禁用
			Username: username,
			Password: string(hashedPassword), // 存储哈希值
		}

		if err = svc.InsertUser(newUser); err != nil {
			zap.L().Error("insert user error", zap.String("username", username), zap.Error(err))
			c.JSON(http.StatusInternalServerError, model.Response{
				Code:    http.StatusInternalServerError,
				Message: "Registration failed due to server error.",
			})
			return
		}

		// 7. 注册成功，返回 201 Created
		zap.L().Info("register success", zap.String("username", username))
		c.JSON(http.StatusCreated, model.Response{
			Code:    http.StatusCreated,
			Message: "User registered successfully.",
		})
	}
}

func UserLogin(svc *service.ResourceService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. 请求参数处理
		var req model.ReqUser
		if err := c.ShouldBind(&req); err != nil {
			zap.L().Error("Login() failed, Invalid request body or missing fields.", zap.Error(err))
			c.JSON(http.StatusBadRequest, model.Response{
				Code:    http.StatusBadRequest,
				Message: "Invalid request body or missing fields.",
			})
			return
		}
		// 2. 请求参数校验
		username := strings.TrimSpace(req.Username)
		password := strings.TrimSpace(req.Password)
		if username == "" || password == "" {
			zap.L().Error("Login() failed, Invalid username or password format.")
			c.JSON(http.StatusBadRequest, model.Response{
				Code:    http.StatusBadRequest,
				Message: "Username and password cannot be empty.",
			})
			return
		}

		// 3. 查询用户是否存在
		user, err := svc.GetUserFromName(username)
		if err != nil {
			if errors.Is(err, mysql.ErrUserNotFound) {
				// 用户不存在。为了安全，避免泄露“用户不存在”的信息，统一返回“用户名或密码错误”。
				zap.L().Warn("Login failed: User not found", zap.String("username", username))
				c.JSON(http.StatusUnauthorized, model.Response{
					Code:    http.StatusUnauthorized, // 401 Unauthorized
					Message: "Invalid username or password.",
				})
				return
			}
			// 数据库查询错误
			zap.L().Error("Login failed: database error", zap.String("username", username), zap.Error(err))
			c.JSON(http.StatusInternalServerError, model.Response{
				Code:    http.StatusInternalServerError,
				Message: "Server database error during login.",
			})
			return
		}

		// 4. 验证密码 (使用 bcrypt）
		// user.Password 是哈希值(数据库)，password 是明文（客户端传入）
		err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
		if err != nil {
			if err == bcrypt.ErrMismatchedHashAndPassword {
				zap.L().Warn("Login failed: Password mismatch", zap.String("username", username))
				c.JSON(http.StatusUnauthorized, model.Response{
					Code:    http.StatusUnauthorized,
					Message: "Invalid username or password.",
				})
				return
			}
			// 其他 bcrypt 错误
			zap.L().Error("Login failed: bcrypt error", zap.String("username", username), zap.Error(err))
			c.JSON(http.StatusInternalServerError, model.Response{
				Code:    http.StatusInternalServerError, // 500
				Message: "Failed to verify password.",
			})
			return
		}

		// 5. 登录成功 (可以考虑生成 JWT 或 Session)
		token, tokenErr := auth.CreateToken(user.ID, user.Username)

		if tokenErr != nil {
			zap.L().Error("Login failed: Failed to generate JWT token.", zap.Error(tokenErr))
			c.JSON(http.StatusInternalServerError, model.Response{
				Code:    http.StatusInternalServerError,
				Message: "Login successful but failed to generate authentication token.",
			})
			return
		}
		// 关键新增步骤：将新生成的 Token 存储到 Redis
		// 获取 Token 的过期时间
		expirationTime, _ := auth.GetTokenExpiration(token) // 假设 GetTokenExpiration 能获取到
		duration := time.Until(expirationTime)

		if duration > 0 {
			err = svc.StoreUserToken(c.Request.Context(), user.ID, token, duration)
			if err != nil {
				zap.L().Error("UserLogin failed to store token in Redis", zap.Int("userID", user.ID), zap.Error(err))
				// 即使存储失败，仍应返回 Token，但建议返回服务器内部错误
				// c.JSON(http.StatusInternalServerError, ... )
				// return
			}
		}

		// 6. 返回成功响应，包含 token
		zap.L().Info("Login success", zap.String("username", username))
		c.JSON(http.StatusOK, model.Response{
			Code:    http.StatusOK,
			Message: "Login successful.",
			Data: gin.H{
				"username": username,
				"id":       user.ID,
				"token":    token,
			},
		})
	}
}

// UserLogout 是处理 POST /user/logout 请求的 Handler
// 它依赖 AuthRequired 中间件在 Context 中注入的用户信息。
func UserLogout(svc *service.ResourceService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. 从 Context 中安全获取中间件设置的参数
		// AuthRequired 中间件保证了这些值存在且有效

		// 获取用户ID
		userIDValue, ok := c.Get("user_id")
		if !ok {
			// 理论上AuthRequired已保证存在，这里作为安全检查
			zap.L().Error("UserLogout failed: user_id not found in context")
			c.JSON(http.StatusUnauthorized, model.Response{Code: http.StatusUnauthorized, Message: "Unauthorized: Missing user session data."})
			return
		}
		userID := userIDValue.(int)

		// 获取裸 Token 字符串
		tokenStringValue, ok := c.Get("tokenString")
		if !ok {
			zap.L().Error("UserLogout failed: tokenString not found in context")
			c.JSON(http.StatusUnauthorized, model.Response{Code: http.StatusUnauthorized, Message: "Unauthorized: Missing token string."})
			return
		}
		tokenString := tokenStringValue.(string)

		// 获取 JWT Claims (包含过期时间)
		userClaimsValue, ok := c.Get("user_claims")
		claims, ok := userClaimsValue.(*auth.UserClaims)
		if !ok || claims.ExpiresAt == nil {
			zap.L().Error("UserLogout failed: invalid or missing user claims in context")
			c.JSON(http.StatusUnauthorized, model.Response{Code: http.StatusUnauthorized, Message: "Unauthorized: Invalid claims."})
			return
		}

		// 2. 获取 Token 的过期时间
		expirationTime := claims.ExpiresAt.Time

		// 3. 调用 Service 层执行登出逻辑 (清除 SSO Session 和加入黑名单)
		err := svc.UserLogout(c.Request.Context(), userID, tokenString, expirationTime)

		if err != nil {
			// 如果 Service 层返回了非 nil 错误，可能是严重的 Redis 或服务器错误
			zap.L().Error("UserLogout failed due to service error",
				zap.Int("userID", userID),
				zap.Error(err))
			c.JSON(http.StatusInternalServerError, model.Response{
				Code:    http.StatusInternalServerError,
				Message: "Logout failed due to server error.",
			})
			return
		}

		// 4. 登出成功
		zap.L().Info("UserLogout successful", zap.Int("userID", userID))
		c.JSON(http.StatusOK, model.Response{
			Code:    200,
			Message: "Logout successful.",
		})
	}
}
