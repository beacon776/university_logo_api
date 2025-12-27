package dto

// UserRegisterReq 接收 /user/register 路由的请求参数
type UserRegisterReq struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// UserLoginReq 接收 /user/login 路由的请求参数
type UserLoginReq struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// UserGetListDTO /user/list 请求参数
type UserGetListDTO struct {
	Page      int    `json:"page"`
	PageSize  int    `json:"pageSize"`
	Keyword   string `json:"keyword"`
	SortBy    string `json:"sortBy"`
	SortOrder string `json:"sortOrder"`
}

// UserInsertDTO 插入 user 表所需字段，不包括 id，id 是数据库自增字段。
type UserInsertDTO struct {
	Username string `gorm:"column:username" json:"username"`
	// json:"-" 忽略 json 映射
	Password string `gorm:"column:password" json:"-"`
	Status   int    `gorm:"column:status" json:"status"`
}

// UserInfoDTO 用户相关信息，保留 Password
type UserInfoDTO struct {
	ID       int    `gorm:"column:id" json:"id"`
	Username string `gorm:"column:username" json:"username"`
	Password string `gorm:"column:password" json:"-"`
	Status   int    `gorm:"column:status" json:"status"`
}
