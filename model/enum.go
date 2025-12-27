package model

// 用户状态码
const (
	StatusActive   int = 1 // 启用
	StatusDisabled int = 0 // 禁用
)

// 自定义业务状态码
const (
	CodeSuccess      = 200 // 成功
	CodeInvalidParam = 400 // 参数错误
	CodeUnauthorized = 401 // 未登录
	CodeNotFound     = 404 // 资源不存在
	CodeUserExist    = 409 // 用户已存在
	CodeServerErr    = 500 // 服务器内部错误
)

// 对应描述
var codeMsg = map[int]string{
	CodeSuccess:      "Success",
	CodeInvalidParam: "Invalid Parameters",
	CodeUnauthorized: "Unauthorized",
	CodeNotFound:     "Resource Not Found",
	CodeUserExist:    "Username Already Exists",
	CodeServerErr:    "Internal Server Error",
}

func GetMsg(code int) string {
	return codeMsg[code]
}
