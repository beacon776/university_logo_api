package mysql

import (
	"database/sql"
	"errors"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
	"logo_api/settings"
	"sort"
)

var db *sqlx.DB

// Init 初始化 *sqlx.DB，以便处理MySQL相关操作
func Init(config *settings.MysqlConfig) (err error) {
	// 注意这里用 %%2F，因为 fmt.Sprintf 会把 % 作为格式符号，如果用单%，会出错。
	databaseSource := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Asia%%2FShanghai",
		config.User,
		config.Password,
		config.Host,
		config.Port,
		config.DBName)
	db, err = sqlx.Connect("mysql", databaseSource)
	if err != nil {
		zap.L().Error("sqlx.Connect() failed", zap.Error(err))
		return err
	}
	db.SetMaxOpenConns(config.MaxOpenConns)
	db.SetMaxIdleConns(config.MaxIdleConns)
	return
}

// QueryFromName 它预计是真正的使用函数，把 QueryFromNameAndSvg+QueryFromNameAndBitmapInfo全部替换掉，直接通过computationId字段去查主计算资源
func QueryFromName(preName string) (res settings.UniversityResources, err error) {
	var university settings.Universities
	querySql := `
SELECT * FROM universities WHERE (short_name = ? OR title = ?)`
	if err = db.Get(&university, querySql, preName, preName); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return settings.UniversityResources{}, errors.New("university not found")
		}
		zap.L().Error("db.Get() failed", zap.Error(err))
		return settings.UniversityResources{}, err
	}
	// 检查 ComputationId 是否有效 (非 NULL)
	if !university.ComputationId.Valid {
		return settings.UniversityResources{}, errors.New("university has not yet computed or has no resource ID")
	}
	// 使用 .Int64 提取值
	computationId := university.ComputationId.Int64
	querySql = `
SELECT * FROM university_resources WHERE id = ?`
	if err = db.Get(&res, querySql, computationId); err != nil {
		zap.L().Error("db.Get() failed", zap.Error(err))
		return settings.UniversityResources{}, err
	}
	return res, nil
}

// QueryFromNameAndSvg 在后缀是Svg的情况下进行查询
func QueryFromNameAndSvg(preName string, ext string) (settings.UniversityResources, error) {
	var resource settings.UniversityResources
	querySql := "SELECT * FROM university_resources WHERE (short_name = ? OR title = ?) AND resource_type = ?"
	err := db.Get(&resource, querySql, preName, preName, ext)
	// 查询出错
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) { // 没查到 svg 资源，err 类型为 sql.ErrNoRows
			zap.L().Error("resource not found")
			return resource, err
		}
		// 其他错误
		zap.L().Error("db.Get() err:", zap.Error(err))
		return settings.UniversityResources{}, err
	}
	zap.L().Info("get svg resource:", zap.Any("resource", resource))
	return resource, nil
}

// QueryFromNameAndBitmapInfo 根据文件名、后缀（保证为位图）、边长/宽+高、背景颜色参数查找对应的资源
func QueryFromNameAndBitmapInfo(preName string, ext string, size int, width int, height int, bgColor string) (settings.UniversityResources, error) {
	var resource settings.UniversityResources
	querySql := "SELECT * FROM university_resources WHERE (short_name = ? OR title = ?) AND resource_type = ? AND ((resolution_width = ? AND resolution_height = ?) OR (resolution_width = ? AND resolution_height = ?)) AND is_deleted = 0 AND background_color = ?"
	err := db.Get(&resource, querySql, preName, preName, ext, size, size, width, height, bgColor)
	// 第一次查询出错
	if err != nil {
		// 没查到
		if errors.Is(err, sql.ErrNoRows) {
			zap.L().Error("current resource not found, next will try to find svg resource")
			// 虽然直接查没查到，但是还有机会查到 svg 资源，继续去查 svg 资源
			querySql = "SELECT * FROM university_resources WHERE (short_name = ? OR title = ?) AND used_for_edge = 1"
			err = db.Get(&resource, querySql, preName, preName)
			// 第二次查询出错
			if err != nil {
				// svg 资源也没查到，err 类型为 sql.ErrNoRows
				if errors.Is(err, sql.ErrNoRows) {
					zap.L().Error("svg resource was not founded as well, this bitmap resource could not be found")
					return settings.UniversityResources{}, err
				}
				// 其他错误
				zap.L().Error("db.Get() err:", zap.Error(err))
				return settings.UniversityResources{}, err
			}
			// 查到了 svg 资源
			return resource, nil
		}
		// 其他错误
		zap.L().Error("db.Get() failed", zap.Error(err))
		return settings.UniversityResources{}, err
	}
	// 直接查到了该资源
	zap.L().Info("get bitmap resource:", zap.Any("resource", resource))
	return resource, nil
}

// InsertUniversityResource 基于结构体数组+db.NamedExec()方法在1.31版本后的特性，对 university_resources 表进行批量插入
func InsertUniversityResource(universityResources []settings.UniversityResources) error {
	sqlStr := `
		INSERT INTO university_resources (
			short_name, title, resource_name, resource_type, resource_md5,
			resource_size_b, last_update_time, is_vector, is_bitmap,
			resolution_width, used_for_edge, is_deleted, resolution_height, background_color
		) VALUES (
			:short_name, :title, :resource_name, :resource_type, :resource_md5,
			:resource_size_b, :last_update_time, :is_vector, :is_bitmap,
			:resolution_width, :used_for_edge, :is_deleted, :resolution_height, :background_color
		)
	`
	_, err := db.NamedExec(sqlStr, universityResources)
	if err != nil {
		zap.L().Error("InsertUniversityResource() failed", zap.Error(err))
		return err
	}
	zap.L().Info("InsertUniversityResource() success")
	return err
}

