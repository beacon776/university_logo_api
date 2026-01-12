package service

import (
	"context"
	"fmt"
	goredis "github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"logo_api/dao/mysql"
	"logo_api/dao/redis"
	"logo_api/model"
	"logo_api/model/resource/do"
	"logo_api/model/resource/dto"
	"logo_api/model/resource/vo"
	"logo_api/settings"
	"logo_api/util"
	"net/url"
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
		fullCosPath := fmt.Sprintf("beacon/downloads/%s/%s", info.ShortName, info.ResourceName) // 这里应该进行 ResourceName 的中文路径转换！
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
		err = redis.AddPendingDelete(ctx, fmt.Sprintf("beacon/downloads/%s/%s", info.ShortName, info.ResourceName), expireAt) // 这里应该进行 ResourceName 的中文路径转换！
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
func InsertResource(ctx context.Context, req dto.ResourceInsertReq) error {
	// 1. 初始化 COS 客户端
	cosClient, err := util.NewClient(settings.Config.CosConfig)
	if err != nil {
		zap.L().Error("util.NewClient() failed", zap.Error(err))
		return err
	}
	// 2. 上传对象到 COS
	uploadCosPath := fmt.Sprintf("beacon/downloads/%s/%s", req.ShortName, req.Name)
	err = cosClient.UploadObject(ctx, req.File, uploadCosPath)
	if err != nil {
		return err
	}
	// 3. 转换为 Entity
	doResource, err := req.ToEntity()
	if err != nil {
		return err
	}
	// 4. 调用 DAO 插入数据 (包含原有的 University 统计更新)
	doResources := []*do.Resource{doResource}
	// Service 层回滚逻辑
	if err = mysql.InsertResource(doResources); err != nil {
		zap.L().Error("mysql.InsertResource() failed", zap.Error(err))
		// 删除刚刚上传到 COS 的文件，保持一致性
		// 使用 Background 确保删除请求不受父级 Context 取消的影响
		if delErr := cosClient.DeleteObject(context.Background(), uploadCosPath); err != nil {
			zap.L().Error("cosClient.DeleteObject() failed during rollback", zap.Error(delErr))
		}
		return err
	}
	return nil
}

func GetResourceList(req dto.ResourceGetListReq) ([]vo.ResourceResp, error) {
	var (
		doResourceList []do.Resource
		voResourceList []vo.ResourceResp
		err            error
	)
	if doResourceList, err = mysql.GetResourceList(req); err != nil {
		zap.L().Error("mysql.GetResourceList() failed", zap.Error(err))
		return nil, err
	}
	for _, resource := range doResourceList {
		var voResource vo.ResourceResp
		voResource = doResourceToVo(resource)
		voResourceList = append(voResourceList, voResource)
	}

	zap.L().Info("GetResourceList() success", zap.Int("success count", len(voResourceList)))
	return voResourceList, nil
}

func GetResources(names []string) ([]vo.ResourceResp, error) {
	var (
		doResources []do.Resource
		voResources []vo.ResourceResp
		err         error
	)
	if doResources, err = mysql.GetResources(names); err != nil {
		zap.L().Error("mysql.GerResources() failed", zap.Error(err))
		return nil, err
	}
	for _, resource := range doResources {
		var voResource vo.ResourceResp
		voResource = doResourceToVo(resource)
		voResources = append(voResources, voResource)
	}
	return voResources, nil
}

func DelResources(names []string) error {
	if err := mysql.DelResources(names); err != nil {
		zap.L().Error("mysql.DelResources() failed", zap.Strings("names", names), zap.Error(err))
		return err
	}
	zap.L().Info("DelResources() success", zap.Strings("names", names))
	return nil
}

func RecoverResources(names []string) error {
	if err := mysql.RecoverResources(names); err != nil {
		zap.L().Error("mysql.RecoverResources() failed", zap.Strings("names", names), zap.Error(err))
		return err
	}
	zap.L().Info("RecoverResources() success", zap.Strings("names", names))
	return nil
}

func doResourceToVo(resource do.Resource) vo.ResourceResp {
	var voResource vo.ResourceResp
	voResource.ID = resource.ID
	voResource.Title = resource.Title
	voResource.ShortName = resource.ShortName
	voResource.Name = resource.Name
	voResource.Type = resource.Type
	voResource.Md5 = resource.Md5
	voResource.Size = resource.Size
	updateTimeStr := ""
	if resource.LastUpdateTime != nil {
		// 格式化为：2026-01-07 00:52:17
		updateTimeStr = resource.LastUpdateTime.Format("2006-01-02 15:04:05") // RFC3339 转格式(示例："lastUpdateTime": "2025-07-04T08:00:00+08:00")
		voResource.LastUpdateTime = updateTimeStr
	}
	voResource.IsVector = resource.IsVector
	voResource.IsBitmap = resource.IsBitmap
	voResource.Width = resource.Width
	voResource.Height = resource.Height
	voResource.UsedForEdge = resource.UsedForEdge
	voResource.IsDeleted = resource.IsDeleted
	voResource.BackgroundColor = resource.BackgroundColor
	voResource.CosURL = fmt.Sprintf("%s/%s/%s", model.BeaconCosPreURL, resource.ShortName, url.PathEscape(resource.Name))
	return voResource
}
