package redis

import (
	"context"
	"fmt"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"logo_api/settings"
	"net/url"
	"strconv"
	"time"
)

var rdb *redis.Client

const (
	CacheKeyPrefix    = "logo_cache:"        // key1: hash -> cosPath
	ReverseKeyPrefix  = "logo_cos_to_key:"   // key2: cosPath -> hash
	PendingDeleteZSET = "cos_pending_delete" // ZSET: cosPath -> expireTime
)

func Init(config *settings.RedisConfig) (err error) {
	rdb = redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", config.Host, config.Port),
		Password: config.Password,
		DB:       config.DB,
		PoolSize: config.PoolSize,
	})
	_, err = rdb.Ping(context.Background()).Result() // v9必须显式传入 context.Background
	return err
}

// key1: hash -> cosPath 缓存映射机制

// SetCacheMapping 设置 hash -> COS 路径 的缓存映射 (Key 1)
func SetCacheMapping(ctx context.Context, cacheKey, cosPath string) error {
	fullKey := CacheKeyPrefix + cacheKey
	// String 结构，无 TTL (永久存储，直到被清理)
	return rdb.Set(ctx, fullKey, cosPath, 0).Err()
}

// GetCacheMapping 根据 hash 查找 COS 路径 (Key 1)
func GetCacheMapping(ctx context.Context, cacheKey string) (string, error) {
	fullKey := CacheKeyPrefix + cacheKey
	return rdb.Get(ctx, fullKey).Result()
}

// DeleteCacheMapping 删除 hash -> COS 路径 的缓存映射 (Key 1)
func DeleteCacheMapping(ctx context.Context, cacheKey string) error {
	fullKey := CacheKeyPrefix + cacheKey
	return rdb.Del(ctx, fullKey).Err()
}

// key2: COS Path -> hash 反向映射

// SetReverseMapping 设置 COS 路径 -> hash 的反向映射 (Key 2)
func SetReverseMapping(ctx context.Context, cosPath, cacheKey string) error {
	encodedPath := url.QueryEscape(cosPath)
	fullKey := ReverseKeyPrefix + encodedPath
	// String 结构，无 TTL (永久存储，直到被清理)
	return rdb.Set(ctx, fullKey, cacheKey, 0).Err()
}

// GetReverseMapping 根据 COS 路径 查找 原始 hash (Key 2)
func GetReverseMapping(ctx context.Context, cosPath string) (string, error) {
	encodedPath := url.QueryEscape(cosPath)
	fullKey := ReverseKeyPrefix + encodedPath
	return rdb.Get(ctx, fullKey).Result()
}

// DeleteReverseMapping 删除 COS 路径 -> hash 的反向映射 (Key 2)
func DeleteReverseMapping(ctx context.Context, cosPath string) error {
	encodedPath := url.QueryEscape(cosPath)
	fullKey := ReverseKeyPrefix + encodedPath
	return rdb.Del(ctx, fullKey).Err()
}

// ZSET: COS 路径待删除集合 (现有逻辑，无需修改)

// AddPendingDelete 把 COS 文件路径加入待删除集合（ZSET）
func AddPendingDelete(ctx context.Context, cosPath string, expireAt time.Time) error {
	score := float64(expireAt.Unix())
	// 编码 URL（UTF-8 下自动）
	encodedPath := url.QueryEscape(cosPath)
	return rdb.ZAdd(ctx, PendingDeleteZSET, redis.Z{
		Score:  score,
		Member: encodedPath,
	}).Err()
}

// GetExpiredPendingDeletePaths 返回所有已经过期的待删除路径（修改：返回 ENCODED 路径）
func GetExpiredPendingDeletePaths(ctx context.Context, now time.Time) ([]string, error) {
	score := float64(now.Unix())
	encodedPaths, err := rdb.ZRangeByScore(ctx, PendingDeleteZSET, &redis.ZRangeBy{
		Min: "-inf",
		Max: strconv.FormatFloat(score, 'f', 0, 64),
	}).Result()
	if err != nil {
		zap.L().Error("rdb.ZRangeByScore() failed", zap.Error(err))
		return nil, err
	}
	// 直接返回 Redis 中存储的原始（已编码）成员
	return encodedPaths, nil
}

// RemovePendingDeletePaths 从待删除集合中移除指定的路径（现在接收 ENCODED 路径）
func RemovePendingDeletePaths(ctx context.Context, paths ...string) error {
	if len(paths) == 0 {
		return nil
	}
	interfaceSlice := make([]interface{}, len(paths))
	for i, p := range paths {
		// 不再需要重新编码，因为 paths 已经是编码后的 ZSET 成员
		interfaceSlice[i] = p
	}
	return rdb.ZRem(ctx, PendingDeleteZSET, interfaceSlice...).Err()
}
