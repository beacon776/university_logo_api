package vo

import "logo_api/model/university/do"

// UniversityListResp /list 响应
type UniversityListResp struct {
	List       []do.University `json:"list"`
	TotalCount int             `json:"totalCount"` // 所有符合条件的 university 数量
}
