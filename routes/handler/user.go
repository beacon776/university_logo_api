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
	"logo_api/model/user/dto"
	"logo_api/model/user/vo"
	"logo_api/service"
	"strings"
	"time"
)

func RegisterFunc() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. 处理请求参数
		var req dto.UserRegisterReq
		if err := c.ShouldBind(&req); err != nil {
			zap.L().Error("Register() failed, Invalid request body or missing fields.")
			model.Error(c, model.CodeInvalidParam)
			return
		}
		// 2. 校验参数
		username := strings.TrimSpace(req.Username)
		password := strings.TrimSpace(req.Password) // 注意：password 必须是明文才能加密

		// 2.1 校验用户名是否合法
		if username == "" {
			zap.L().Error("Register() failed, Username cannot be empty or consist only of spaces.", zap.String("username", username))
			model.Error(c, model.CodeInvalidParam, "用户名不能为空")
			return
		}
		if username == "null" || username == "nil" {
			zap.L().Error("Register() failed, Username cannot be null or nil.", zap.String("username", username))
			model.Error(c, model.CodeInvalidParam, "用户名不能为 null 或 nil")
			return
		}
		// 2.2 校验密码是否合法
		if password == "" {
			zap.L().Error("Register() failed, Password cannot be empty or consist only of spaces.", zap.String("username", username))
			model.Error(c, model.CodeInvalidParam, "密码不能为空")
			return
		}
		if password == "null" || password == "nil" {
			zap.L().Error("Register() failed, Password cannot be null or nil.", zap.String("username", username))
			model.Error(c, model.CodeInvalidParam, "密码不能为 null 或 nil")
			return
		}

		// 2.3 校验长度（长度最短为6）
		const minLength = 6
		if len(password) < minLength {
			zap.L().Error("Register() failed, Password must be at least 6 characters.", zap.String("username", username))
			model.Error(c, model.CodeInvalidParam, "密码长度应不短于6")
			return
		}

		// 3. 检查用户是否存在
		_, err := service.GetUserFromName(username)
		if err == nil {
			// 用户已存在，返回 409 Conflict
			zap.L().Error("Register() failed, Username already exists.", zap.String("username", username))
			model.Error(c, model.CodeUserExist)
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
			model.Error(c, model.CodeServerErr, "数据库匹配发生错误")
			return
		}

		// 5. 对密码进行哈希处理 (关键安全步骤)
		hashedPassword, hashErr := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if hashErr != nil {
			zap.L().Error("bcrypt hashing failed", zap.Error(hashErr))
			model.Error(c, model.CodeServerErr, "哈希加密密码发生错误")
			return
		}

		// 6. 插入新用户 (存储哈希后的密码)
		newUser := dto.UserInsertReq{
			Status:   model.StatusActive, // 1 启用 0 禁用
			Username: username,
			Password: string(hashedPassword), // 存储哈希值
		}
		var insertId int
		if insertId, err = service.InsertUser(newUser); err != nil {
			zap.L().Error("insert user error", zap.String("username", username), zap.Error(err))
			model.Error(c, model.CodeServerErr, "插入用户发生错误")
			return
		}

		// 7. 注册成功，返回 200
		zap.L().Info("register success", zap.String("username", username))
		var voUser vo.UserRegisterResp
		voUser.ID = insertId
		model.Success(c, voUser)
	}
}

