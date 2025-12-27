package model

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

// Response 统一响应结构
type Response[T any] struct {
	Code    int    `json:"code"`    // 业务状态码
	Message string `json:"message"` // 提示信息
	Data    T      `json:"data"`    // 业务数据
}

// Success 成功返回
func Success[T any](c *gin.Context, data T, customMsg ...string) {
	msg := GetMsg(CodeSuccess)
	if len(customMsg) > 0 {
		msg = customMsg[0]
	}
	c.JSON(http.StatusOK, Response[T]{
		Code:    CodeSuccess,
		Message: msg,
		Data:    data,
	})
}

// SuccessEmpty 成功返回，但不带 Data
func SuccessEmpty(c *gin.Context, customMsg ...string) {
	msg := GetMsg(CodeSuccess)
	if len(customMsg) > 0 {
		msg = customMsg[0]
	}
	c.JSON(http.StatusOK, Response[interface{}]{
		Code:    CodeSuccess,
		Message: msg,
		Data:    nil,
	})
}

// Error 错误返回
func Error(c *gin.Context, code int, customMsg ...string) {
	msg := GetMsg(code)
	if len(customMsg) > 0 {
		msg = customMsg[0] // 如果有自定义消息则覆盖
	}

	// 根据业务码映射合适的 HTTP 状态码，或者统一返回 200
	c.JSON(http.StatusOK, Response[interface{}]{
		Code:    code,
		Message: msg,
		Data:    nil,
	})
}
