package handler

import (
	"database/sql"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"logo_api/model"
	"logo_api/service"
	"logo_api/settings"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
)

func GetUniversityResource(svc *service.ResourceService) gin.HandlerFunc {
	return func(c *gin.Context) {
		name := c.Param("name")
		resource, err := svc.GetUniversityResourceFromName(name)
		if err != nil {
			zap.L().Error("GetUniversityResourceFromName() failed", zap.Error(err))
			// 资源未找到 (假设 Service 层返回特定错误或 nil)
			c.JSON(http.StatusNotFound, model.Response{
				Code:    http.StatusNotFound,
				Message: "Resource not exist",
				Data:    nil,
			})
			return
		}
		// 3. 成功响应
		response := model.Response{
			Code:    http.StatusOK, // 200
			Message: "success",
			Data:    resource, // 自动序列化
		}

		// 使用 c.JSON() 自动完成结构体到 JSON 的序列化
		c.JSON(http.StatusOK, response)
	}
}

func GetLogoFromNameHandler(svc *service.ResourceService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var (
			size   int
			width  int
			height int
			err    error
		)

		fullName := c.Param("fullName")
		ext := filepath.Ext(fullName)
		ext = ext[1:] // 输入参数文件类型

		// 解析 query 参数
		bgColor := c.DefaultQuery("bg", "") // 例如 "#FFFFFF"

		size, err = parseQueryInt(c, "size")
		if err != nil {
			respondWithError(c, 400, err.Error(), nil, "parseQueryInt()")
			return
		}
		width, err = parseQueryInt(c, "width")
		if err != nil {
			respondWithError(c, 400, err.Error(), nil, "parseQueryInt()")
			return
		}
		height, err = parseQueryInt(c, "height")
		if err != nil {
			respondWithError(c, 400, err.Error(), nil, "parseQueryInt()")
			return
		}

		// 加日志看看参数是否解析成功
		zap.L().Info("Received params",
			zap.Int("size", size),
			zap.Int("width", width),
			zap.Int("height", height),
			zap.String("bg", bgColor),
		)
		data, ext, _, err := svc.GetLogo(fullName, bgColor, size, width, height) // 调用service层中的方法，对参数进行处理，具体的逻辑在 GetLogo 中的方法
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) { // 没查到
				respondWithError(c, 404, "resource not found", nil, "GetLogo")
			} else { // 其他错误
				respondWithError(c, 500, "internal error", err, "GetLogo")
			}
			return
		}

		contentType := getContentType(ext)
		c.Header("Content-Disposition", "inline")
		c.Data(200, contentType, data)
	}
}

func InsertResourceHandler(svc *service.ResourceService) gin.HandlerFunc {
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
		err := svc.InsertResource(requestData)
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
}

func UpdateResourceHandler(svc *service.ResourceService) gin.HandlerFunc {
	return func(c *gin.Context) {

	}
}

// DeleteResourceHandler 注意！删除不是真的删除，而是把资源的 is_deleted 字段设置从默认的 0 设置为 1
func DeleteResourceHandler(svc *service.ResourceService) gin.HandlerFunc {
	return func(c *gin.Context) {

	}
}

// respondWithError 用户传参失误时，code为400，err为nil; 系统内部错误时，code为500，err为真实err
func respondWithError(c *gin.Context, code int, msg string, err error, location string) {
	if err != nil {
		zap.L().Error(
			location,
			zap.String("msg", msg),
			zap.Error(err))
	}
	c.JSON(code, gin.H{
		"code": code,
		"msg":  msg,
	})
}

// parseQueryInt 解析输入url中的参数
func parseQueryInt(c *gin.Context, key string) (int, error) {
	valStr := c.DefaultQuery(key, "")
	if valStr == "" {
		return 0, nil
	}
	val, err := strconv.Atoi(valStr)
	if err != nil {
		zap.L().Error("parseQueryInt() err:", zap.Error(err))
		return 0, fmt.Errorf("invalid %s: %v", key, err)
	}
	return val, nil
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
