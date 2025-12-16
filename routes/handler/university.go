package handler

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"logo_api/model"
	"logo_api/service"
	"net/http"
	"strconv"
	"strings"
)

func GetUniversityFromName(svc *service.ResourceService) gin.HandlerFunc {
	return func(c *gin.Context) {
		name := c.Param("name")
		// 手动把半角转圆角
		name = strings.ReplaceAll(name, "(", "（")
		name = strings.ReplaceAll(name, ")", "）")
		university, err := svc.GetUniversityFromName(name)
		if err != nil {
			zap.L().Error("getUniversityFromName() failed", zap.Error(err))
			// 资源未找到 (Service 层返回特定错误或 nil)
			c.JSON(http.StatusNotFound, model.Response{
				Code:    http.StatusNotFound,
				Message: "Resource not exist",
				Data:    nil,
			})
			return
		}
		zap.L().Info("getUniversityFromName() success", zap.String("name", name))
		// c.JSON() 自动完成结构体到 JSON 的序列化
		c.JSON(http.StatusOK, model.Response{
			Code:    http.StatusOK, // 200
			Message: "success",
			Data:    university, // 自动序列化为 {"slug": "...", "title": "...", ...}
		})
	}
}

func InsertUniversity(svc *service.ResourceService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var u []model.ReqInsertUniversity
		if err := c.ShouldBindJSON(&u); err != nil {
			zap.L().Error("InsertUniversity() bind error", zap.Error(err))
			c.JSON(http.StatusBadRequest, model.Response{
				Code:    http.StatusBadRequest,
				Message: err.Error(),
			})
			return
		}
		if err := svc.InsertUniversity(u); err != nil {
			zap.L().Error("InsertUniversity() insert error", zap.Error(err))
			c.JSON(http.StatusInternalServerError, model.Response{
				Code:    http.StatusInternalServerError,
				Message: err.Error(),
			})
			return
		}
		c.JSON(http.StatusOK, model.Response{
			Code:    http.StatusOK,
			Message: "Successfully inserted " + strconv.Itoa(len(u)) + " universities.",
		})
	}
}

func GetUniversityList(svc *service.ResourceService) gin.HandlerFunc {
	return func(c *gin.Context) {
		pageStr := c.DefaultQuery("page", "1")
		pageSizeStr := c.DefaultQuery("pageSize", "10")
		keyword := c.DefaultQuery("keyword", "")

		// 手动把半角转圆角
		keyword = strings.ReplaceAll(keyword, "(", "（")
		keyword = strings.ReplaceAll(keyword, ")", "）")

		sortBy := c.DefaultQuery("sortBy", "id")
		sortOrder := c.DefaultQuery("sortOrder", "asc")

		// 统一处理空值和默认值(调 apifox 时，即使字段为空，也会传入空字符串，因此需要手动处理空字符串)
		if pageStr == "" {
			pageStr = "1"
		}
		if pageSizeStr == "" {
			pageSizeStr = "10"
		}
		if sortBy == "" {
			sortBy = "title"
		}
		if sortOrder == "" {
			sortOrder = "asc"
		}

		// 将字符串转换为整数 (int)
		page, err := strconv.Atoi(pageStr)
		if err != nil {
			zap.L().Error("strconv.Atoi(pageStr) Error", zap.String("pageStr", pageStr), zap.Error(err), zap.String("keyword", keyword))
			// 处理错误：如果转换失败，返回 400 错误给客户端
			c.JSON(400, model.Response{
				Code:    400,
				Message: "Invalid page parameter",
				Data:    nil,
			})
			return
		}

		pageSize, err := strconv.Atoi(pageSizeStr)
		if err != nil {
			zap.L().Error("strconv.Atoi(pageSizeStr) Error", zap.String("pageSizeStr", pageSizeStr), zap.Error(err), zap.String("keyword", keyword))
			// 处理错误：如果转换失败，返回 400 错误给客户端
			c.JSON(400, model.Response{
				Code:    400,
				Message: "Invalid page parameter",
				Data:    nil,
			})
			return
		}

		// 参数有效性检查（推荐）
		// 确保 page 和 pageSize 是正数
		if page <= 0 || pageSize <= 0 {
			c.JSON(400, model.Response{
				Code:    400,
				Message: "Invalid page parameter, page and pageSize must be greater than 0.",
				Data:    nil,
			})
			return
		}
		universities, totalCount, err := svc.GetUniversityList(page, pageSize, keyword, sortBy, sortOrder)
		if err != nil {
			zap.L().Error("svc.GetUniversityList() failed", zap.Error(err), zap.Int("page", page), zap.Int("pageSize", pageSize), zap.String("keyword", keyword))
			c.JSON(http.StatusInternalServerError, model.Response{
				Code:    http.StatusInternalServerError,
				Message: err.Error(),
				Data:    nil,
			})
			return
		}

		// 根据 totalCount 进行判断
		if totalCount == 0 && keyword != "" {
			// 如果 totalCount 为 0 且用户使用了 keyword 进行搜索

			// 保持 200，但修改 Message
			// 客户端看到 200 状态码知道接口运行正常，但可以根据 Message 提示用户
			message := fmt.Sprintf("No universities found matching keyword '%s'", keyword)
			zap.L().Info(message, zap.Int("page", page), zap.Int("pageSize", pageSize), zap.Int64("totalCount", totalCount), zap.String("keyword", keyword))

			c.JSON(http.StatusOK, model.Response{
				Code:    http.StatusOK,
				Message: message,
				Data: gin.H{
					"list":       universities, // list 为空 []
					"page":       page,
					"pageSize":   pageSize,
					"totalCount": totalCount, // totalCount 为 0
				},
			})
			return
		}

		zap.L().Info("Success get university list", zap.Int("page", page), zap.Int("pageSize", pageSize), zap.Int64("totalCount", totalCount), zap.String("keyword", keyword))
		c.JSON(200, model.Response{
			Code:    http.StatusOK,
			Message: "Success get university list",
			Data: gin.H{
				"list":       universities,
				"page":       page,
				"pageSize":   pageSize,
				"totalCount": totalCount,
			},
		})
	}
}
