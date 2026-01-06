package dto

type ResourceGetLogoReq struct {
	Name    string `json:"name"`   // short_name / title sdut or 山东理工大学
	Type    string `json:"type"`   // logo_type png/jpg/svg
	Size    int    `json:"size"`   // logo_size px
	Height  int    `json:"height"` // logo_height px
	Width   int    `json:"width"`  // logo_width px
	BgColor string `json:"bg"`     // bg_color
}

type ResourceGetListReq struct {
	Name      string `json:"name"` // 模糊匹配 title 或者 short_name
	SortBy    string `json:"sortBy" binding:"omitempty,oneof=id name size type lastUpdateTime"`
	SortOrder string `json:"sortOrder" binding:"omitempty,oneof=asc desc"`
}
