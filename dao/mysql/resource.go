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

// InsertUniversityResource 基于结构体数组+db.Create()方法，对 university_resources 表进行批量插入
func InsertUniversityResource(universityResources []settings.UniversityResources) error {
	// GORM API 要点: 批量插入。
	// 对切片使用 db.Create()，GORM 自动处理字段映射和批量 INSERT
	if len(universityResources) == 0 {
		return nil
	}

	err := db.Table("resource").Create(&universityResources).Error
	if err != nil {
		zap.L().Error("InsertUniversityResource() failed", zap.Error(err))
		return err
	}

	zap.L().Info("InsertUniversityResource() success", zap.Int("count", len(universityResources)))
	return nil
}

func UpdateUniversityResource(universityResource settings.UniversityResources) error {
	// GORM API 要点: 保存所有字段。
	// db.Save() 会根据结构体的主键 (ID) 来执行 UPDATE 或 INSERT/UPDATE (Upsert)。
	// 它会更新结构体中的所有字段（包括零值），这是最简单的全字段更新方法。
	err := db.Table("resource").Save(&universityResource).Error

	if err != nil {
		zap.L().Error("UpdateUniversityResource() failed", zap.Error(err))
		return err
	}
	zap.L().Info("UpdateUniversityResource() success", zap.Int("id", universityResource.ID))
	return nil

	// 如果只需要更新非零值，可以使用 db.Updates(&universityResource)。
	// 如果只需要更新特定字段，可以使用 db.Model().Where("id = ?", universityResource.Id).Updates(map[string]interface{})
}

func DeleteUniversityResource(universityResource settings.UniversityResources) error {
	// GORM API 要点: 软删除。
	// 假设您的 model.UniversityResources 结构体中没有 gorm.DeletedAt 字段，我们使用 Updates 实现逻辑删除。

	// db.Model() 指定要操作的对象
	// db.Where() 指定要更新的记录
	// db.Update() 更新单个字段
	err := db.Table("resource").Model(&settings.UniversityResources{}).Where("id = ?", universityResource.ID).Update("is_deleted", 1).Error

	// 如果您的结构体包含 gorm.DeletedAt 字段，则可以使用 db.Delete(&universityResource) 来触发 GORM 的内置软删除。

	if err != nil {
		zap.L().Error("DeleteUniversityResource() failed", zap.Error(err))
		return err
	}
	zap.L().Info("DeleteUniversityResource() success", zap.Int("id", universityResource.ID))
	return nil
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

func DelResources(names []string) error {
	if len(names) == 0 {
		zap.L().Error("DelResources() failed", zap.Strings("names", names))
		return nil
	}
	result := db.Table("resource").Where("name IN ? AND is_deleted = ?", names, model.ResourceIsActive).Updates(map[string]interface{}{"is_deleted": model.ResourceIsDeleted})
	if result.Error != nil {
		zap.L().Error("DelResources() failed", zap.Strings("names", names), zap.Error(result.Error))
		return result.Error
	}
	zap.L().Info("DelResources() success", zap.Int64("deleted_count", result.RowsAffected))

	return nil
}

func RecoverResources(names []string) error {
	if len(names) == 0 {
		zap.L().Error("RecoverResources() failed", zap.Strings("names", names))
		return nil
	}
	result := db.Table("resource").Where("name IN ? AND is_deleted = ?", names, model.ResourceIsDeleted).Updates(map[string]interface{}{"is_deleted": model.ResourceIsActive})
	if result.Error != nil {
		zap.L().Error("RecoverResources() failed", zap.Strings("names", names), zap.Error(result.Error))
		return result.Error
	}
	zap.L().Info("RecoverResources() success", zap.Int64("deleted_count", result.RowsAffected))
	return nil
}
