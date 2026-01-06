package do

import "time"

// Resource 代表一个资源，同时用于数据库映射和JSON数据绑定
type Resource struct {
	// ID 是 GORM 默认的主键，但为了清晰，我们显式设置 column
	ID        int    `gorm:"primaryKey;column:id;autoIncrement" json:"id"`
	Title     string `gorm:"column:title" json:"title" binding:"required"`
	ShortName string `gorm:"column:short_name" json:"shortName" binding:"required"`
	Name      string `gorm:"column:name" json:"name" binding:"required"`
	Type      string `gorm:"column:type" json:"type" binding:"required"`
	Md5       string `gorm:"column:md5" json:"md5"`
	Size      int    `gorm:"column:size" json:"size"`

	// LastUpdateTime 对应数据库的 TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
	// 使用 *time.Time 避免零值覆盖，GORM 会忽略 nil 指针
	LastUpdateTime *time.Time `gorm:"column:last_update_time" json:"lastUpdateTime"`

	IsVector        int    `gorm:"column:is_vector" json:"isVector"`
	IsBitmap        int    `gorm:"column:is_bitmap" json:"isBitmap"`
	Width           int    `gorm:"column:width" json:"width"`
	Height          int    `gorm:"column:height" json:"height"`
	UsedForEdge     int    `gorm:"column:used_for_edge" json:"usedForEdge"`
	IsDeleted       int    `gorm:"column:is_deleted" json:"isDeleted"`
	BackgroundColor string `gorm:"column:background_color" json:"backgroundColor"`
}
