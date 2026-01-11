package dto

import (
	"logo_api/model"
	"logo_api/model/resource/do"
	"logo_api/util"
	"mime/multipart"
)

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

type ResourceGetReq struct {
	Name string `json:"name"` // 指定资源名称
}
type ResourceDelReq struct {
	Name string `json:"name"` // 指定资源名称
}

type ResourceRecoverReq struct {
	Name string `json:"name"`
}

type ResourceInsertReq struct {
	File            *multipart.FileHeader `form:"file" binding:"required"` // 文件流
	Title           string                `form:"title" binding:"required"`
	ShortName       string                `form:"shortName" binding:"required"`
	Name            string                `form:"name" binding:"required"`
	Type            string                `form:"type" binding:"required"`
	UsedForEdge     int                   `form:"usedForEdge" binding:"required"`
	BackgroundColor string                `form:"backgroundColor" binding:"required"`
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
