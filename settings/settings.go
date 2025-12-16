package settings

import (
	"fmt"
	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
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

func localInit() (err error) {
	viper.SetConfigFile("conf/config.yaml")
	// 读取文件（如果存在）
	err = viper.ReadInConfig()
	if err != nil {
		panic(fmt.Errorf("Fatal error config file: %s \n", err))
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
	viper.AutomaticEnv()                   // 临时启用以读取 RUN_MODE
	runMode := viper.GetString("RUN_MODE") // 新引入的环境变量

	// 需要重新初始化 viper 或确保只读取 RUN_MODE
	// 为了安全起见，我们通常在主函数中调用一次 viper.AutomaticEnv()。
	// 这里我们假设 viper 实例是全局的。
	fmt.Printf("[DEBUG] Detected RUN_MODE: '%s'\n", runMode)

	// 2. 如果是本地模式，使用 config.yaml
	if runMode == "local" {
		fmt.Println("[INFO] Running in LOCAL mode, loading conf/config.yaml...")
		// 1. 读取配置文件
		if err := localInit(); err != nil {
			return err
		}

		// 2. 启用环境变量覆盖（让环境变量在配置文件之后生效）
		// 优化：设置 EnvKeyReplacer，使环境变量能自动映射到嵌套结构体
		viper.SetEnvKeyReplacer(strings.NewReplacer("_", "."))
		viper.AutomaticEnv()

		// 3. 重新 Unmarshal，应用环境变量的覆盖
		if err = viper.Unmarshal(Config); err != nil {
			fmt.Printf("viper.Unmarshal() err: %v\n", err)
			return err
		}
	} else {
		// --- CLOUD/PRODUCTION 模式：完全依赖环境变量 ---
		fmt.Println("[INFO] Running in CLOUD/PRODUCTION mode, prioritizing environment variables...")

		// 优化：设置 EnvKeyReplacer，让 Unmarshal 自动映射环境变量
		// 示例：MYSQL_HOST -> mysql.host
		viper.SetEnvKeyReplacer(strings.NewReplacer("_", "."))
		// 告诉 viper 检查所有环境变量
		viper.AutomaticEnv()

		// 确保所有指针非空，以便 Unmarshal 能够正确填充嵌套字段
		if Config.AppSettings == nil {
			Config.AppSettings = &AppSettings{}
		}
		// ... (其他非空检查，保持原逻辑) ...
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

		// 4. 解析到结构体
		// 这将自动把环境变量映射到结构体（例如将 MYSQL_HOST 映射到 MysqlConfig.Host）
		if err := viper.Unmarshal(Config); err != nil {
			fmt.Printf("viper.Unmarshal() err: %v\n", err)
			return err
		}

		// 5. 打印确认 (使用配置结构体中的值)
		fmt.Println("[INFO] Configuration loaded successfully from environment variables.")
		/*
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
				if secretID := viper.GetString("COS_SECRET_ID"); secretID != "" {
					Config.CosConfig.SecretID = secretID
				}
				if secretKey := viper.GetString("COS_SECRET_KEY"); secretKey != "" {
					Config.CosConfig.SecretKey = secretKey
				}
				if jwtSecret := viper.GetString("JWT_SECRET"); jwtSecret != "" {
					Config.JWTSecret = jwtSecret
				}
				// 解析到结构体
				if err = viper.Unmarshal(Config); err != nil {
					fmt.Printf("viper.Unmarshal() err: %v\n", err)
					return err
				}

				// 打印确认
				fmt.Println("MYSQL_HOST:", Config.MysqlConfig.Host)
				fmt.Println("MYSQL_PORT:", Config.MysqlConfig.Port)
		*/
	}
	// 打印确认 (JWT 密钥等)
	fmt.Printf("JWT Secret Loaded: %s...\n", Config.JWTSecret[:8]) // 打印前8位
	return nil
}
