package dto

import (
	"logo_api/model"
	"logo_api/model/resource/do"
	"logo_api/util"
	"mime/multipart"
)

type ResourceInfoDTO struct {
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
type ResourceGetLogoReq struct {
	Name    string `json:"name" binding:"required"`     // short_name / title sdut or 山东理工大学
	Type    string `json:"type" binding:"required"`     // logo_type png/jpg/svg
	Size    int    `json:"size" binding:"omitempty"`    // logo_size px
	Height  int    `json:"height" binding:"omitempty"`  // logo_height px
	Width   int    `json:"width" binding:"omitempty"`   // logo_width px
	BgColor string `json:"bgColor" binding:"omitempty"` // bg_color
}

type ResourceGetListReq struct {
	Name      string `json:"name" binding:"required"` // 模糊匹配 title 或者 short_name
	SortBy    string `json:"sortBy" binding:"omitempty,oneof=id name size type lastUpdateTime"`
	SortOrder string `json:"sortOrder" binding:"omitempty,oneof=asc desc"`
}

type ResourceGetReq struct {
	Name string `json:"name"` // 指定资源名称
}
type ResourceDelReq struct {
	Name      string `json:"name"`      // 指定资源名称
	Title     string `json:"title"`     // 指定资源所属高校中文全称(防止出现多个资源重名的情况)
	ShortName string `json:"shortName"` // 指定资源所属高校英文简称
}

type ResourceRecoverReq struct {
	Name      string `json:"name"`
	Title     string `json:"title"`     // 指定资源所属高校中文全称(防止出现多个资源重名的情况)
	ShortName string `json:"shortName"` // 指定资源所属高校英文简称
}

type ResourceInsertReq struct {
	File            *multipart.FileHeader `form:"file" binding:"required"` // 文件流
	Title           string                `form:"title" binding:"required"`
	ShortName       string                `form:"shortName" binding:"required"`
	Name            string                `form:"name" binding:"required"`
	Type            string                `form:"type" binding:"required"`
	UsedForEdge     int                   `form:"usedForEdge" binding:"oneof=0 1"`
	BackgroundColor string                `form:"backgroundColor" binding:"omitempty"`
}

func (req ResourceInsertReq) ToEntity() (*do.Resource, error) {
	// 1. 计算 MD5 (需要处理文件流)
	md5Val, err := util.CalculateMD5(req.File)
	if err != nil {
		return nil, err
	}

	// 2. 获取图片信息 (宽高、类型)
	w, h, isVec, isBit := util.GetImageInfo(req.File)

	return &do.Resource{
		Title:           req.Title,
		ShortName:       req.ShortName,
		Name:            req.Name,
		Type:            req.Type,
		Md5:             md5Val,
		Size:            int(req.File.Size),
		Width:           w,
		Height:          h,
		IsVector:        isVec,
		IsBitmap:        isBit,
		UsedForEdge:     req.UsedForEdge,
		BackgroundColor: req.BackgroundColor,
		IsDeleted:       model.ResourceIsActive,
	}, nil
}
