package auth

import (
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"
	"logo_api/model"
	"logo_api/service"
	"time"
)

// 密钥是一个私有变量
var jwtSecret []byte

// MinSecretLength 推荐的最小长度 (例如，至少 32 字节 for HS256/256 bits)
const MinSecretLength = 32

// InitJWTSecret 供 main 函数调用，用于设置密钥
func InitJWTSecret(secret string) error {
	if len(secret) < MinSecretLength {
		return fmt.Errorf("JWT secret length is %d bytes, but minimum required is %d bytes",
			len(secret), MinSecretLength)
	}

	// 检查是否为空字符串
	if secret == "" {
		return errors.New("JWT secret cannot be empty")
	}

	jwtSecret = []byte(secret)
	return nil
}

type UserClaims struct {
	UserID   int    `json:"user_id"`
	Username string `json:"username"`
	jwt.RegisteredClaims
}

// CreateToken 根据用户信息生成一个新的 JWT 字符串
func CreateToken(userID int, username string) (string, error) {
	// 设置 token 的过期时间：例如 24 小时
	expirationTime := time.Now().Add(24 * time.Hour)

	// 创建 Claims (Payload)
	claims := &UserClaims{
		UserID:   userID,
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime), // 设置过期时间
			IssuedAt:  jwt.NewNumericDate(time.Now()),     // 设置签发时间
			Issuer:    "logo-api-server",                  // 签发者
		},
	}

	// 使用我们定义的 Claims 和 HS256 签名方法创建一个新的 Token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// 使用密钥签名 Token，并生成最终的字符串
	tokenString, err := token.SignedString(jwtSecret)
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

// -------------------------------------------------------------
// 2. 检查 Token (验证)
// -------------------------------------------------------------

// CheckToken 验证传入的 Authorization Header 中的 Token 是否有效
// 它期望的格式是 "Bearer <token>"
func CheckToken(authHeader string) (*UserClaims, error) {
	// 检查 Header 是否是 "Bearer <token>" 格式
	const prefix = "Bearer "
	if len(authHeader) < len(prefix) || authHeader[:len(prefix)] != prefix {
		return nil, errors.New("invalid authorization header format")
	}

	tokenString := authHeader[len(prefix):]

	// 解析 Token
	token, err := jwt.ParseWithClaims(tokenString, &UserClaims{}, func(token *jwt.Token) (interface{}, error) {
		// 确保签名方法是 HMAC
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return jwtSecret, nil // 返回签名密钥
	})

	// 任何验证失败，包括过期 (exp claim 检查)，都会导致 err != nil
	if err != nil {
		// 解析失败，可能是过期、签名错误或格式错误
		return nil, err
	}

	// 检查 Token 是否有效，并断言 Claims 类型
	if claims, ok := token.Claims.(*UserClaims); ok && token.Valid {
		// jwt.ParseWithClaims 在解析时自动检查了 exp 和 nbf (如果存在)
		// token.Valid 也会检查 exp, iat, nbf 等时间相关的 Claims。
		return claims, nil
	}

	return nil, errors.New("invalid or expired token")
}

// GetTokenExpiration 从 Token 字符串中解析出过期时间
func GetTokenExpiration(tokenString string) (time.Time, error) {
	claims := &UserClaims{}

	// 使用密钥解析 Token，只关心 Claims
	_, _, err := new(jwt.Parser).ParseUnverified(tokenString, claims)
	if err != nil {
		// 如果是解析错误，返回
		return time.Time{}, err
	}

	// 检查 claims 是否包含 RegisteredClaims
	if claims.ExpiresAt == nil {
		return time.Time{}, errors.New("token must have an expiration time (exp claim)")
	}

	return claims.ExpiresAt.Time, nil
}

// -------------------------------------------------------------
// 3. 认证中间件 (整合 SSO/黑名单检查)
// -------------------------------------------------------------

// AuthRequired 认证中间件 (接受 Service 实例)
func AuthRequired(svc *service.ResourceService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从 Header 中获取完整的 Authorization 字符串
		authHeader := c.GetHeader("Authorization")

		// 1. 检查 JWT 签名和有效期
		claims, err := CheckToken(authHeader) // CheckToken 已经完成了签名和有效期检查
		if err != nil || claims == nil {
			model.Error(c, model.CodeUnauthorized, "Unauthorized: Invalid or expired token. "+err.Error())
			c.Abort()
			return
		}

		// -----------------------------------------------------
		// 2. SSO / Blacklist 状态校验 (重点新增)
		// -----------------------------------------------------

		tokenString := authHeader[len("Bearer "):] // 提取裸 Token 字符串

		// 2.1 检查 Token 是否在黑名单中 (用于登出/强制撤销)
		isBlacklisted, err := service.IsTokenBlacklisted(c.Request.Context(), tokenString)
		if err != nil {
			// Redis 查询错误
			zap.L().Error("AuthRequired: Blacklist check failed",
				zap.String("token", tokenString),
				zap.Int("userID", claims.UserID),
				zap.Error(err))
			model.Error(c, model.CodeServerErr, "Server error during token verification.")
			c.Abort()
			return
		}
		if isBlacklisted {
			zap.L().Warn("AuthRequired: Token is blacklisted",
				zap.String("token", tokenString),
				zap.Int("userID", claims.UserID))
			model.Error(c, model.CodeUnauthorized, "Token has been revoked.")
			c.Abort()
			return
		}

		// 2.2 检查 SSO (单点登录) 状态
		// 只有当客户端 Token == Redis 中存储的 Token 时，才有效
		redisToken, err := service.GetUserSessionToken(c.Request.Context(), claims.UserID)
		if err != nil && !errors.Is(err, service.ErrSessionNotFound) {
			// 匹配到数据库或 Redis 查询错误 (非 Key 不存在，而是连接或I/O错误)
			zap.L().Error("AuthRequired: SSO check failed due to server error",
				zap.Int("userID", claims.UserID),
				zap.Error(err))
			model.Error(c, model.CodeServerErr, "Server error during session check.")
			c.Abort()
			return
		}

		// 检查 Session 失效的条件：
		// 1. err 是 service.ErrSessionNotFound (Session不存在/已过期)
		// 2. 或者 Redis 中存储的 Token 不匹配请求中的 Token (被新登录覆盖)
		if errors.Is(err, service.ErrSessionNotFound) || redisToken != tokenString {
			zap.L().Info("AuthRequired: Session unauthorized or overwritten",
				zap.Int("userID", claims.UserID),
				zap.String("requestToken", tokenString),
				zap.String("redisToken", redisToken))
			model.Error(c, model.CodeUnauthorized, "Session expired or overwritten by a new login.")
			c.Abort()
			return
		}

		// -----------------------------------------------------

		// 3. 校验通过，设置 Context 并继续
		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("user_claims", claims)
		c.Set("tokenString", tokenString) // 存储 Token 字符串，方便 Logout 接口使用
		zap.L().Info("AuthRequired: Success", zap.Int("userID", claims.UserID))
		c.Next()
	}
}
