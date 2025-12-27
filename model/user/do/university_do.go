package do

import "time"

type University struct {
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
	ComputationID    *int    `gorm:"column:computation_id" json:"computation_id"`

	CreatedTime *time.Time `gorm:"column:created_time" json:"created_time"`
	UpdatedTime *time.Time `gorm:"column:updated_time" json:"updated_time"`
}
