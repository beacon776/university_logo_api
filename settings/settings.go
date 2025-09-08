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
	Slug                   string     `db:"slug"`
	ShortName              string     `db:"short_name"`
	Title                  string     `db:"title"`
	Vis                    string     `db:"vis"`
	Website                string     `db:"website"`
	FullNameEn             string     `db:"full_name_en"`
	Region                 string     `db:"region"`
	Province               string     `db:"province"`
	City                   string     `db:"city"`
	Story                  string     `db:"story"`
	HasVector              bool       `db:"has_vector"`
	MainVectorFormat       string     `db:"main_vector_format"`
	MainVectorSizeB        int        `db:"main_vector_size_b"`
	HasBitmap              bool       `db:"has_bitmap"`
	MainBitmapFormat       string     `db:"main_bitmap_format"`
	MainBitmapSizeB        int        `db:"main_bitmap_size_b"`
	ResourcesCount         int        `db:"resources_count"`
	EdgeComputationInputId int        `db:"edge_computation_input_id"`
	CreatedTime            *time.Time `db:"created_time"`
	UpdatedTime            *time.Time `db:"updated_time"`
}

type UniversityResources struct {
	Title            string        `db:"title"`
	Id               int           `db:"id"`
	ShortName        string        `db:"short_name"`
	ResourceName     string        `db:"resource_name"`
	ResourceType     string        `db:"resource_type"`
	ResourceMd5      string        `db:"resource_md5"`
	ResourceSizeB    sql.NullInt64 `db:"resource_size_b"`
	LastUpdateTime   sql.NullTime  `db:"last_update_time"`
	IsVector         bool          `db:"is_vector"`
	IsBitmap         bool          `db:"is_bitmap"`
	ResolutionWidth  sql.NullInt64 `db:"resolution_width"`
	ResolutionHeight sql.NullInt64 `db:"resolution_height"`
	UsedForEdge      bool          `db:"used_for_edge"`
	IsDeleted        bool          `db:"is_deleted"`
	BackgroundColor  string        `db:"background_color"`
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
	// 环境变量优先
	viper.AutomaticEnv()

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