func UserLogin() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. 请求参数处理
		var userReq dto.UserLoginReq
		if err := c.ShouldBind(&userReq); err != nil {
			zap.L().Error("Login() failed, Invalid request body or missing fields.", zap.Error(err))
			model.Error(c, model.CodeInvalidParam)
			return
		}
		// 2. 请求参数校验
		username := strings.TrimSpace(userReq.Username)
		password := strings.TrimSpace(userReq.Password)
		if username == "" || password == "" {
			zap.L().Error("Login() failed, Invalid username or password format.")
			model.Error(c, model.CodeInvalidParam, "用户名或密码不能为空")
			return
		}

		// 3. 查询用户是否存在
		user, err := mysql.GetUserFromName(username)
		if err != nil {
			if errors.Is(err, mysql.ErrUserNotFound) {
				zap.L().Warn("Login failed: User not found", zap.String("username", username))
				model.Error(c, model.CodeUnauthorized, "用户不存在")
				return
			}
			// 数据库查询错误
			zap.L().Error("Login failed: database error", zap.String("username", username), zap.Error(err))
			model.Error(c, model.CodeServerErr, "数据库查询错误")
			return
		}

		// 4. 验证密码 (使用 bcrypt）
		// user.Password 是哈希值(数据库)，password 是明文（客户端传入）
		err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
		if err != nil {
			if err == bcrypt.ErrMismatchedHashAndPassword {
				zap.L().Warn("Login failed: Password mismatch", zap.String("username", username))
				model.Error(c, model.CodeUnauthorized, "用户名或密码错误")
				return
			}
			// 其他 bcrypt 错误
			zap.L().Error("Login failed: bcrypt error", zap.String("username", username), zap.Error(err))
			model.Error(c, model.CodeServerErr, "bcrypt error")
			return
		}

		// 5. 登录成功 (可以考虑生成 JWT 或 Session)
		token, tokenErr := auth.CreateToken(user.ID, user.Username)

		if tokenErr != nil {
			zap.L().Error("Login failed: Failed to generate JWT token.", zap.Error(tokenErr))
			model.Error(c, model.CodeServerErr, "登录后尝试生成 token 失败")
			return
		}
		// 关键新增步骤：将新生成的 Token 存储到 Redis
		// 获取 Token 的过期时间
		expirationTime, _ := auth.GetTokenExpiration(token) // 假设 GetTokenExpiration 能获取到
		duration := time.Until(expirationTime)

		if duration > 0 {
			err = service.StoreUserToken(c.Request.Context(), user.ID, token, duration)
			if err != nil {
				// 存储 token 失败
				zap.L().Error("UserLogin failed to store token in Redis", zap.Int("userID", user.ID), zap.Error(err))
				model.Error(c, model.CodeServerErr, "存储 token 失败")
				return
			}
		}

		// 6. 返回成功响应，包含 token
		zap.L().Info("Login success", zap.String("username", username))
		var userResp vo.UserLoginResp
		userResp.Token = token
		userResp.Username = username
		userResp.ID = user.ID
		if user.Status == model.StatusDeleted {
			userResp.Status = model.StatusDeletedStr
		} else if user.Status == model.StatusActive {
			userResp.Status = model.StatusActiveStr
		}

		model.Success(c, userResp)
	}
}

// UserLogout 是处理 POST /user/logout 请求的 Handler
// 它依赖 AuthRequired 中间件在 Context 中注入的用户信息。
func UserLogout() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. 从 Context 中安全获取中间件设置的参数
		// AuthRequired 中间件保证了这些值存在且有效

		// 获取用户ID
		userIDValue, ok := c.Get("user_id")
		if !ok {
			// 理论上AuthRequired已保证存在，这里作为安全检查
			zap.L().Error("UserLogout failed: user_id not found in context")
			model.Error(c, model.CodeServerErr, "Unauthorized: Missing user session data.")
			return
		}
		userID := userIDValue.(int)

		// 获取裸 Token 字符串
		tokenStringValue, ok := c.Get("tokenString")
		if !ok {
			zap.L().Error("UserLogout failed: tokenString not found in context")
			model.Error(c, model.CodeServerErr, "Unauthorized: Missing token string.")
			return
		}
		tokenString := tokenStringValue.(string)

		// 获取 JWT Claims (包含过期时间)
		userClaimsValue, ok := c.Get("user_claims")
		claims, ok := userClaimsValue.(*auth.UserClaims)
		if !ok || claims.ExpiresAt == nil {
			zap.L().Error("UserLogout failed: invalid or missing user claims in context")
			model.Error(c, model.CodeServerErr, "Unauthorized: Invalid JWT claims.")
			return
		}

		// 2. 获取 Token 的过期时间
		expirationTime := claims.ExpiresAt.Time

		// 3. 调用 Service 层执行登出逻辑 (清除 SSO Session 和加入黑名单)
		err := service.UserLogout(c.Request.Context(), userID, tokenString, expirationTime)

		if err != nil {
			// 如果 Service 层返回了非 nil 错误，可能是严重的 Redis 或服务器错误
			zap.L().Error("UserLogout failed due to service error",
				zap.Int("userID", userID),
				zap.Error(err))
			model.Error(c, model.CodeServerErr, "Logout failed due to server error.")
			return
		}

		// 4. 登出成功
		zap.L().Info("UserLogout successful", zap.Int("userID", userID))
		model.SuccessEmpty(c)
	}
}

