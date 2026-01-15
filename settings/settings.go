package settings

import (
	"fmt"
	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
	"os"
	"strings"
	"time"
)

var Config = new(AppConfig)

type AppConfig struct {
	AppSettings *AppSettings `mapstructure:"app"`
	LogConfig   *LogConfig   `mapstructure:"log"`
	MysqlConfig *MysqlConfig `mapstructure:"mysql"`
	RedisConfig *RedisConfig `mapstructure:"redis"`
	CosConfig   *CosConfig   `mapstructure:"cos"`
	JWTSecret   string       `mapstructure:"jwt_secret"`
}

type AppSettings struct {
	Host    string `mapstructure:"host"`
	Port    int    `mapstructure:"port"`
	Version string `mapstructure:"version"`
	Mode    string `mapstructure:"mode"`
	Name    string `mapstructure:"name"`
}

type LogConfig struct {
	Level      string `mapstructure:"level"`
	Filename   string `mapstructure:"filename"`
	MaxSize    int    `mapstructure:"max_size"`
	MaxBackups int    `mapstructure:"max_backups"`
	MaxAge     int    `mapstructure:"max_age"`
	Compress   bool   `mapstructure:"compress"`
}

type MysqlConfig struct {
	Host         string `mapstructure:"host"`
	Port         int    `mapstructure:"port"`
	DBName       string `mapstructure:"dbname"`
	User         string `mapstructure:"user"`
	Password     string `mapstructure:"password"`
	MaxOpenConns int    `mapstructure:"max_open_conns"`
	MaxIdleConns int    `mapstructure:"max_idle_conns"`
}

type RedisConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
	PoolSize int    `mapstructure:"pool_size"`
}
type CosConfig struct {
	BucketUrl string `mapstructure:"bucket_url"`
	SecretID  string `mapstructure:"secret_id"`
	SecretKey string `mapstructure:"secret_key"`
}

type Universities struct {
	Slug      string `gorm:"column:slug;primaryKey" json:"slug"`
	ShortName string `gorm:"column:short_name" json:"short_name"`
	Title     string `gorm:"column:title" json:"title"`
	// 使用 *string 处理 NULL 字段
	Vis              *string `gorm:"column:vis" json:"vis"`
	Website          string  `gorm:"column:website" json:"website"`
	FullNameEn       string  `gorm:"column:full_name_en" json:"full_name_en"`
	Region           string  `gorm:"column:region" json:"region"`
	Province         string  `gorm:"column:province" json:"province"`
	City             string  `gorm:"column:city" json:"city"`
	Story            *string `gorm:"column:story" json:"story"`
	HasVector        int     `gorm:"column:has_vector" json:"has_vector"`
	MainVectorFormat *string `gorm:"column:main_vector_format" json:"main_vector_format"`
	ResourceCount    int     `gorm:"column:resource_count" json:"resource_count"`
	ComputationID    *int    `gorm:"column:computation_id" json:"computation_id"`

	CreatedTime *time.Time `gorm:"column:created_time" json:"created_time"`
	UpdatedTime *time.Time `gorm:"column:updated_time" json:"updated_time"`
}

type InitUniversities struct {
	Slug      string `gorm:"column:slug;primaryKey" json:"slug"`
	ShortName string `gorm:"column:short_name" json:"short_name"`
	Title     string `gorm:"column:title" json:"title"`
	// 使用 *string 处理 NULL 字段
	Vis        *string `gorm:"column:vis" json:"vis"`
	Website    string  `gorm:"column:website" json:"website"`
	FullNameEn string  `gorm:"column:full_name_en" json:"full_name_en"`
	Region     string  `gorm:"column:region" json:"region"`
	Province   string  `gorm:"column:province" json:"province"`
	City       string  `gorm:"column:city" json:"city"`
	Story      *string `gorm:"column:story" json:"story"`
}

