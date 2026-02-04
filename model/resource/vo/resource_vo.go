package vo

import (
	"logo_api/model/resource/dto"
)

type ResourceResp struct {
	List       []dto.ResourceInfoDTO `json:"list"`
	TotalCount int                   `json:"totalCount"`
}