func UpdateUniversityResource(universityResource settings.UniversityResources) error {
	sqlStr := `
		UPDATE university_resources SET 
			short_name = :short_name,
		    title = :title,
           resource_name = :resource_name,
           resource_type = :resource_type,
           resource_md5 = :resource_md5,
           resource_size_b = :resource_size_b,
           last_update_time = :last_update_time,
           is_vector = :is_vector,
           is_bitmap = :is_bitmap,
           resolution_width = :resolution_width,
           used_for_edge = :used_for_edge,
           is_deleted = :is_deleted,
           resolution_height = :resolution_height,
           background_color = :background_color
		WHERE id = :id`

	_, err := db.NamedExec(sqlStr, universityResource)
	if err != nil {
		zap.L().Error("UpdateUniversityResource() failed", zap.Error(err))
		return err
	}
	zap.L().Info("UpdateUniversityResource() success")
	return nil
}

func DeleteUniversityResource(universityResource settings.UniversityResources) error {
	sqlStr := `UPDATE university_resources SET is_deleted = 1 WHERE id = :id`
	_, err := db.NamedExec(sqlStr, universityResource)
	if err != nil {
		zap.L().Error("DeleteUniversityResource() failed", zap.Error(err))
		return err
	}
	zap.L().Info("DeleteUniversityResource() success")
	return nil
}

// GetAllUniversities　与InitUniversities 搭配使用，先查出来所有，然后再遍历，对每一个学校进行初始化操作
func GetAllUniversities() ([]settings.Universities, error) {
	var universities []settings.Universities
	sqlStr := `SELECT * FROM universities`
	err := db.Select(&universities, sqlStr)
	if err != nil {
		zap.L().Error("GetAllUniversities() failed", zap.Error(err))
		return nil, err
	}
	zap.L().Info("GetAllUniversities() success")
	return universities, nil
}

func GetInitUniversities() ([]settings.InitUniversities, error) {
	var universities []settings.InitUniversities
	sqlStr := `SELECT * FROM universities`
	err := db.Select(&universities, sqlStr)
	if err != nil {
		zap.L().Error("GetInitUniversities() failed", zap.Error(err))
		return nil, err
	}
	zap.L().Info("GetInitUniversities() success")
	return universities, nil
}

func GetAllUniversityResources() ([]settings.UniversityResources, error) {
	var universityResources []settings.UniversityResources
	sqlStr := `SELECT * FROM university_resources`
	err := db.Select(&universityResources, sqlStr)
	if err != nil {
		zap.L().Error("GetAllUniversityResources() failed", zap.Error(err))
		return nil, err
	}
	zap.L().Info("GetAllUniversityResources() success")
	return universityResources, err
}

// InitUniversitiesParams 它的作用是通过 university_resources 表的数据，为 universities 表 的 has_vector, main_vector_format, resource_count, computation_id 四个字段进行初始化的计算
func InitUniversitiesParams(universities settings.Universities) error {
	var universityResources []settings.UniversityResources
	sqlStr := `SELECT * FROM university_resources WHERE short_name = ?`
	if err := db.Select(&universityResources, sqlStr, universities.ShortName); err != nil {
		zap.L().Error("db.Select() failed", zap.Error(err))
		return err
	}
	universities.HasVector = 0
	universities.MainVectorFormat = sql.NullString{}
	universities.ComputationId = sql.NullInt64{}
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
				universities.ComputationId = sql.NullInt64{
					Int64: int64(resource.Id),
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
			universities.ComputationId = sql.NullInt64{
				Int64: int64(universityResources[0].Id),
				Valid: true,
			}
		}
	}

	UpdateSqlStr := `
UPDATE universities SET
                        has_vector = :has_vector,
                        main_vector_format = :main_vector_format,
                        computation_id = :computation_id,
                        resource_count = :resource_count
                        WHERE short_name = :short_name`
	if _, err := db.NamedExec(UpdateSqlStr, universities); err != nil { // 这里需要保证 universities 是一个单一结构体读写，不能是切片或者数组哦
		zap.L().Error("UpdateUniversities() failed", zap.Error(err))
		return err
	}
	zap.L().Info("UpdateUniversities() success")
	return nil
}

func InsertUniversities(universities []settings.Universities) error {
	insertUniversitySql := `INSERT INTO universities(
                         slug, short_name, title, vis, 
                         website, full_name_en, region, province, city, 
                         story, has_vector, main_vector_format, resource_count, 
                         computation_id, created_time, updated_time
) VALUES(:slug, :short_name, :title, :vis, 
                         :website, :full_name_en, :region, :province, :city, 
                         :story, :has_vector, :main_vector_format, :resource_count, 
                         :computation_id, :created_time, :updated_time) `
	if _, err := db.NamedExec(insertUniversitySql, universities); err != nil {
		zap.L().Error("db.NamedExec(insertUniversitySql, universities) failed", zap.Error(err))
		return err
	}
	zap.L().Info("db.NamedExec(insertUniversitySql, universities) successfully")
	return nil
}

func InitInsertUniversities(universities []settings.InitUniversities) error {
	insertUniversitySql := `INSERT INTO universities(
                         slug, short_name, title, vis, 
                         website, full_name_en, region, province, city, story
) VALUES(:slug, :short_name, :title, :vis, 
                         :website, :full_name_en, :region, :province, :city, :story) `
	if _, err := db.NamedExec(insertUniversitySql, universities); err != nil {
		zap.L().Error("db.NamedExec(insertUniversitySql, universities) failed", zap.Error(err))
		return err
	}
	zap.L().Info("db.NamedExec(insertUniversitySql, universities) successfully")
	return nil
}
