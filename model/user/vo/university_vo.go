package vo

import "logo_api/model/user/do"

// UniversityListResp /list 响应
type UniversityListResp struct {
	List       []do.University `json:"list"`
	Page       int             `json:"page"`
	PageSize   int             `json:"pageSize"`
	TotalCount int             `json:"totalCount"`
}
