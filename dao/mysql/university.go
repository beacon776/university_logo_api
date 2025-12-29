package mysql

import (
	"errors"
	"fmt"
	"go.uber.org/zap"
	"logo_api/model/user/do"
	"logo_api/model/user/dto"
	"logo_api/settings"
	// 确保导入 GORM
	"gorm.io/gorm"
)

// 假设 db 已经是 *gorm.DB 类型，并已在 Init 中初始化

/*
// QueryFromName 它预计是真正的使用函数，把 QueryFromNameAndSvg+QueryFromNameAndBitmapInfo全部替换掉，直接通过computationId字段去查主计算资源
func QueryFromName(preName string) (res settings.UniversityResources, err error) {
	var university settings.Universities

	// GORM API 要点: 复合 WHERE 条件。
	// 使用 db.Where() 配合 OR 子句查询 universities 表
	err = db.Where("short_name = ? OR title = ?", preName, preName).First(&university).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) { // GORM 找不到记录时返回 gorm.ErrRecordNotFound
			return settings.UniversityResources{}, errors.New("university not found")
		}
		zap.L().Error("db.First() failed", zap.Error(err))
		return settings.UniversityResources{}, err
	}

	// 检查 ComputationId 是否有效 (非 NULL)
	if !university.ComputationID.Valid {
		return settings.UniversityResources{}, errors.New("university has not yet computed or has no resource ID")
	}

	// 使用 .Int64 提取值
	computationId := university.ComputationID.Int64

	// GORM API 要点: 根据主键 ID 查询（GORM 快捷方式）。
	// db.First(&res, ID) 相当于 WHERE id = ID
	if err = db.First(&res, computationId).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			zap.L().Error("university_resources not found for computationId", zap.Int64("id", computationId), zap.Error(err))
			// 这里的错误处理取决于业务逻辑，如果资源不存在是正常情况，可以不返回错误
			return settings.UniversityResources{}, errors.New("main resource not found")
		}
		zap.L().Error("db.First() failed", zap.Error(err))
		return settings.UniversityResources{}, err
	}
	return res, nil
}
*/

// GetAllUniversities　与InitUniversities 搭配使用，先查出来所有，然后再遍历，对每一个学校进行初始化操作
func GetAllUniversities() ([]settings.Universities, error) {
	var universities []settings.Universities

	// GORM API 要点: 简单查询所有。
	// 使用 Find() 查询所有记录，GORM 自动映射到切片
	err := db.Table("university").Find(&universities).Error

	if err != nil {
		zap.L().Error("GetAllUniversities() failed", zap.Error(err))
		return nil, err
	}
	zap.L().Info("GetAllUniversities() success")
	return universities, nil
}

func GetInitUniversities() ([]settings.InitUniversities, error) {
	var universities []settings.InitUniversities

	// GORM API 要点: 查询并映射到不同结构体。
	// 使用 Find() 查询所有记录并映射到目标结构体
	err := db.Table("university").Find(&universities).Error

	if err != nil {
		zap.L().Error("GetInitUniversities() failed", zap.Error(err))
		return nil, err
	}
	zap.L().Info("GetInitUniversities() success")
	return universities, nil
}

func GetUniversityByName(name string) (do.University, error) {
	var university do.University
	// GORM API 要点: WHERE 条件查询单条记录。
	/*
		// 只查询 is_deleted = 0 的记录(未删除) 目前没有这个字段 */
	// 使用 db.Where() 设置条件，并用 First() 获取结果
	err := db.Table("university").Where("short_name = ? OR title = ?", name, name).First(&university).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// 找不到记录
			return do.University{}, errors.New("university not found")
		}
		zap.L().Error("GetUniversityByName() failed", zap.Error(err))
		return do.University{}, err
	}
	zap.L().Info("GetUniversityByName() success", zap.String("name", name))
	return university, nil
}

