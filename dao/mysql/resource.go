package mysql

import (
	"errors"
	"fmt"
	"go.uber.org/zap"
	"logo_api/model"
	"logo_api/model/resource/do"
	"logo_api/model/resource/dto"
	"logo_api/settings"
	"strings"

	"gorm.io/gorm"
)

// GORM API 要点:
// 1. Where() 替换 SQL WHERE 子句。
// 2. First() 替换 db.Get() 查单条。
// 3. Find() 替换 db.Select() 查多条。
// 4. Create() 替换 db.NamedExec() 批量插入。
// 5. Save() 或 Updates() 替换 UPDATE 语句。

// QueryFromNameAndSvg 在后缀是Svg的情况下进行查询
func QueryFromNameAndSvg(preName string, ext string) (settings.UniversityResources, error) {
	var resource settings.UniversityResources

	// GORM API 要点: 复合 WHERE 条件查询单条记录。
	// 使用 First() 查找，GORM 自动添加 LIMIT 1
	err := db.Table("resource").Where("(short_name = ? OR title = ?) AND resource_type = ? AND is_deleted = ?", preName, preName, ext, 0).First(&resource).Error

	// 查询出错
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) { // GORM 找不到记录时返回 gorm.ErrRecordNotFound
			zap.L().Error("resource not found")
			return resource, err // 返回 err 让上层知道是未找到
		}
		// 其他错误
		zap.L().Error("db.First() err:", zap.Error(err))
		return settings.UniversityResources{}, err
	}
	zap.L().Info("get svg resource:", zap.Any("resource", resource))
	return resource, nil
}

// QueryFromNameAndBitmapInfo 根据文件名、后缀（保证为位图）、边长/宽+高、背景颜色参数查找对应的资源
func QueryFromNameAndBitmapInfo(preName string, ext string, size int, width int, height int, bgColor string) (settings.UniversityResources, error) {
	var resource settings.UniversityResources

	// GORM API 要点: 复杂的 WHERE/OR 组合查询
	// 使用 Where() 包含所有的 AND 条件
	tx := db.Table("resource").Where("(short_name = ? OR title = ?) AND resource_type = ? AND is_deleted = 0 AND background_color = ? AND is_deleted = ?",
		preName, preName, ext, bgColor, 0)

	// 使用 Or() 组合宽度/高度的 OR 逻辑
	tx = tx.Where("(resolution_width = ? AND resolution_height = ?) OR (resolution_width = ? AND resolution_height = ? AND is_deleted = ?)",
		size, size, width, height, 0)

	err := tx.First(&resource).Error // 执行第一次查询

	// 第一次查询出错
	if err != nil {
		// 没查到
		if errors.Is(err, gorm.ErrRecordNotFound) {
			zap.L().Error("current resource not found, next will try to find svg resource")

			// 虽然直接查没查到，但是还有机会查到 svg 资源，继续去查 svg 资源
			// GORM 第二次查询: 查找用于 edge 的 SVG 资源
			err = db.Table("resource").Where("(short_name = ? OR title = ?) AND used_for_edge = ?", preName, preName, 1).First(&resource).Error

			// 第二次查询出错
			if err != nil {
				// svg 资源也没查到
				if errors.Is(err, gorm.ErrRecordNotFound) {
					zap.L().Error("svg resource was not founded as well, this bitmap resource could not be found")
					return settings.UniversityResources{}, errors.New("bitmap and svg resource not found") // 返回一个更明确的错误
				}
				// 其他错误
				zap.L().Error("db.First() err:", zap.Error(err))
				return settings.UniversityResources{}, err
			}
			// 查到了 svg 资源
			return resource, nil
		}
		// 其他错误
		zap.L().Error("db.First() failed", zap.Error(err))
		return settings.UniversityResources{}, err
	}

	// 直接查到了该资源
	zap.L().Info("get bitmap resource:", zap.Any("resource", resource))
	return resource, nil
}

