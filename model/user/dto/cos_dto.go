package dto

// CleanResultDTO Clean 任务的返回结果
type CleanResultDTO struct {
	Total        int      `json:"total"`
	SuccessCount int      `json:"success_count"`
	FailCount    int      `json:"fail_count"`
	FailedPaths  []string `json:"failed_paths"`
}