/*
// InitUniversitiesParams 它的作用是通过 university_resources 表的数据，为 universities 表 的 has_vector, main_vector_format, resource_count, computation_id 四个字段进行初始化的计算
func InitUniversitiesParams(universities settings.Universities) error {
	var universityResources []settings.UniversityResources

	// GORM API 要点: 条件查询多条记录。
	// 使用 db.Where() 设置条件，并用 Find() 获取结果切片
	if err := db.Where("short_name = ?", universities.ShortName).Find(&universityResources).Error; err != nil {
		zap.L().Error("db.Find() failed", zap.Error(err))
		return err
	}

	// --- 原有计算逻辑（Go代码部分，保持不变）---
	universities.HasVector = 0
	universities.MainVectorFormat = ""
	universities.ComputationID = -1
	universities.ResourceCount = len(universityResources)
	if len(universityResources) == 0 {
		zap.L().Debug("当前院校没有对应的下载资源", zap.String("院校名称", universities.Title))
	} else {
		for _, resource := range universityResources {
			if resource.IsVector == 1 && resource.ResourceType == "svg" {
				universities.HasVector = 1
				universities.MainVectorFormat = sql.NullString{
					String: resource.ResourceType,
					Valid:  true,
				}
				universities.ComputationID = sql.NullInt64{
					Int64: int64(resource.ID),
					Valid: true,
				}
				break
			}
		}
		if universities.HasVector == 0 {
			sort.Slice(universityResources, func(i, j int) bool {
				widthI, heightI := universityResources[i].ResolutionWidth, universityResources[i].ResolutionHeight
				widthJ, heightJ := universityResources[j].ResolutionWidth, universityResources[j].ResolutionHeight
				return widthI*heightI > widthJ*heightJ
			})
			universities.ComputationID = sql.NullInt64{
				Int64: int64(universityResources[0].ID),
				Valid: true,
			}
		}
	}
	// ----------------------------------------

	// GORM API 要点: 结构体更新指定字段。
	// 1. 使用 db.Model() 指定模型，
	// 2. 使用 db.Where() 指定更新条件（这里我们用 ShortName，也可以用 ID），
	// 3. 使用 Updates() 传入需要更新的字段名和值。
	// 注意：GORM 的 Updates() 默认只更新非零值字段，但如果结构体字段是 sql.NullString/sql.NullInt64，GORM 会正确处理。
	// 或者，为了清晰，我们可以使用 db.Save(&universities) 来保存所有修改后的字段（前提是 universities 包含主键）。
	if err := db.Model(&universities).Where("short_name = ?", universities.ShortName).Updates(map[string]interface{}{
		"has_vector":         universities.HasVector,
		"main_vector_format": universities.MainVectorFormat,
		"computation_id":     universities.ComputationID,
		"resource_count":     universities.ResourceCount,
	}).Error; err != nil {
		zap.L().Error("UpdateUniversities() failed", zap.Error(err))
		return err
	}
	zap.L().Info("UpdateUniversities() success")
	return nil
}
*/

func InsertUniversities(universities []settings.Universities) error {
	if len(universities) == 0 {
		return nil
	}

	// GORM API 要点: 批量插入。
	// 对切片使用 db.Create()，GORM 会自动生成批量 INSERT 语句

	// 核心修改：使用 Omit() 排除 CreatedTime 和 UpdatedTime 字段
	// GORM 的 Omit 方法参数是 Go 结构体中的字段名 (即 CreatedTime, UpdatedTime)
	if err := db.Table("university").Omit("CreatedTime", "UpdatedTime").Create(&universities).Error; err != nil {
		zap.L().Error("db.Create(universities) failed", zap.Error(err))
		return err
	}

	zap.L().Info("db.Create(universities) successfully", zap.Int("count", len(universities)))
	return nil
}

func InitInsertUniversities(universities []settings.InitUniversities) error {
	if len(universities) == 0 {
		return nil
	}

	// GORM API 要点: 批量插入（部分字段）。
	// 对切片使用 db.Create()。由于 settings.InitUniversities 是部分字段，GORM 只插入该结构体中的字段。
	if err := db.Table("university").Create(&universities).Error; err != nil {
		zap.L().Error("db.Create(universities) failed", zap.Error(err))
		return err
	}

	zap.L().Info("db.Create(universities) successfully", zap.Int("count", len(universities)))
	return nil
}

