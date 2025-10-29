package service

import (
	"context"
	"crypto/sha256" // 新增导入
	"encoding/hex"  // 新增导入
	"fmt"
	goredis "github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"logo_api/dao/mysql"
	"logo_api/dao/redis"
	"logo_api/settings"
	"logo_api/util"
	"net/url"
	"path/filepath"
	"strings"
	"time"
)

// 定义一个结果结构体
type CleanResult struct {
	Total        int      `json:"total"`
	SuccessCount int      `json:"success_count"`
	FailCount    int      `json:"fail_count"`
	FailedPaths  []string `json:"failed_paths"`
}
type ResourceService struct {
	CosClient *util.CosClient
}

func NewResourceService(cosClient *util.CosClient) *ResourceService {
	return &ResourceService{CosClient: cosClient}
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

// GetLogo 获取logo文件二进制数据、相关字段数据
func (s *ResourceService) GetLogo(fullName, bgColor string, size, width, height int) ([]byte, string, string, error) {
	ext := filepath.Ext(fullName)[1:]              // 去掉点
	preName := fullName[:len(fullName)-len(ext)-1] // 英文缩写 / 中文全称

	ctx := context.Background()
	// 1. 缓存查找 (仅对位图进行缓存查找)
	if ext != "svg" {
		cacheKey := generateCacheKey(preName, ext, bgColor, size, width, height)
		cosPath, err := redis.GetCacheMapping(ctx, cacheKey)
		if err == nil && cosPath != "" {
			// 缓存命中 (Key 1命中): 尝试从 COS 获取文件
			parts := strings.Split(cosPath, "/")
			if len(parts) >= 3 {
				shortName := parts[1]
				resourceName := parts[2]
				data, err := s.CosClient.GetObjectByResourceName(resourceName, shortName)
				if err == nil {
					zap.L().Info("Cache Hit - Serving from COS via Redis mapping", zap.String("key", cacheKey))
					return data, ext, resourceName, nil
				}
				// COS 文件获取失败，可能已被清理，删除脏缓存，继续执行生成逻辑
				zap.L().Warn("Cache Miss - COS object retrieval failed, deleting stale mapping", zap.String("path", cosPath), zap.Error(err))
			}
			// 确保删除 Key 1，避免下次继续查到这个错误的路径
			_ = redis.DeleteCacheMapping(ctx, cacheKey)
		} else if err != goredis.Nil {
			zap.L().Error("Redis GetCacheMapping failed", zap.Error(err))
		}
	}
	// 2. 缓存未命中或请求 SVG，执行数据库查找和文件生成逻辑
	// 先查 COS 缓存，如果不存在就生成
	var resource settings.UniversityResources
	var err error

	// 先找出来需要计算的主文件
	if ext == "svg" {
		resource, err = mysql.QueryFromNameAndSvg(preName, ext)
		if err != nil {
			zap.L().Error("Could not find source SVG file for conversion", zap.String("name", preName), zap.Error(err))
			return nil, ext, "", err
		}
	} else {
		resource, err = mysql.QueryFromNameAndBitmapInfo(preName, ext, size, width, height, bgColor)
	}
	if err != nil {
		zap.L().Error("mysql.Query() failed", zap.Error(err))
		return nil, ext, "", err
	}

	// 如果是 svg 转出来的位图，说明缓存没有生效
	if ext != "svg" && resource.ResourceType == "svg" {
		data, info, err := s.CosClient.GetObjectByResourceNameAndSvgToBitmap(
			resource.ResourceName, resource.Title, resource.ShortName,
			ext, size, width, height, bgColor,
		)
		if err != nil {
			zap.L().Error("CosClient.GetObjectByResourceNameAndSvgToBitmap() failed", zap.Error(err))
			return nil, ext, "", err
		}
		// 4. 转换成功，执行三层缓存写入
		fullCosPath := fmt.Sprintf("beacon/downloads/%s/%s", info.ShortName, info.ResourceName)
		cacheKey := generateCacheKey(preName, ext, bgColor, size, width, height)
		//ttl := time.Hour // 缓存过期时间
		localTtl := time.Second * 20
		// 4a. 写入 Key 1: hash -> cosPath (查询映射)
		if err = redis.SetCacheMapping(ctx, cacheKey, fullCosPath); err != nil {
			zap.L().Warn("redis.SetCacheMapping() failed", zap.Error(err))
		}

		// 4b. 写入 Key 2: cosPath -> hash (反向映射)
		if err = redis.SetReverseMapping(ctx, fullCosPath, cacheKey); err != nil {
			zap.L().Warn("redis.SetReverseMapping() failed", zap.Error(err))
		}

		// 4c. 写入 ZSET: cosPath -> expireTime (定时清理)
		expireAt := time.Now().Add(localTtl)
		err = redis.AddPendingDelete(ctx, fmt.Sprintf("beacon/downloads/%s/%s", info.ShortName, info.ResourceName), expireAt)
		if err != nil {
			zap.L().Warn("redis.AddPendingDelete() failed", zap.Error(err))
		}

		return data, ext, info.ResourceName, nil
	}
	// 可以直接获取到这张图片
	data, err := s.CosClient.GetObjectByResourceName(resource.ResourceName, resource.ShortName)
	if err != nil {
		zap.L().Error("CosClient.GetObjectByResourceName() failed", zap.Error(err))
		return nil, ext, "", err
	}
	return data, ext, resource.ResourceName, nil

}

// CleanExpiredCOSObjects 清理过期的 COS 对象 以及 Redis 对象
func (s *ResourceService) CleanExpiredCOSObjects(ctx context.Context) (*CleanResult, error) {
	encodedPaths, err := redis.GetExpiredPendingDeletePaths(ctx, time.Now())
	if err != nil {
		return nil, err
	}
	result := &CleanResult{
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
		err = s.CosClient.DeleteObject(cosPath)
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

// InsertResource 插入资源. 不需要插入Redis缓存，缓存只给转换后的图片使用
func (s *ResourceService) InsertResource(resource []settings.UniversityResources) error {
	// 先插 university_resources 表的数据，然后查找
	if err := mysql.InsertUniversityResource(resource); err != nil {
		zap.L().Error("mysql.InsertUniversityResource() failed", zap.Error(err))
		return err
	}
	zap.L().Info("InsertUniversityResource() success", zap.Any("resource", resource))
	return nil
}

func (s *ResourceService) UpdateResource(resource settings.UniversityResources) error {
	if err := mysql.UpdateUniversityResource(resource); err != nil {
		zap.L().Error("mysql.UpdateUniversityResource() failed", zap.Error(err))
		return err
	}
	zap.L().Info("UpdateUniversityResource() success", zap.Any("resource", resource))
	return nil
}
