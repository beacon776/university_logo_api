package model

import (
	"time"
)

const (
	StatusActive   int = 1 // 启用
	StatusDisabled int = 0 // 禁用
)

// Response 是所有 API 调用的统一返回结构。
// 使用 T 泛型（如果你的 Go 版本支持）或 interface{} 来适应 Data 字段
// 如果 Go 版本低于 1.18，使用 interface{}
type Response struct {
	Code    int    `json:"code"`    // HTTP 状态码或自定义业务码 (例如 0 表示成功, >0 表示失败)
	Message string `json:"message"` // 响应的文字信息
	// Data 字段用来承载实际的业务数据（例如 University 结构体或列表）
	Data interface{} `json:"data"` // 实际的业务数据
}
type ReqInsertUniversity struct {
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

type Universities struct {
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

	HasVector        int     `gorm:"column:has_vector" json:"has_vector"`
	MainVectorFormat *string `gorm:"column:main_vector_format" json:"main_vector_format"`
	ResourceCount    int     `gorm:"column:resource_count" json:"resource_count"`
	ComputationId    *int    `gorm:"column:computation_id" json:"computation_id"`

	CreatedTime *time.Time `gorm:"column:created_time" json:"created_time"`
	UpdatedTime *time.Time `gorm:"column:updated_time" json:"updated_time"`
}

/*
// UpdateUniversity 用于请求更新 University 的结构体
type UpdateUniversity struct {
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

	HasVector        int     `gorm:"column:has_vector" json:"has_vector"`
	MainVectorFormat *string `gorm:"column:main_vector_format" json:"main_vector_format"`
	ResourceCount    int     `gorm:"column:resource_count" json:"resource_count"`
	ComputationId    *int    `gorm:"column:computation_id" json:"computation_id"`

	CreatedTime *time.Time `gorm:"column:created_time" json:"created_time"`
}
*/

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
type ReqUser struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type User struct {
	ID       int    `gorm:"primaryKey;column:id" json:"id"`
	Username string `gorm:"column:username" json:"username"`

	// 安全提醒：通常不在结构体中暴露密码，或使用 gorm:"-" 忽略映射
	// 如果需要映射，请使用 gorm:"column:password"，但务必注意安全。
	Password string `gorm:"column:password" json:"-"`

	Status int `gorm:"column:status" json:"status"`
}

// 上面的 Password 字段使用了 json:"-" 来避免序列化到 JSON 响应。
// 如果只是想避免写入/读取数据库，可以使用 gorm:"-"。

// UserListResponse 专门用于列表接口的响应，不包含 Password
type UserListResponse struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Status   int    `json:"status"`
	// 没有 Password 字段
}
