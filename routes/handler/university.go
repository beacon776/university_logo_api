package handler

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"logo_api/dao/mysql"
	"logo_api/model"
	"logo_api/model/university/do"
	"logo_api/model/university/dto"
	"logo_api/model/university/vo"
	"logo_api/service"
	"strconv"
	"strings"
)

func GetUniversityFromName() gin.HandlerFunc {
	return func(c *gin.Context) {
		name := c.Param("name") // path param
		// 手动把半角转圆角
		name = strings.ReplaceAll(name, "(", "（")
		name = strings.ReplaceAll(name, ")", "）")
		university, err := service.GetUniversityFromName(name)
		if err != nil {
			zap.L().Error("getUniversityFromName() failed", zap.Error(err))
			// 资源未找到
			model.Error(c, model.CodeNotFound)
			return
		}
		// 找到资源
		zap.L().Info("getUniversityFromName() success", zap.String("name", name))
		// c.JSON() 自动完成结构体到 JSON 的序列化
		model.Success(c, university)
	}
}

func InsertUniversity() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req []dto.UniversityInsertReq
		if err := c.ShouldBindJSON(&req); err != nil {
			zap.L().Error("InsertUniversity() bind error", zap.Error(err))
			model.Error(c, model.CodeServerErr)
			return
		}
		for _, reqUni := range req {
			if _, err := service.GetUniversityFromName(reqUni.Title); err == nil { // err == nil 说明找到了该高校，目前已经重复了
				zap.L().Error("This University Is Exist", zap.String("title", reqUni.Title), zap.Error(err))
				model.Error(c, model.CodeUniversityExist, "This University Is Exist:"+reqUni.Title)
				return
			}
		}

		if err := service.InsertUniversity(req); err != nil {
			zap.L().Error("InsertUniversity() insert error", zap.Error(err))
			model.Error(c, model.CodeServerErr)
			return
		}
		zap.L().Info("InsertUniversity() success", zap.Int("success count", len(req)))
		model.SuccessEmpty(c, "Successfully inserted "+strconv.Itoa(len(req))+" universities.")
	}
}

func GetUniversityList() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req dto.UniversityGetListReq
		// 如果 Request Body 不为空，才进行 JSON 绑定
		if c.Request.ContentLength > 0 {
			if err := c.ShouldBindJSON(&req); err != nil { // 使用结构体tag 检验 SortBy+SortOrder 参数范围合法性
				zap.L().Error("GetUniversityList() bind error", zap.Error(err))
				model.Error(c, model.CodeInvalidParam)
				return
			}
		}
		// 允许空参数的存在
		zap.L().Info("receive request body", zap.Any("req", req))
		// 设置默认值
		if req.Page <= 0 {
			req.Page = 1
		}
		if req.PageSize <= 0 {
			req.PageSize = 10
		}
		if req.SortBy == "" {
			req.SortBy = "title"
		}
		if req.SortOrder == "" {
			req.SortOrder = "asc"
		}
		// 半角转圆角
		req.Keyword = strings.ReplaceAll(req.Keyword, "(", "（")
		req.Keyword = strings.ReplaceAll(req.Keyword, ")", "）")

		// page、pageSize 参数范围有效性检查
		// 确保 page 和 pageSize 是正数
		if req.Page <= 0 || req.PageSize <= 0 {
			model.Error(c, model.CodeInvalidParam, "Invalid page parameter, page and pageSize must be greater than 0.")
			return
		}
		universities, totalCount, err := mysql.GetUniversityList(req)
		if err != nil {
			zap.L().Error("svc.GetUniversityList() failed", zap.Error(err), zap.Int("page", req.Page), zap.Int("pageSize", req.PageSize), zap.String("keyword", req.Keyword))
			model.Error(c, model.CodeServerErr)
			return
		}

		// 根据 totalCount 进行判断
		if totalCount == 0 && req.Keyword != "" {
			// 如果 totalCount 为 0 且用户使用了 keyword 进行搜索

			// 保持 200，但修改 Message
			// 客户端看到 200 状态码知道接口运行正常，但可以根据 Message 提示用户
			message := fmt.Sprintf("No universities found matching keyword '%s'", req.Keyword)
			zap.L().Info(message, zap.Int("page", req.Page), zap.Int("pageSize", req.PageSize), zap.Int64("totalCount", totalCount), zap.String("keyword", req.Keyword))
			var resp vo.UniversityListResp
			resp.List = universities
			resp.TotalCount = int(totalCount)
			model.Success(c, resp, message)
			return
		}
		// 查找成功
		zap.L().Info("Success get university list", zap.Int("page", req.Page), zap.Int("pageSize", req.PageSize), zap.Int64("totalCount", totalCount), zap.String("keyword", req.Keyword))
		var resp vo.UniversityListResp
		resp.List = universities
		resp.TotalCount = int(totalCount)
		model.Success(c, resp)
	}
}

func UpdateUniversities() gin.HandlerFunc {
	return func(c *gin.Context) {
		var universities []do.University
		if err := c.ShouldBindJSON(&universities); err != nil {
			zap.L().Error("c.ShouldBindJSON(&universities) failed", zap.Error(err))
			model.Error(c, model.CodeInvalidParam)
			return
		}
		if err := service.UpdateUniversities(universities); err != nil {
			zap.L().Error("svc.UpdateUniversities() failed", zap.Error(err))
			model.Error(c, model.CodeServerErr)
			return
		}
		// 成功
		zap.L().Info("Success update universities", zap.Int("count", len(universities)))
		model.Success(c, universities)
	}
}
