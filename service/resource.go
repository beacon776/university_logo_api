package service

import (
	"context"
	"fmt"
	goredis "github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"logo_api/dao/mysql"
	"logo_api/dao/redis"
	"logo_api/model/user/do"
	"logo_api/model/user/dto"
	"logo_api/settings"
	"logo_api/util"
	"strings"
	"time"
)

type ResourceService struct {
	CosClient *util.CosClient
}

func NewResourceService(cosClient *util.CosClient) *ResourceService {
	return &ResourceService{CosClient: cosClient}
}

// GetLogo 获取logo文件二进制数据、相关字段数据
func (svc *ResourceService) GetLogo(req dto.ResourceGetLogoReq) ([]byte, string, string, error) {
	ext := req.Type
	preName := req.Name // 英文缩写 / 中文全称
	size := req.Size
	width := req.Width
	height := req.Height
	bgColor := req.BgColor
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
				data, err := svc.CosClient.GetObjectByResourceName(resourceName, shortName)
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
		data, info, err := svc.CosClient.GetObjectByResourceNameAndSvgToBitmap(
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
	data, err := svc.CosClient.GetObjectByResourceName(resource.ResourceName, resource.ShortName)
	if err != nil {
		zap.L().Error("CosClient.GetObjectByResourceName() failed", zap.Error(err))
		return nil, ext, "", err
	}
	return data, ext, resource.ResourceName, nil

}

func GetResourceByName(name string) (do.Resource, error) {
	var (
		daoUniversity do.Resource
		err           error
	)
	daoUniversity, err = mysql.GetResourceByName(name)
	if err != nil {
		zap.L().Error("mysql.GetResourceByName() failed", zap.Error(err))
		return do.Resource{}, err
	}
	zap.L().Info("GetResourceByName() success", zap.String("name", name))
	return daoUniversity, nil
}

// InsertResource 插入资源. 不需要插入Redis缓存，缓存只给转换后的图片使用
func InsertResource(resource []settings.UniversityResources) error {
	// 先插 university_resources 表的数据，然后查找
	if err := mysql.InsertUniversityResource(resource); err != nil {
		zap.L().Error("mysql.InsertUniversityResource() failed", zap.Error(err))
		return err
	}
	zap.L().Info("InsertUniversityResource() success", zap.Any("resource", resource))
	return nil
}

func UpdateResource(resource settings.UniversityResources) error {
	if err := mysql.UpdateUniversityResource(resource); err != nil {
		zap.L().Error("mysql.UpdateUniversityResource() failed", zap.Error(err))
		return err
	}
	zap.L().Info("UpdateUniversityResource() success", zap.Any("resource", resource))
	return nil
}
