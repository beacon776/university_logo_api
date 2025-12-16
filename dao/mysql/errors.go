package mysql

import "errors"

// 导出一个公有的错误变量，用于外部检查
var ErrUserNotFound = errors.New("user not found")
