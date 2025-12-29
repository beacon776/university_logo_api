package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	goredis "github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"logo_api/dao/redis"
	"logo_api/util"
	"strings"
	"time"
)

// UserLogout 处理用户的登出逻辑，包括 SSO 撤销和黑名单操作。
func UserLogout(ctx context.Context, userID int, tokenString string, expirationTime time.Time) error {

	// 1. 撤销 SSO Session：从 Redis 中删除该用户的 Session Token
	err := redis.DeleteUserSessionToken(ctx, userID)
	if err != nil {
		zap.L().Error("UserLogout: Failed to delete user session token from Redis (SSO)",
			zap.Int("userID", userID), zap.Error(err))
		// 尽管失败，我们仍然尝试执行黑名单操作，保证 Token 安全失效。
	}

	// 2. 将当前的 JWT 加入黑名单 (仅当 Token 还没过期时)
	durationUntilExpiry := time.Until(expirationTime)

	if durationUntilExpiry > 0 {
		err = redis.BlacklistToken(ctx, tokenString, durationUntilExpiry)
		if err != nil {
			zap.L().Error("UserLogout: Failed to blacklist token",
				zap.String("token", tokenString),
				zap.Int("userID", userID),
				zap.Error(err))
			// 黑名单操作失败是严重的，可能需要返回错误。
			return err
		}
	}

	zap.L().Info("UserLogout successful", zap.Int("userID", userID))
	return nil
}

// GetUserSessionToken (Service 层实现)
func GetUserSessionToken(ctx context.Context, userID int) (string, error) {
	token, err := redis.GetUserSessionToken(ctx, userID)

	// 捕获 go-redis 的 key 不存在错误
	if errors.Is(err, goredis.Nil) {
		// 将低层的 redis.Nil 转换为 Service 层的 ErrSessionNotFound
		return "", ErrSessionNotFound
	}

	// 返回其他 Redis 错误或 nil
	return token, err
}

// IsTokenBlacklisted (Service 层实现)
func IsTokenBlacklisted(ctx context.Context, tokenString string) (bool, error) {
	// 调用 DAO 层检查黑名单
	return redis.IsTokenBlacklisted(ctx, tokenString)
	// Note: IsTokenBlacklisted 在 DAO 层已经使用了 EXISTS 命令，
	// EXISTS 命令返回的是 bool/int，不会返回 redis.Nil 错误。
	// 因此这里不需要额外的错误转换。
}

// generateCacheKey 使用 SHA-256 对所有影响图片生成的参数进行哈希，以生成唯一的缓存 Key
func generateCacheKey(preName, ext, bgColor string, size, width, height int) string {
	normalizedBgColor := util.NormalizeColor(bgColor)
	// 如果颜色规范化失败，可以决定使用一个固定的默认颜色值（例如 "NIL"）来保持哈希稳定
	if normalizedBgColor == "" {
		normalizedBgColor = "NIL"
	}
	// 1. 将所有参数拼接成一个唯一的输入字符串
	// 确保 size, width, height 至少有一个是有效值，否则用 0 代替，以保证哈希一致性
	input := fmt.Sprintf("name:%s|ext:%s|bg:%s|size:%d|w:%d|h:%d",
		preName,
		ext,
		// 清理掉 # 符号，防止在某些系统中引起歧义
		strings.ReplaceAll(bgColor, "#", ""),
		size,
		width,
		height)

	// 2. 使用 SHA-256 对输入字符串进行哈希计算
	hasher := sha256.New()
	hasher.Write([]byte(input))

	// 3. 将哈希结果编码为十六进制字符串
	// 返回一个固定长度 64 字符的唯一 Key
	return hex.EncodeToString(hasher.Sum(nil))
}

// StoreUserToken 是 Service 层的方法，用于将用户的当前有效 Token 存储到 Redis
// 实现了单点登录的业务逻辑。
// 它调用了 redis DAO 层中的 SetUserSessionToken 函数。
func StoreUserToken(ctx context.Context, userID int, token string, duration time.Duration) error {
	// 直接调用 DAO 层的封装函数
	return redis.SetUserSessionToken(ctx, userID, token, duration)
}