// InsertResources 对 resource 表进行批量插入
func InsertResources(resources []*do.Resource) error {
	// GORM API 要点: 批量插入。
	// 对切片使用 db.Create()，GORM 自动处理字段映射和批量 INSERT
	if len(resources) == 0 {
		zap.L().Warn("mysql.InsertResources() Warn: resources is empty")
		return nil
	}
	// 1. 提取所有待插入资源的 MD5 用于初步筛选查询
	md5s := make([]string, 0, len(resources))
	for _, resource := range resources {
		md5s = append(md5s, resource.Md5)
	}

	// 开启事务
	return db.Transaction(func(tx *gorm.DB) error {
		// 2. 查重：从数据库找出 MD5 匹配且未删除的记录
		var existing []struct {
			Md5  string
			Size int
		}
		if err := tx.Table("resource").
			Where("md5 IN ? AND is_deleted = ?", md5s, model.ResourceIsActive).
			Select("md5, size").
			Find(&existing).Error; err != nil {
			zap.L().Error("mysql.tx.Find() err:", zap.Error(err))
			return err
		}
		// 3. 构建已存在资源的映射表 (Map)，Key 为 "md5_size"
		existMap := make(map[string]bool)
		for _, e := range existing {
			key := fmt.Sprintf("%s_%d", e.Md5, e.Size)
			existMap[key] = true
		}

		// 4. 过滤 resources，只保留数据库中不存在的记录
		var toInsert []*do.Resource
		shortNames := make(map[string]bool)
		for _, resource := range resources {
			key := fmt.Sprintf("%s_%d", resource.Md5, resource.Size)
			if !existMap[key] {
				toInsert = append(toInsert, resource)
				shortNames[resource.ShortName] = true
			} else {
				zap.L().Info("mysql.InsertResources() skip duplicate",
					zap.String("title", resource.Title),
					zap.String("md5", resource.Md5))
			}
		}
		// 5. 如果过滤后没有新资源，直接退出
		if len(toInsert) == 0 {
			zap.L().Info("mysql.InsertResources(): all resources are duplicates, nothing to insert")
			return nil
		}

		// 6. 执行批量插入 (使用过滤后的 toInsert)
		err := tx.Table("resource").Omit("last_update_time").Create(toInsert).Error
		if err != nil {
			zap.L().Error("mysql.InsertResources() failed", zap.Error(err))
			return err
		}

		// 7. 针对受影响的大学进行数据聚合更新
		for shortName := range shortNames {
			if err = refreshUniversityStats(tx, shortName); err != nil {
				zap.L().Error("mysql.InsertResources() failed", zap.Error(err))
				return err
			}
		}
		zap.L().Info("mysql.InsertResources() success", zap.Int("count", len(resources)))
		return nil
	})
}

func GetAllUniversityResources() ([]settings.UniversityResources, error) {
	var universityResources []settings.UniversityResources

	// GORM API 要点: 简单查询所有。
	err := db.Table("resource").Where("is_deleted = ?", model.ResourceIsActive).Find(&universityResources).Error

	if err != nil {
		zap.L().Error("GetAllUniversityResources() failed", zap.Error(err))
		return nil, err
	}
	zap.L().Info("GetAllUniversityResources() success", zap.Int("count", len(universityResources)))
	return universityResources, nil
}

func GetResourceByName(name string) (do.Resource, error) {
	var result do.Resource

	// GORM API 要点: WHERE 条件查询单条记录。
	err := db.Table("resource").Where("resource_name = ? and is_deleted = ?", name, model.ResourceIsActive).First(&result).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			zap.L().Error("resource not found")
			return do.Resource{}, errors.New("resource not found")
		}
		zap.L().Error("GetUniversityResourceByName() failed", zap.Error(err))
		return do.Resource{}, err
	}

	zap.L().Info("GetUniversityResourceByName() success", zap.String("name", name))
	return result, nil
}

