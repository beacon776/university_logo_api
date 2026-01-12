package vo

import (
	"logo_api/model/resource/do"
)

type ResourceGetResp struct {
	Resource do.Resource `json:"resource"`
	CosURL   string      `json:"cosURL"`
}

type ResourceResp struct {
	// ID 是 GORM 默认的主键，但为了清晰，我们显式设置 column
	ID             int    `json:"id"`
	Title          string `json:"title" binding:"required"`
	ShortName      string `json:"shortName" binding:"required"`
	Name           string `json:"name" binding:"required"`
	Type           string `json:"type" binding:"required"`
	Md5            string `json:"md5"`
	Size           int    `json:"size"`
	LastUpdateTime string `json:"lastUpdateTime"` // 由 *Time.time 转成 string

	IsVector        int    `json:"isVector"`
	IsBitmap        int    `json:"isBitmap"`
	Width           int    `json:"width"`
	Height          int    `json:"height"`
	UsedForEdge     int    `json:"usedForEdge"`
	IsDeleted       int    `json:"isDeleted"`
	BackgroundColor string `json:"backgroundColor"`
	CosURL          string `json:"cosURL"`
}
