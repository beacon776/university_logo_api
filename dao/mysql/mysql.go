package mysql

import (
	"database/sql"
	"errors"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
	"logo_api/settings"
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
	return resource, nil
}

// InsertUniversityResource 在svg资源转化出新位图资源后，在 university_resources 表中记录新位图资源相应信息
func InsertUniversityResource(universityResource settings.UniversityResources) error {
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
	_, err := db.NamedExec(sqlStr, universityResource)
	if err != nil {
		zap.L().Error("insert university_resource failed", zap.Error(err))
	}
	return err
}
