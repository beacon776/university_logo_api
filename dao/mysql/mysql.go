package mysql

import (
	"fmt"
	"go.uber.org/zap"
	"gorm.io/driver/mysql" // GORM MySQL 驱动
	"gorm.io/gorm"         // GORM 核心库
	"gorm.io/gorm/logger"  // 导入 GORM logger
	"logo_api/settings"
)

// 将全局 db 变量类型更改为 *gorm.DB
var db *gorm.DB

// GetDB 公开一个获取 GORM 实例的方法，如果需要的话
func GetDB() *gorm.DB {
	return db
}

// Init 初始化 *gorm.DB，以便处理MySQL相关操作
func Init(config *settings.MysqlConfig) (err error) {
	// 1. 构建 DSN (Data Source Name)
	// 注意：GORM 的 MySQL 驱动可以直接处理 %2F，无需使用 %%2F
	databaseSource := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		config.User,
		config.Password,
		config.Host,
		config.Port,
		config.DBName)

	// GORM 配置项
	gormConfig := &gorm.Config{
		// 开启 SQL 日志
		Logger: logger.Default.LogMode(logger.Info), // 或者 logger.Silent/Warn/Error/Info
	}

	// 2. 使用 gorm.Open 连接数据库
	dbInstance, err := gorm.Open(mysql.Open(databaseSource), gormConfig)
	if err != nil {
		// 注意：GORM 的连接日志在 Open 失败时也会在驱动层打印，但我们仍然需要记录应用程序级别的错误
		zap.L().Error("gorm.Open() failed", zap.Error(err))
		return err
	}

	// 3. 获取底层 *sql.DB 对象以设置连接池参数
	sqlDB, err := dbInstance.DB()
	if err != nil {
		zap.L().Error("dbInstance.DB() failed", zap.Error(err))
		return err
	}

	// 4. 设置连接池参数（与 sqlx 类似）
	sqlDB.SetMaxOpenConns(config.MaxOpenConns)
	sqlDB.SetMaxIdleConns(config.MaxIdleConns)

	// 5. 赋值给全局变量
	db = dbInstance

	// GORM 的 AutoMigrate 可以在这里执行，用于自动创建或更新表结构
	// 例如: db.AutoMigrate(&model.User{})

	return
}
