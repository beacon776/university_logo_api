package dto

type ResourceGetLogoReq struct {
	Name    string `json:"name"`   // short_name / title sdut or 山东理工大学
	Type    string `json:"type"`   // logo_type png/jpg/svg
	Size    int    `json:"size"`   // logo_size px
	Height  int    `json:"height"` // logo_height px
	Width   int    `json:"width"`  // logo_width px
	BgColor string `json:"bg"`     // bg_color
}
