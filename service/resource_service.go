package service

import (
	"database/sql"
	"go.uber.org/zap"
	"logo_api/dao/mysql"
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

// GetLogo 获取logo
func (s *ResourceService) GetLogo(fullName, bgColor string, size, width, height int) ([]byte, string, error) {
	ext := filepath.Ext(fullName)[1:] // 去掉点
	preName := fullName[:len(fullName)-len(ext)-1]

	var resource settings.UniversityResources
	var err error

	if ext == "svg" {
		resource, err = mysql.QueryFromNameAndSvg(preName, ext)
	} else {
		resource, err = mysql.QueryFromNameAndBitmap(preName, ext, size, width, height)
	}
	if err != nil {
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

		// 构造 DB 记录并插入（幂等处理）
		now := time.Now().UTC()
		rec := settings.UniversityResources{
			ShortName:    resource.ShortName,
			Title:        resource.Title,
			ResourceName: info.ResourceName,
			ResourceType: ext,
			ResourceMd5:  info.ResourceMd5,
			ResourceSizeB: sql.NullInt64{
				Int64: info.ResourceSizeB,
				Valid: true,
			},
			ResolutionWidth: sql.NullInt64{
				Int64: info.ResolutionWidth,
				Valid: true,
			},
			ResolutionHeight: sql.NullInt64{
				Int64: info.ResolutionHeight,
				Valid: true,
			},
			LastUpdateTime:  sql.NullTime{Time: now, Valid: true},
			UploadTime:      sql.NullTime{Time: now, Valid: true},
			ExpireTime:      sql.NullTime{Time: now.Add(time.Hour), Valid: true},
			IsVector:        false,
			IsBitmap:        true,
			UsedForEdge:     false,
			IsDeleted:       false,
			BackgroundColor: bgColor,
		}

		if err = mysql.InsertUniversityResource(rec); err != nil {
			// 插入失败不要直接让请求失败（可能是重复 key），记录 warn 并继续返回图片数据
			zap.L().Warn("InsertUniversityResource failed, maybe duplicate", zap.Error(err))
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

/*
// GenerateBitmapAndSave 生成转换后 Bitmap 文件字段数据，并保存到 mysql（保存到腾讯云COS已经在cos_util里做了）
func GenerateBitmapAndSave(resourceName, title, shortName, resourceType string, size, width, height int, bgColor string) error {
	cosClient, err := util.NewClient(settings.Config.CosConfig)
	if err != nil {
		zap.L().Error("util.NewClient() failed", zap.Error(err))
		return err
	}
	data, info, err := cosClient.GetObjectByResourceNameAndSvgToBitmap(resourceName, title, shortName, resourceType, size, width, height, bgColor)

	// 构造数据库对象
	now := time.Now().UTC()
	record := settings.UniversityResources{
		ResourceName: resourceName,
		Title:        title,
		ShortName:    shortName,
		ResourceType: resourceType,
		ResourceMd5:  info.ResourceMd5,
		ResourceSizeB: sql.NullInt64{
			Int64: info.ResourceSizeB,
			Valid: true,
		},
		ResolutionWidth: sql.NullInt64{
			Int64: info.ResolutionWidth,
			Valid: true,
		},
		ResolutionHeight: sql.NullInt64{
			Int64: info.ResolutionHeight,
			Valid: true,
		},
		LastUpdateTime: sql.NullTime{
			Time:  now,
			Valid: true,
		},
		UploadTime: sql.NullTime{
			Time:  now,
			Valid: true,
		},
		ExpireTime: sql.NullTime{
			Time:  now.Add(time.Hour),
			Valid: true,
		},
		IsVector:    false,
		IsBitmap:    true,
		UsedForEdge: false,
		IsDeleted:   false,
		BackgroundColor: bgColor,
	}
	// 数据库插入
	if err = mysql.InsertUniversityResource(record); err != nil {
		zap.L().Error("mysql.InsertUniversityResource() failed", zap.Error(err))
		return err
	}

	// data 可用于进一步处理，比如返回给 API
	_ = data
	return nil
}
*/

// ClearExpiredCache 由 service 调用，负责业务流程协调
func (s *ResourceService) ClearExpiredCache(expireDuration time.Duration) {
	expiredResources, err := mysql.QueryExpiredResources(expireDuration)
	if err != nil {
		zap.L().Error("query expired resources failed", zap.Error(err))
		return
	}

	for _, res := range expiredResources {
		cosPath := "beacon/" + res.ShortName + "/" + res.ResourceName

		// 删除 cos 上资源
		if err = s.CosClient.DeleteObject(cosPath); err != nil {
			zap.L().Error("delete cos object failed", zap.String("path", cosPath), zap.Error(err))
			continue
		}

		// 删除标记数据库资源
		if err = mysql.MarkResourceDeleted(res.Id); err != nil {
			zap.L().Error("mark resource deleted failed", zap.Int("id", res.Id), zap.Error(err))
		}
	}
}
