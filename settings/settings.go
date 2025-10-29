package settings

import (
	"database/sql"
	"fmt"
	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
	"time"
)

var Config = new(AppConfig)

type AppConfig struct {
	AppSettings *AppSettings `mapstructure:"app"`
	LogConfig   *LogConfig   `mapstructure:"log"`
	MysqlConfig *MysqlConfig `mapstructure:"mysql"`
	RedisConfig *RedisConfig `mapstructure:"redis"`
	CosConfig   *CosConfig   `mapstructure:"cos"`
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
	SecretId  string `mapstructure:"secret_id"`
	SecretKey string `mapstructure:"secret_key"`
}
type Universities struct {
	Slug             string         `db:"slug" json:"slug"`
	ShortName        string         `db:"short_name" json:"short_name"`
	Title            string         `db:"title" json:"title"`
	Vis              sql.NullString `db:"vis" json:"vis"`
	Website          string         `db:"website" json:"website"`
	FullNameEn       string         `db:"full_name_en" json:"full_name_en"`
	Region           string         `db:"region" json:"region"`
	Province         string         `db:"province" json:"province"`
	City             string         `db:"city" json:"city"`
	Story            sql.NullString `db:"story" json:"story"`
	HasVector        int            `db:"has_vector" json:"has_vector"`
	MainVectorFormat sql.NullString `db:"main_vector_format" json:"main_vector_format"`
	ResourceCount    int            `db:"resource_count" json:"resource_count"`
	ComputationId    sql.NullInt64  `db:"computation_id" json:"computation_id"`
	CreatedTime      *time.Time     `db:"created_time" json:"created_time"`
	UpdatedTime      *time.Time     `db:"updated_time" json:"updated_time"`
}

type InitUniversities struct {
	Slug       string `db:"slug" json:"slug"`
	ShortName  string `db:"short_name" json:"short_name"`
	Title      string `db:"title" json:"title"`
	Vis        string `db:"vis" json:"vis"`
	Website    string `db:"website" json:"website"`
	FullNameEn string `db:"full_name_en" json:"full_name_en"`
	Region     string `db:"region" json:"region"`
	Province   string `db:"province" json:"province"`
	City       string `db:"city" json:"city"`
	Story      string `db:"story" json:"story"`
}

// UniversityResources 代表一个资源，同时用于数据库映射和JSON数据绑定
type UniversityResources struct {
	Title            string     `db:"title" json:"title" binding:"required"`
	Id               int        `db:"id" json:"id"`
	ShortName        string     `db:"short_name" json:"short_name" binding:"required"`
	ResourceName     string     `db:"resource_name" json:"resource_name" binding:"required"`
	ResourceType     string     `db:"resource_type" json:"resource_type" binding:"required"`
	ResourceMd5      string     `db:"resource_md5" json:"resource_md5"`
	ResourceSizeB    int        `db:"resource_size_b" json:"resource_size_b"`
	LastUpdateTime   *time.Time `db:"last_update_time" json:"last_update_time"`
	IsVector         int        `db:"is_vector" json:"is_vector"`
	IsBitmap         int        `db:"is_bitmap" json:"is_bitmap"`
	ResolutionWidth  int        `db:"resolution_width" json:"resolution_width"`
	ResolutionHeight int        `db:"resolution_height" json:"resolution_height"`
	UsedForEdge      int        `db:"used_for_edge" json:"used_for_edge"`
	IsDeleted        int        `db:"is_deleted" json:"is_deleted"`
	BackgroundColor  string     `db:"background_color" json:"background_color"`
}

