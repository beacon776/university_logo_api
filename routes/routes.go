package routes

import (
	"database/sql"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"logo_api/logger"
	"logo_api/service"
	"logo_api/settings"
	"logo_api/util"
	"path/filepath"
	"strconv"
	"strings"
)

// Setup 注册接口
func Setup() *gin.Engine {
	router := gin.New()
	router.Use(logger.GinLogger(), logger.GinRecovery(true))

	// 创建全局 cosClient 并注入 service
	cosClient, err := util.NewClient(settings.Config.CosConfig)
	if err != nil {
		zap.L().Fatal("util.NewClient failed", zap.Error(err))
	}
	svc := service.NewResourceService(cosClient)

	router.GET("/getLogo/:fullName", getLogoFromNameHandler(svc))

	return router
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

// getLogoFromNameHandler 路由处理函数
func getLogoFromNameHandler(svc *service.ResourceService) gin.HandlerFunc {
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
		width, err = parseQueryInt(c, "w")
		if err != nil {
			respondWithError(c, 400, err.Error(), nil, "parseQueryInt()")
			return
		}
		height, err = parseQueryInt(c, "h")
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
		data, ext, err := svc.GetLogo(fullName, bgColor, size, width, height) // 调用service层中的方法，对参数进行处理，具体的逻辑在 GetLogo 中的方法
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) { // 没查到
				respondWithError(c, 404, "resource not found", nil, "GetLogo")
			} else { // 其他错误
				respondWithError(c, 500, "internal error", err, "GetLogo")
			}
			return
		}

		contentType := getContentType(ext)
		c.Data(200, contentType, data)
	}
}
