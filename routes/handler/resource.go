package handler

import (
	"database/sql"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"logo_api/model"
	"logo_api/model/resource/dto"
	"logo_api/model/resource/vo"
	"logo_api/service"
	"net/http"
	"net/url"
	"strings"
)

func GetResource() gin.HandlerFunc {
	return func(c *gin.Context) {
		name := c.Param("name")
		resource, err := service.GetResourceByName(name)
		if err != nil {
			zap.L().Error("GetResourceByName() failed", zap.Error(err))
			// 资源未找到
			model.Error(c, http.StatusNotFound)
			return
		}
		var resourceVo vo.ResourceGetResp
		resourceVo.Resource = resource
		escapedName := url.PathEscape(name)
		cosURL := fmt.Sprintf("%s/%s/%s", model.BeaconCosPreURL, resource.ShortName, escapedName)
		resourceVo.CosURL = cosURL
		// 3. 成功响应
		model.Success(c, resourceVo)
	}
}

// GetResourceList 参数 name sortBy sortOrder
func GetResourceList() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req dto.ResourceGetListReq
		if err := c.ShouldBindJSON(&req); err != nil {
			zap.L().Error("GetResourceList() ShouldBindJSON failed", zap.Error(err))
			model.Error(c, http.StatusBadRequest)
			return
		}
		zap.L().Info("success get req param", zap.Any("req", req))
		var (
			resourceList []vo.ResourceListResp
			err          error
		)
		if resourceList, err = service.GetResourceList(req); err != nil {
			zap.L().Error("GetResourceList() failed", zap.Error(err))
			model.Error(c, http.StatusInternalServerError)
			return
		}
		zap.L().Info("GetResourceList() success", zap.Int("success count", len(resourceList)))
		model.Success(c, resourceList)
	}
}

func GetLogoFromNameHandler(svc *service.ResourceService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req dto.ResourceGetLogoReq
		if err := c.ShouldBindJSON(&req); err != nil {
			zap.L().Error("GetLogoFromNameHandler() ShouldBind failed", zap.Error(err))
			model.Error(c, http.StatusBadRequest)
			return
		}
		// 加日志看看参数是否解析成功
		zap.L().Info("Received params",
			zap.Any("req params", req))
		data, ext, _, err := svc.GetLogo(req) // 调用service层中的方法，对参数进行处理，具体的逻辑在 GetLogo 中的方法
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) { // 没查到
				zap.L().Error("GetLogoFromNameHandler() err, resource not found ", zap.Error(err))
				model.Error(c, http.StatusNotFound)
				return
			} else { // 其他错误
				zap.L().Error("GetLogoFromNameHandler() err, internal error", zap.Error(err))
				model.Error(c, http.StatusInternalServerError)
				return
			}
		}

		contentType := getContentType(ext)
		c.Header("Content-Disposition", "inline")
		c.Data(200, contentType, data)
	}
}

/*
func InsertResourceHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		var requestData []settings.UniversityResources
		if err := c.ShouldBindJSON(&requestData); err != nil {
			// 如果绑定失败，返回 400 错误
			c.JSON(http.StatusBadRequest, gin.H{
				"code": 400,
				"msg":  fmt.Sprintf("Invalid request data: %s", err.Error()),
				"data": nil,
			})
			return
		}
		err := service.InsertResource(requestData)
		if err != nil {
			// 如果服务层插入失败，返回 500 错误
			c.JSON(http.StatusInternalServerError, gin.H{
				"code": 500,
				"msg":  "insert resource error",
				"data": err,
			})
			return
		}
		// 插入成功，返回 201 Created
		c.JSON(http.StatusCreated, gin.H{
			"code": 201,
			"msg":  "Insert resource success",
			"data": requestData, // 返回插入的资源信息
		})
	}
}*/
/*
func UpdateResourceHandler(svc *service.ResourceService) gin.HandlerFunc {
	return func(c *gin.Context) {

	}
}

// DeleteResourceHandler 注意！删除不是真的删除，而是把资源的 is_deleted 字段设置从默认的 0 设置为 1
func DeleteResourceHandler(svc *service.ResourceService) gin.HandlerFunc {
	return func(c *gin.Context) {

	}
}
*/

// getContentType 根据需求图片类型，获取响应图片的类型
func getContentType(ext string) string {
	switch strings.ToLower(ext) {
	case "png":
		return "image/png"
	case "jpg", "jpeg":
		return "image/jpeg"
	case "svg":
		return "image/svg+xml"
	case "webp":
		return "image/webp"
	default:
		return "application/octet-stream"
	}
}
