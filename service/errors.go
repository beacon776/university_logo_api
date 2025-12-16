package service

import "errors"

// ErrSessionNotFound 是 Service 层定义的错误，表示会话/Token 在存储中不存在。
var ErrSessionNotFound = errors.New("user session or token not found")
