package do

import "time"

type University struct {
	Slug      string `gorm:"column:slug;primaryKey" json:"slug"`
	ShortName string `gorm:"column:short_name" json:"shortName"`
	Title     string `gorm:"column:title" json:"title"`
	// 使用 *string 处理 NULL 字段
	Vis        *string `gorm:"column:vis" json:"vis"`
	Website    string  `gorm:"column:website" json:"website"`
	FullNameEn string  `gorm:"column:full_name_en" json:"fullNameEn"`
	Region     string  `gorm:"column:region" json:"region"`
	Province   string  `gorm:"column:province" json:"province"`
	City       string  `gorm:"column:city" json:"city"`
	Story      *string `gorm:"column:story" json:"story"`

	HasVector        int     `gorm:"column:has_vector" json:"hasVector"`
	MainVectorFormat *string `gorm:"column:main_vector_format" json:"mainVectorFormat"`
	ResourceCount    int     `gorm:"column:resource_count" json:"resourceCount"`
	ComputationID    *int    `gorm:"column:computation_id" json:"computationID"`
	// autoCreateTime 告诉 GORM 在插入时忽略此字段，让数据库生成或由 GORM 生成时间
	CreatedTime *time.Time `gorm:"column:created_time;autoCreateTime" json:"createdTime"`
	// autoUpdateTime 告诉 GORM 在创建和更新时都自动处理
	UpdatedTime *time.Time `gorm:"column:updated_time;autoUpdateTime" json:"updatedTime"`
}
