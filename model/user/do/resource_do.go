package do

import "time"

// Resource 代表一个资源，同时用于数据库映射和JSON数据绑定
type Resource struct {
	// ID 是 GORM 默认的主键，但为了清晰，我们显式设置 column
	ID            int    `gorm:"primaryKey;column:id;autoIncrement" json:"id"`
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
