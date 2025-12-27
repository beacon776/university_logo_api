package vo

// UserInfoResp 响应一个 User 的基本内容，不包含 Password 字段
type UserInfoResp struct {
	ID       int    `json:"id" binding:"required"`
	Username string `json:"username" binding:"required"`
	Status   int    `json:"status" binding:"required"`
	// 去除 Password 字段
}

// UserLoginResp login 接口的 data 字段响应内容
type UserLoginResp struct {
	ID       int    `json:"id" binding:"required"`
	Username string `json:"username" binding:"required"`
	Status   int    `json:"status" binding:"required"`
	Token    string `json:"token" binding:"required"`
}

// UserListResp /list 接口的 data 字段响应内容
type UserListResp struct {
	List       []UserInfoResp `json:"list"`
	Page       int            `json:"page"`
	PageSize   int            `json:"pageSize"`
	TotalCount int            `json:"totalCount"`
}