// UniversityResources 代表一个资源，同时用于数据库映射和JSON数据绑定
type UniversityResources struct {
	// ID 是 GORM 默认的主键，但为了清晰，我们显式设置 column
	ID            int    `gorm:"primaryKey;column:id" json:"id"`
	Title         string `gorm:"column:title" json:"title" binding:"required"`
	ShortName     string `gorm:"column:short_name" json:"short_name" binding:"required"`
	ResourceName  string `gorm:"column:resource_name" json:"resource_name" binding:"required"`
	ResourceType  string `gorm:"column:resource_type" json:"resource_type" binding:"required"`
	ResourceMd5   string `gorm:"column:resource_md5" json:"resource_md5"`
	ResourceSizeB int    `gorm:"column:resource_size_b" json:"resource_size_b"`

	// LastUpdateTime 对应数据库的 TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
	// 使用 *time.Time 避免零值覆盖，GORM 会忽略 nil 指针
	LastUpdateTime *time.Time `gorm:"column:last_update_time" json:"last_update_time"`

	IsVector         int    `gorm:"column:is_vector" json:"is_vector"`
	IsBitmap         int    `gorm:"column:is_bitmap" json:"is_bitmap"`
	ResolutionWidth  int    `gorm:"column:resolution_width" json:"resolution_width"`
	ResolutionHeight int    `gorm:"column:resolution_height" json:"resolution_height"`
	UsedForEdge      int    `gorm:"column:used_for_edge" json:"used_for_edge"`
	IsDeleted        int    `gorm:"column:is_deleted" json:"is_deleted"`
	BackgroundColor  string `gorm:"column:background_color" json:"background_color"`
}

func Init() (err error) {
	// 初始化 viper 实例
	viper.Reset()

	// 1. 直接用标准库 os 获取模式，避免 Viper 还没初始化导致的逻辑错误
	runMode := strings.ToLower(os.Getenv("RUN_MODE"))
	fmt.Printf("[DEBUG] Detected RUN_MODE: '%s'\n", runMode)
	// 2. 检查运行模式环境变量
	// 优化：设置 EnvKeyReplacer，使环境变量能自动映射到嵌套结构体
	// 示例：MYSQL_HOST -> mysql.host
	viper.SetEnvKeyReplacer(strings.NewReplacer("_", "."))
	viper.AutomaticEnv()

	// 3. 加载配置文件（仅 local 模式）
	if runMode == "local" {
		// 1. 读取配置文件
		fmt.Println("[INFO] Local mode: loading conf/test_config.yaml")
		viper.SetConfigFile("conf/test_config.yaml")
		// 读取文件（如果存在）
		if err := viper.ReadInConfig(); err != nil {
			panic(fmt.Errorf("Fatal error read config file: %s \n", err))
			return err
		}

		// 热更新（仅本地用）
		viper.WatchConfig()
		viper.OnConfigChange(func(e fsnotify.Event) {
			fmt.Printf("Config file changed: %s\n", e.Name)
			if err := viper.Unmarshal(Config); err != nil {
				fmt.Printf("viper.Unmarshal() err: %v\n", err)
				return
			}
		})
	} else {
		// --- CLOUD/PRODUCTION 模式：完全依赖环境变量 ---
		fmt.Println("[INFO] Running in CLOUD/PRODUCTION mode, prioritizing environment variables...")

	}

	// 4. 统一处理：初始化指针并解析到结构体
	ensureConfigPointers()

	// 5. 解析到结构体
	// 这将自动把环境变量映射到结构体（例如将 MYSQL_HOST 映射到 MysqlConfig.Host）
	if err := viper.Unmarshal(Config); err != nil {
		fmt.Printf("viper.Unmarshal() err: %v\n", err)
		return err
	}

	// 6. 打印确认 (JWT 密钥等)
	fmt.Printf("JWT Secret Loaded: %s...\n", Config.JWTSecret[:8]) // 打印前8位
	// 打印当前生效的文件名，方便排错
	fmt.Printf("[DEBUG] Using config file: %s\n", viper.ConfigFileUsed())

	return nil
}

func ensureConfigPointers() {
	// 确保所有指针非空，以便 Unmarshal 能够正确填充嵌套字段
	if Config.AppSettings == nil {
		Config.AppSettings = &AppSettings{}
	}
	if Config.LogConfig == nil {
		Config.LogConfig = &LogConfig{}
	}
	if Config.MysqlConfig == nil {
		Config.MysqlConfig = &MysqlConfig{}
	}
	if Config.RedisConfig == nil {
		Config.RedisConfig = &RedisConfig{}
	}
	if Config.CosConfig == nil {
		Config.CosConfig = &CosConfig{}
	}
}
