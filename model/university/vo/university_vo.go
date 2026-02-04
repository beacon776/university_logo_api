package vo

import (
	"logo_api/model/university/do"
	"time"
)

// UniversityListResp /list 响应
type UniversityListResp struct {
	List       []do.University `json:"list"`
	TotalCount int             `json:"totalCount"` // 所有符合条件的 university 数量
}

type UniversityResp struct {
	Slug       string  `json:"slug"`
	ShortName  string  `json:"shortName"`
	Title      string  `json:"title"`
	Vis        *string `json:"vis"`
	Website    string  `json:"website"`
	FullNameEn string  `json:"fullNameEn"`
	Region     string  `json:"region"`
	Province   string  `json:"province"`
	City       string  `json:"city"`
	Story      *string `json:"story"`

	HasVector        int        `json:"hasVector"`
	MainVectorFormat *string    `json:"mainVectorFormat"`
	ResourceCount    int        `json:"resourceCount"`
	ComputationID    *int       `json:"computationID"`
	CreatedTime      *time.Time `json:"createdTime"`
	UpdatedTime      *time.Time `json:"updatedTime"`
}
