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
	Port    string `mapstructure:"port"`
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
	Port         string `mapstructure:"port"`
	DBName       string `mapstructure:"dbname"`
	User         string `mapstructure:"user"`
	Password     string `mapstructure:"password"`
	MaxOpenConns int    `mapstructure:"max_open_conns"`
	MaxIdleConns int    `mapstructure:"max_idle_conns"`
}

type RedisConfig struct {
	Host     string `mapstructure:"host"`
	Port     string `mapstructure:"port"`
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

func Init() (err error) {
	viper.SetConfigFile("./conf/config.yaml")
	err = viper.ReadInConfig()
	if err != nil {
		panic(fmt.Errorf("Fatal error config file: %s \n", err))
		return err
	}
	if err = viper.Unmarshal(Config); err != nil {
		fmt.Printf("viper.Unmarshal() err: %v\n", err)
		return err
	}
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