func GetResourceList(req dto.ResourceGetListReq) ([]do.Resource, error) {
	var (
		resourceDoList []do.Resource
	)

	// 1. 初始化查询实例
	query := db.Table("resource").Where("is_deleted = ?", model.ResourceIsActive)
	// 2. 模糊查询（处理通配符）
	if req.Name != "" {
		nameParam := "%" + req.Name + "%"
		query = query.Where("(title LIKE ? OR short_name LIKE ?)", nameParam, nameParam)
	}

	// 3. 排序处理（防止 SQL 语法错误）
	if req.SortBy != "" {
		dbSortByMap := map[string]string{ // 请求体参数映射成db字段（把大驼峰映射成下划线）
			"id":             "id",
			"name":           "name",
			"size":           "size",
			"type":           "type",
			"lastUpdateTime": "last_update_time", // 核心转换
		}
		dbSortBy := dbSortByMap[req.SortBy] // 把大驼峰映射成下划线
		sortOrder := "ASC"
		if strings.ToUpper(req.SortOrder) == "DESC" {
			sortOrder = "DESC"
		}
		query = query.Order(fmt.Sprintf("%s %s", dbSortBy, sortOrder))
	} else {
		// 默认排序，防止没有 ORDER BY
		query = query.Order("id DESC")
	}

	// 4. 执行查询
	tx := query.Find(&resourceDoList)
	if tx.Error != nil {
		zap.L().Error("GetResourceList() failed", zap.Error(tx.Error))
		return nil, tx.Error
	}
	zap.L().Info("GetResourceList() success", zap.Int("count", len(resourceDoList)))

	return resourceDoList, tx.Error
}

func GetResources(names []string) ([]do.Resource, error) {
	if len(names) == 0 {
		return []do.Resource{}, nil
	}
	var (
		doResources []do.Resource
	)
	err := db.Table("resource").
		Where("is_deleted = ?", model.ResourceIsActive).
		Where("name IN ?", names).
		Find(&doResources).Error
	if err != nil {
		zap.L().Error("GetResources() failed", zap.Strings("names", names), zap.Error(err))
		return nil, err
	}
	zap.L().Info("GetResources() success", zap.Int("count", len(doResources)))
	return doResources, nil
}

// GetResourceByStatus 根据 资源全称、所属高校英文简写、是否被删除进行查询
func GetResourceByStatus(name, shortName string, isDeleted int) (do.Resource, error) {
	var result do.Resource
	// 使用 First，找不到记录会报 gorm.ErrRecordNotFound
	if err := db.Table("resource").
		Where("name = ? AND short_name = ? AND is_deleted = ?", name, shortName, isDeleted).
		First(&result).Error; err != nil {
		zap.L().Error("mysql.GetResourceByStatus() failed", zap.String("name", name), zap.String("shortName", shortName), zap.Int("isDeleted", isDeleted), zap.Error(err))
		return do.Resource{}, err
	}
	zap.L().Info("GetResourceByStatus() success", zap.String("name", name), zap.String("shortName", shortName), zap.Int("isDeleted", isDeleted))
	return result, nil
}

// DelResource 删除列表资源，并同步更新 university 表的数据
func DelResource(req dto.ResourceDelReq) error {
	resource, err := GetResourceByStatus(req.Name, req.ShortName, model.ResourceIsActive) // 先通过 资源名称 拿到资源信息
	if err != nil {
		zap.L().Error("mysql.GetResourceByNameAndSn() failed", zap.Any("req", req), zap.Error(err))
		return err
	}
	return db.Transaction(func(tx *gorm.DB) error {
		// 1. 将删除的资源设置为 is_deleted = 1
		result := tx.Table("resource").Where("id = ?", resource.ID).Updates(map[string]interface{}{"is_deleted": model.ResourceIsDeleted})
		if result.Error != nil {
			zap.L().Error("mysql.DelResource() failed", zap.Any("req", req), zap.Error(result.Error))
			return result.Error
		}
		zap.L().Info("mysql.DelResource() 1. Del Resource success", zap.Int64("deleted_count", result.RowsAffected))

		if err = refreshUniversityStats(tx, resource.ShortName); err != nil {
			zap.L().Error("mysql.refreshUniversityStats() failed", zap.String("short_name", req.ShortName), zap.Error(err))
			return err
		}
		zap.L().Info("mysql.DelResource() 2. refreshUniversityStats() success", zap.Any("req", req))
		return nil
	})
}