func localInit() (err error) {
	viper.SetConfigFile("conf/config.yaml")
	// 读取文件（如果存在）
	err = viper.ReadInConfig()
	if err != nil {
		panic(fmt.Errorf("Fatal error config file: %s \n", err))
		return err
	}
	// 解析到结构体
	if err = viper.Unmarshal(Config); err != nil {
		fmt.Printf("viper.Unmarshal() err: %v\n", err)
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
	return nil
}
func Init() (err error) {
	// 1. 检查运行模式环境变量
	viper.AutomaticEnv()
	runMode := viper.GetString("RUN_MODE") // 新引入的环境变量
	// ⬇️ 临时调试代码
	fmt.Printf("[DEBUG] Detected RUN_MODE: '%s'\n", runMode)
	// 2. 如果是本地模式，使用 config.yaml
	if runMode == "local" {
		fmt.Println("[INFO] Running in LOCAL mode, loading conf/config.yaml...")
		return localInit()
	}
	// 3. 否则，运行在云函数模式（或默认模式），主要依赖环境变量
	fmt.Println("[INFO] Running in CLOUD/PRODUCTION mode, prioritizing environment variables...")

	// 初始化指针，避免 nil
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
	// 环境变量
	if host := viper.GetString("host"); host != "" {
		Config.AppSettings.Host = host
	}
	if port := viper.GetInt("port"); port != 0 {
		Config.AppSettings.Port = port
	}
	if version := viper.GetString("version"); version != "" {
		Config.AppSettings.Version = version
	}
	if name := viper.GetString("name"); name != "" {
		Config.AppSettings.Name = name
	}
	if mode := viper.GetString("mode"); mode != "" {
		Config.AppSettings.Mode = mode
	}
	if level := viper.GetString("LOG_LEVEL"); level != "" {
		Config.LogConfig.Level = level
	}
	if compress := viper.GetString("LOG_COMPRESS"); compress != "" {
		Config.LogConfig.Compress = true
	}
	if host := viper.GetString("MYSQL_HOST"); host != "" {
		Config.MysqlConfig.Host = host
	}
	if port := viper.GetString("MYSQL_PORT"); port != "" {
		// 转 int，防止报错
		var portInt int
		_, err := fmt.Sscanf(port, "%d", &portInt)
		if err != nil {
			return fmt.Errorf("invalid MYSQL_PORT: %v", err)
		}
		Config.MysqlConfig.Port = portInt
	}
	if user := viper.GetString("MYSQL_USER"); user != "" {
		Config.MysqlConfig.User = user
	}
	if pwd := viper.GetString("MYSQL_PASSWORD"); pwd != "" {
		Config.MysqlConfig.Password = pwd
	}
	if dbname := viper.GetString("MYSQL_DBNAME"); dbname != "" {
		Config.MysqlConfig.DBName = dbname
	}

	if host := viper.GetString("REDIS_HOST"); host != "" {
		Config.RedisConfig.Host = host
	}
	if port := viper.GetInt("REDIS_PORT"); port != 0 {
		Config.RedisConfig.Port = port
	}
	if password := viper.GetString("REDIS_PASSWORD"); password != "" {
		Config.RedisConfig.Password = password
	}
	if db := viper.GetInt("REDIS_DB"); db != 0 {
		Config.RedisConfig.DB = db
	}
	if poolSize := viper.GetInt("REDIS_POOL_SIZE"); poolSize != 0 {
		Config.RedisConfig.PoolSize = poolSize
	}
	if bucketUrl := viper.GetString("COS_BUCKET_URL"); bucketUrl != "" {
		Config.CosConfig.BucketUrl = bucketUrl
	}
	if secretId := viper.GetString("COS_SECRET_ID"); secretId != "" {
		Config.CosConfig.SecretId = secretId
	}
	if secretKey := viper.GetString("COS_SECRET_KEY"); secretKey != "" {
		Config.CosConfig.SecretKey = secretKey
	}
	// 解析到结构体
	if err = viper.Unmarshal(Config); err != nil {
		fmt.Printf("viper.Unmarshal() err: %v\n", err)
		return err
	}

	// 打印确认
	fmt.Println("MYSQL_HOST:", Config.MysqlConfig.Host)
	fmt.Println("MYSQL_PORT:", Config.MysqlConfig.Port)

	return nil
}
