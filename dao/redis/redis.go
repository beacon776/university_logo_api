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

// AddPendingDelete 把 COS 文件路径加入待删除集合（ZSET）
func AddPendingDelete(ctx context.Context, cosPath string, expireAt time.Time) error {
	score := float64(expireAt.Unix())
	// 编码 URL（UTF-8 下自动）
	encodedPath := url.QueryEscape(cosPath)
	return rdb.ZAdd(ctx, "cos_pending_delete", redis.Z{
		Score:  score,
		Member: encodedPath,
	}).Err()
}

// GetExpiredPendingDeletePaths 返回所有已经过期的待删除路径（自动解码）
func GetExpiredPendingDeletePaths(ctx context.Context, now time.Time) ([]string, error) {
	score := float64(now.Unix())
	encodedPaths, err := rdb.ZRangeByScore(ctx, "cos_pending_delete", &redis.ZRangeBy{
		Min: "-inf",
		Max: strconv.FormatFloat(score, 'f', 0, 64),
	}).Result()
	if err != nil {
		zap.L().Error("rdb.ZRangeByScore() failed", zap.Error(err))
		return nil, err
	}
	// 使用UTF-8进行解码，防止中文出错
	decodedPaths := make([]string, len(encodedPaths))
	for i, ep := range encodedPaths {
		if decoded, err := url.QueryUnescape(ep); err == nil {
			decodedPaths[i] = decoded
		} else {
			decodedPaths[i] = ep // 解码失败就原样返回
		}
	}
	return decodedPaths, nil
}

// RemovePendingDeletePaths 从待删除集合中移除指定的路径
func RemovePendingDeletePaths(ctx context.Context, paths ...string) error {
	if len(paths) == 0 {
		return nil
	}
	interfaceSlice := make([]interface{}, len(paths))
	for i, p := range paths {
		interfaceSlice[i] = url.QueryEscape(p) // 编码后再删
	}
	return rdb.ZRem(ctx, "cos_pending_delete", interfaceSlice...).Err()
}