// RecoverResource 恢复列表资源，并同步更新 university 表的数据
func RecoverResource(req dto.ResourceRecoverReq) error {
	resource, err := GetResourceByStatus(req.Name, req.ShortName, model.ResourceIsDeleted) // 先通过 资源名称 拿到资源信息
	if err != nil {
		zap.L().Error("mysql.RecoverResource() failed", zap.Any("req", req), zap.Error(err))
		return err
	}
	// 关键防御：确保查询到的 resource 确实有值
	if resource.Name == "" {
		zap.L().Error("mysql.RecoverResource() failed", zap.Any("req", req), zap.Error(fmt.Errorf("resource name is empty for req: %v", req)))
		return fmt.Errorf("resource name is empty for req: %v", req)
	}
	return db.Transaction(func(tx *gorm.DB) error {
		result := tx.Table("resource").Where("id = ?", resource.ID).Updates(map[string]interface{}{"is_deleted": model.ResourceIsActive})
		if result.Error != nil {
			zap.L().Error("mysql.RecoverResource() failed", zap.Any("req", req), zap.Error(result.Error))
			return result.Error
		}
		zap.L().Info("mysql.RecoverResource() 1. Recover Resource success", zap.Int64("deleted_count", result.RowsAffected))
		if err = refreshUniversityStats(tx, resource.ShortName); err != nil {
			zap.L().Error("mysql.RecoverResource() failed", zap.String("short_name", req.ShortName), zap.Error(err))
			return err
		}
		zap.L().Info("mysql.RecoverResource() 2. refreshUniversityStats() success", zap.Any("req", req))
		return nil
	})
}

// refreshUniversityStats 更新指定大学的资源统计信息（必须传入事务中的 tx）
func refreshUniversityStats(tx *gorm.DB, shortName string) error {
	var baseStats struct { // university 表需要首批更新的两个字段
		Count     int
		HasVector int
	}
	// 1. 统计有效资源
	if err := tx.Table("resource").
		Where("short_name = ? AND is_deleted = ?", shortName, model.ResourceIsActive).
		Select("COUNT(*) AS count, MAX(is_vector) AS has_vector").
		Scan(&baseStats).Error; err != nil {
		zap.L().Error("refreshUniversityStats() failed", zap.String("short_name", shortName), zap.Error(err))
		return err
	}

	// 2. 查找最新的主计算文件
	// 显式初始化 mainRes，确保每次查询都是干净的
	var mainRes do.Resource
	if err := tx.Table("resource").
		Where("short_name = ? AND is_deleted = ? AND used_for_edge = 1", shortName, model.ResourceIsActive).
		Order("id DESC").Limit(1).Find(&mainRes).Error; err != nil {
		zap.L().Error("refreshUniversityStats() failed", zap.String("short_name", shortName), zap.Error(err))
		return err // 捕获可能的数据库查询错误
	}

	// 3. 构建更新 Map
	updateData := map[string]interface{}{
		"resource_count":     baseStats.Count,
		"has_vector":         baseStats.HasVector,
		"computation_id":     nil,
		"main_vector_format": "",
	}
	// 只有查到了 ID 才会覆盖 nil
	if mainRes.ID > 0 {
		updateData["computation_id"] = mainRes.ID
		updateData["main_vector_format"] = mainRes.Type
	}
	// 对受影响的高校进行更新：
	return tx.Table("university").Where("short_name = ?", shortName).Updates(updateData).Error
}
