package service

import (
	"context"
	"fmt"
	"go.uber.org/zap"
	"logo_api/dao/mysql"
	"logo_api/dao/redis"
	"logo_api/settings"
	"logo_api/util"
	"path/filepath"
	"time"
)

type ResourceService struct {
	CosClient *util.CosClient
}

func NewResourceService(cosClient *util.CosClient) *ResourceService {
	return &ResourceService{CosClient: cosClient}
}

// GetLogo 获取logo文件二进制数据、相关字段数据
func (s *ResourceService) GetLogo(fullName, bgColor string, size, width, height int) ([]byte, string, error) {
	ext := filepath.Ext(fullName)[1:] // 去掉点
	preName := fullName[:len(fullName)-len(ext)-1]

	ctx := context.Background()
	// 先查 Redis 缓存
	cosPath, found, err := redis.GetLogoCache(ctx, preName, ext, size, width, height, bgColor)
	if err != nil {
		zap.L().Error("redis.GetLogoCache() failed", zap.Error(err))
		return nil, "", err
	}

	// 缓存命中，从 COS 读取文件数据返回
	if found {
		data, err := s.CosClient.GetObjectByResourceName(cosPath, preName)
		if err == nil {
			zap.L().Info("GetObjectByResourceName()", zap.String("cosPath", cosPath))
			return data, ext, nil
		}
	}

	// 缓存没有命中，走流程，并生成新缓存
	var resource settings.UniversityResources

	if ext == "svg" {
		resource, err = mysql.QueryFromNameAndSvg(preName, ext)
	} else {
		resource, err = mysql.QueryFromNameAndBitmapInfo(preName, ext, size, width, height, bgColor)
	}
	if err != nil {
		zap.L().Error("mysql.Query() failed", zap.Error(err))
		return nil, ext, err
	}

	// 如果是 svg 转出来的位图
	if ext != "svg" && resource.ResourceType == "svg" {
		data, info, err := s.CosClient.GetObjectByResourceNameAndSvgToBitmap(
			resource.ResourceName, resource.Title, resource.ShortName,
			ext, size, width, height, bgColor,
		)
		if err != nil {
			zap.L().Error("CosClient.GetObjectByResourceNameAndSvgToBitmap failed", zap.Error(err))
			return nil, ext, err
		}

		// 写 Redis 缓存，设置 TTL 1小时
		ttl := time.Hour
		err = redis.SetLogoCache(ctx, preName, ext, size, width, height, bgColor, info.ResourceName, info.ResourceMd5, ttl)
		//tempTTL := time.Minute * 1
		//err = redis.SetLogoCache(ctx, preName, ext, size, width, height, bgColor, fmt.Sprintf("beacon/%s/%s", info.ShortName, info.ResourceName), info.ResourceMd5, tempTTL)
		if err != nil {
			zap.L().Warn("SetLogoCache failed", zap.Error(err))
		}

		// 把 COS 路径和过期时间放入待删除集合
		expireAt := time.Now().Add(ttl)
		//expireAt := time.Now().Add(tempTTL)
		err = redis.AddPendingDelete(ctx, fmt.Sprintf("beacon/%s/%s", preName, info.ResourceName), expireAt)
		if err != nil {
			zap.L().Warn("AddPendingDelete failed", zap.Error(err))
		}

		return data, ext, nil
	} else { // 可以直接获取到这张图片
		data, err := s.CosClient.GetObjectByResourceName(resource.ResourceName, resource.ShortName)
		if err != nil {
			zap.L().Error("CosClient.GetObjectByResourceName failed", zap.Error(err))
			return nil, ext, err
		}
		return data, ext, nil
	}
}

func (s *ResourceService) CleanExpiredCOSObjects(ctx context.Context) error {
	paths, err := redis.GetExpiredPendingDeletePaths(ctx, time.Now())
	if err != nil {
		return err
	}

	for _, path := range paths {
		// 从腾讯云COS进行删除
		err = s.CosClient.DeleteObject(path)
		if err != nil {
			zap.L().Error("Failed to delete COS object", zap.String("path", path), zap.Error(err))
			continue
		}
		// 删除成功后，从 Redis 集合移除
		err = redis.RemovePendingDeletePaths(ctx, path)
		if err != nil {
			zap.L().Error("Failed to remove path from pending delete", zap.String("path", path), zap.Error(err))
		}
		zap.L().Info("Deleted expired COS object", zap.String("path", path))
	}
	return nil
}
