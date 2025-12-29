package service

import (
	"context"
	goredis "github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"logo_api/dao/redis"
	"logo_api/model/user/dto"
	"net/url"
	"time"
)

// CleanExpiredCOSObjects 清理过期的 COS 对象 以及 Redis 对象
func (svc *ResourceService) CleanExpiredCOSObjects(ctx context.Context) (*dto.CleanResultDTO, error) {
	encodedPaths, err := redis.GetExpiredPendingDeletePaths(ctx, time.Now())
	if err != nil {
		return nil, err
	}
	result := &dto.CleanResultDTO{
		Total: len(encodedPaths),
	}

	if len(encodedPaths) == 0 {
		zap.L().Info("No expired COS objects to clean")
		return result, nil
	}

	for _, encodedPath := range encodedPaths {
		// 对 ZSET 取出的 ENCODED 路径进行 DECODE
		cosPath, decodeErr := url.QueryUnescape(encodedPath)
		if decodeErr != nil {
			zap.L().Error("Failed to unescape COS path, skipping deletion", zap.String("encodedPath", encodedPath), zap.Error(decodeErr))
			// 无法解码，认为路径损坏，但 ZSET 成员仍然需要移除
			_ = redis.RemovePendingDeletePaths(ctx, encodedPath)
			result.FailCount++
			continue
		}

		// 1. 从 Key 2 (反向映射) 中查找对应的 cacheKey (用于清理 Key 1)
		cacheKey, err := redis.GetReverseMapping(ctx, cosPath)
		if err != nil && err != goredis.Nil {
			zap.L().Error("Failed to get cacheKey from ReverseMapping (Key 2)", zap.String("path", cosPath), zap.Error(err))
			// 即使查询失败也继续清理 COS，防止存储泄漏
		}

		// 2. 从腾讯云COS进行删除
		// 传入 COS DeleteObject 的必须是 DECODED 原始路径
		err = svc.CosClient.DeleteObject(cosPath)
		if err != nil {
			// COS 删除失败：不从 ZSET 和 Key 2 中移除，等待下一次重试
			zap.L().Error("Failed to delete COS object", zap.String("path", cosPath), zap.Error(err))
			result.FailCount++
			result.FailedPaths = append(result.FailedPaths, cosPath)
			continue
		}
		// 3. COS 删除成功，执行 Redis 缓存清理
		// 3a. 删除 Key 1: hash -> cosPath (如果 Key 2 查到了 Key)
		if cacheKey != "" && err != goredis.Nil {
			if err = redis.DeleteCacheMapping(ctx, cacheKey); err != nil {
				zap.L().Warn("Failed to delete CacheMapping (Key 1)", zap.String("key", cacheKey), zap.Error(err))
			}
		}

		// 3b. 删除 Key 2: cosPath -> hash
		if err = redis.DeleteReverseMapping(ctx, cosPath); err != nil {
			zap.L().Warn("Failed to delete ReverseMapping (Key 2)", zap.String("path", cosPath), zap.Error(err))
		}
		// 3c. 从 ZSET 移除
		// ZSET 移除使用 ENCODED 路径（Redis 中存储的原始成员）
		err = redis.RemovePendingDeletePaths(ctx, encodedPath)
		if err != nil {
			zap.L().Error("Failed to remove path from pending delete", zap.String("path", cosPath), zap.Error(err))
		}
		// 这里的日志应该显示 DECODED 路径，更友好
		zap.L().Info("Deleted expired COS object", zap.String("path", cosPath))
		result.SuccessCount++
	}
	return result, nil
}
