package dto

// UniversityInsertReq 接收 /insert 路由的请求参数
type UniversityInsertReq struct {
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
}

// UniversityGetListReq /university/list 请求参数
type UniversityGetListReq struct {
	Page      int    `json:"page"`
	PageSize  int    `json:"pageSize"`
	Keyword   string `json:"keyword"`
	SortBy    string `json:"sortBy" binding:"omitempty,oneof=slug title createTime updateTime"`
	SortOrder string `json:"sortOrder" binding:"omitempty,oneof=asc desc"`
}