func GetUserList() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req dto.UserGetListReq
		// 如果 Request Body 不为空，才进行 JSON 绑定
		if c.Request.ContentLength > 0 {
			if err := c.ShouldBindJSON(&req); err != nil { // 使用结构体 tag 进行 SortBy 和 SortOrder 范围合法检验
				zap.L().Error("c.ShouldBind(&userGetListDTO)", zap.Error(err))
				model.Error(c, model.CodeInvalidParam)
				return
			}
		}
		// 允许空参数的存在
		zap.L().Info("receive request body", zap.Any("req", req))
		// 设置默认值
		if req.Page <= 0 {
			req.Page = 1
		}
		if req.PageSize <= 0 {
			req.PageSize = 10
		}
		if req.SortBy == "" {
			req.SortBy = "id"
		}
		if req.SortOrder == "" {
			req.SortOrder = "asc"
		}

		// page、pageSize 参数范围有效性检查
		// 确保 page 和 pageSize 是正数
		if req.Page <= 0 || req.PageSize <= 0 {
			zap.L().Error("GetUserList() req param doesn't valid", zap.Any("req", req))
			model.Error(c, model.CodeInvalidParam, "Invalid page parameter, page and pageSize must be greater than 0.")
			return
		}

		users, totalCount, err := service.GetUserList(req)
		// 处理服务层错误
		if err != nil {
			// 记录日志，并返回 500（内部错误）或 400（如果确定是客户端输入导致的服务错误）
			zap.L().Error("svc.GetUserList failed", zap.Error(err),
				zap.Int("page", req.Page), zap.Int("pageSize", req.PageSize), zap.String("keyword", req.Keyword))
			model.Error(c, model.CodeServerErr, "Failed to retrieve user list.")
			return
		}
		// 根据 totalCount 进行判断1
		if totalCount == 0 && req.Keyword != "" {
			// 如果 totalCount 为 0 且用户使用了 keyword 进行搜索
			// 保持 200，但修改 Message
			// 客户端看到 200 状态码知道接口运行正常，但可以根据 Message 提示用户
			message := fmt.Sprintf("No users found matching keyword '%s'", req.Keyword)
			zap.L().Info(message, zap.Int("page", req.Page), zap.Int("pageSize", req.PageSize), zap.Int64("totalCount", totalCount), zap.String("keyword", req.Keyword))

			var userListResp vo.UserListResp
			userListResp.List = users
			userListResp.TotalCount = int(totalCount)
			model.Success(c, userListResp, message)
			return
		}

		// 成功返回响应
		zap.L().Info("success get user list", zap.Int("page", req.Page), zap.Int("pageSize", req.PageSize), zap.Int64("totalCount", totalCount), zap.String("keyword", req.Keyword))
		var userListResp vo.UserListResp
		userListResp.List = users
		userListResp.TotalCount = int(totalCount)
		model.Success(c, userListResp)
	}
}