func GetUniversityList(req dto.UniversityGetListReq) (universities []do.University, totalCount int64, err error) {
	page := req.Page
	pageSize := req.PageSize
	keyword := req.Keyword
	sortBy := req.SortBy
	sortOrder := req.SortOrder
	// 1. 初始化查询构建器
	// db.Model(&model.Universities{}) 创建一个基于 Universities 模型的查询事务 (tx)
	tx := db.Table("university").Model(&do.University{})

	/*
		// 只查询 is_deleted = 0 的记录
		目前没有这个字段
		tx = tx.Where("is_deleted = ?", 0)*/

	// 2. 处理关键字搜索 (Keyword)
	// 根据图片中 'keyword' 的说明 "模糊查询"
	if keyword != "" {
		query := "%" + keyword + "%"
		// 模糊查询：查找 'title' 字段中包含 keyword 的记录
		// GORM 的 Where 方法用于构建 WHERE 子句
		tx = tx.Where("(title LIKE ?) or (short_name LIKE ?)", query, query)
	}

	// 3. 统计总记录数 (Total Count)
	// 在应用分页 (Limit/Offset) 之前，先统计符合搜索条件的总数
	// GORM 的 Count 方法会将结果存储到 totalCount 变量
	if err = tx.Count(&totalCount).Error; err != nil {
		return nil, 0, err
	}

	// 4. 处理排序 (SortBy & SortOrder)
	// GORM 的 Order 方法用于构建 ORDER BY 子句
	// 排序字段 (sortBy) 和排序方向 (sortOrder) 拼接成 "字段 方向" 的形式
	orderClause := sortBy + " " + sortOrder
	tx = tx.Order(orderClause)

	// 5. 处理分页 (Pagination)
	// 计算偏移量 Offset = (page - 1) * pageSize
	offset := (page - 1) * pageSize

	// GORM 的 Limit 用于 LIMIT 子句 (每页记录数)
	tx = tx.Limit(pageSize)

	// GORM 的 Offset 用于 OFFSET 子句 (跳过多少记录)
	tx = tx.Offset(offset)

	// 6. 执行查询 (Find)
	// Find 方法执行 SELECT 查询，并将结果映射到 universities 变量
	if err = tx.Find(&universities).Error; err != nil {
		return nil, 0, err
	}

	return universities, totalCount, nil
}

// UpdateUniversities 根据传入的 model.Universities 数组，更新 universities 表
func UpdateUniversities(universities []do.University) error {
	if len(universities) == 0 {
		zap.L().Info("UpdateUniversities() no universities")
		return nil
	}
	return db.Table("university").Transaction(func(tx *gorm.DB) error {
		for _, v := range universities {
			// 1. 检查主键是否存在
			if v.Slug == "" {
				return fmt.Errorf("university slug cannot be empty")
			}
			// 2. 执行更新
			// 使用 Select("*") 或将结构体转为 map 可以强制更新零值
			// 这里指定 Model 为 Universities 结构体对应的表
			result := tx.Model(&do.University{}).
				Omit("CreatedTime", "UpdatedTime").
				Where("slug = ?", v.Slug).
				Updates(&v) // 如果 v 中是字段是指针，nil 不更新，非 nil 的零值会更新
			if result.Error != nil {
				zap.L().Error("UpdateUniversities failed", zap.Error(result.Error), zap.String("slug", v.Slug))
				return result.Error // 返回错误，事务会自动回滚
			}
			// 如果没有匹配到行，报错
			if result.RowsAffected == 0 {
				// 这里仅记录日志，不中断事务。
				zap.L().Warn("No record found to update", zap.String("slug", v.Slug))
			}
		}
		zap.L().Info("UpdateUniversities() success", zap.Int("count", len(universities)))
		return nil // 返回 nil，事务会提交
	})
}
