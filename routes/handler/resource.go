package handler

import (
	"database/sql"
	"errors"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"logo_api/model"
	"logo_api/model/resource/dto"
	"logo_api/model/resource/vo"
	"logo_api/service"
	"logo_api/util"
	"net/http"
	"strings"
)

func GetResources() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req []dto.ResourceGetReq
		if err := c.ShouldBindJSON(&req); err != nil {
			zap.L().Error("c.ShouldBindJSON(&req failed", zap.Error(err))
			model.Error(c, http.StatusBadRequest)
			return
		}
		var names []string
		for _, name := range req {
			names = append(names, name.Name)
		}
		resource, err := service.GetResources(names)
		if err != nil {
			zap.L().Error("GetResourceByName() failed", zap.Error(err))
			// 资源未找到
			model.Error(c, http.StatusNotFound)
			return
		}
		// 3. 成功响应
		model.Success(c, resource)
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
			resourceList []vo.ResourceResp
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

func InsertResource() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req dto.ResourceInsertReq
		if err := c.ShouldBind(&req); err != nil {
			zap.L().Error("InsertResource() ShouldBind failed", zap.Any("req", req), zap.Error(err))
			model.Error(c, http.StatusBadRequest)
			return
		}
		req.BackgroundColor = util.NormalizeColor(req.BackgroundColor)
		if err := service.InsertResource(c.Request.Context(), req); err != nil {
			zap.L().Error("InsertResource() failed", zap.Error(err))
			model.Error(c, http.StatusInternalServerError)
			return
		}
		zap.L().Info("success insert", zap.Any("req", req))
		model.SuccessEmpty(c, "success")
	}
}

// DelResources 把资源的 is_deleted 字段设置从默认的 0 设置为 1
func DelResources() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req []dto.ResourceDelReq
		if err := c.ShouldBindJSON(&req); err != nil {
			zap.L().Error("DelResource() ShouldBindJSON failed", zap.Error(err))
			model.Error(c, http.StatusBadRequest)
			return
		}
		var names []string
		for _, name := range req {
			names = append(names, name.Name)
		}
		if _, err := service.GetResources(names); err != nil {
			zap.L().Error("DelResource() failed because at least one resource couldn't be found", zap.Strings("names", names), zap.Error(err))
			model.Error(c, http.StatusInternalServerError)
			return
		}
		if err := service.DelResources(names); err != nil {
			zap.L().Error("DelResource() failed", zap.Strings("names", names), zap.Error(err))
			model.Error(c, http.StatusInternalServerError)
			return
		}
		model.SuccessEmpty(c, "Success")
	}
}

func RecoverResources() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req []dto.ResourceRecoverReq
		if err := c.ShouldBindJSON(&req); err != nil {
			zap.L().Error("RecoverResources() ShouldBindJSON failed", zap.Error(err))
			model.Error(c, http.StatusBadRequest)
			return
		}
		var names []string
		for _, name := range req {
			names = append(names, name.Name)
		}
		if err := service.RecoverResources(names); err != nil {
			zap.L().Error("RecoverResources() failed", zap.Strings("names", names), zap.Error(err))
			model.Error(c, http.StatusInternalServerError)
			return
		}
		model.SuccessEmpty(c, "Success")
	}
}

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
