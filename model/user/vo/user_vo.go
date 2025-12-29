package vo

import "logo_api/model/user/dto"

// UserInfoResp 响应一个 User 的基本内容，不包含 Password 字段
type UserInfoResp struct {
	ID       int    `json:"id" binding:"required"`
	Username string `json:"username" binding:"required"`
	Status   string `json:"status" binding:"required"` // active/deleted
	// 去除 Password 字段
}

// UserRegisterResp /register 的 data 字段响应内容
type UserRegisterResp struct {
	ID int `json:"id" binding:"required"`
}

// UserLoginResp login 接口的 data 字段响应内容
type UserLoginResp struct {
	ID       int    `json:"id" binding:"required"`
	Username string `json:"username" binding:"required"`
	Status   string `json:"status" binding:"required"`
	Token    string `json:"token" binding:"required"`
}

// UserListResp /list 接口的 data 字段响应内容
type UserListResp struct {
	List       []dto.UserListDTO `json:"list"`
	TotalCount int               `json:"totalCount"` // 符合条件的所有 user 的总量
}
