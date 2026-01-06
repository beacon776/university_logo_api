package model

// 用户状态码
const (
	StatusActive  int = 1 // 启用
	StatusDeleted int = 0 // 禁用
	StatusError   int = -1
)
const (
	StatusActiveStr  string = "active"
	StatusDeletedStr string = "deleted"
	StatusErrorStr   string = "error"
)

// 资源删除码
const (
	ResourceIsActive  int = 0
	ResourceIsDeleted int = 1
)

// 自定义业务状态码
const (
	CodeSuccess         = 200 // 成功
	CodeInvalidParam    = 400 // 参数错误
	CodeUnauthorized    = 401 // 未登录
	CodeNotFound        = 404 // 资源不存在
	CodeUserExist       = 409 // 用户已存在
	CodeUniversityExist = 410 // 高校已存在
	CodeServerErr       = 500 // 服务器内部错误
)

const (
	CodeSuccessStr         string = "Success"
	CodeInvalidParamStr    string = "Invalid Param"
	CodeUnauthorizedStr    string = "Unauthorized"
	CodeNotFoundStr        string = "Resource Not Found"
	CodeUserExistStr       string = "User Already Exists"
	CodeUniversityExistStr string = "University Already Exists"
	CodeServerErrStr       string = "Internal Server Error"
)

// 对应描述
var codeMsg = map[int]string{
	CodeSuccess:         CodeSuccessStr,
	CodeInvalidParam:    CodeInvalidParamStr,
	CodeUnauthorized:    CodeUnauthorizedStr,
	CodeNotFound:        CodeNotFoundStr,
	CodeUserExist:       CodeUserExistStr,
	CodeUniversityExist: CodeUniversityExistStr,
	CodeServerErr:       CodeServerErrStr,
	StatusActive:        StatusActiveStr,
	StatusDeleted:       StatusDeletedStr,
	StatusError:         StatusErrorStr,
}

func GetMsg(code int) string {
	return codeMsg[code]
}

const (
	BeaconCosPreURL string = "https://shaly-1353984479.cos.ap-shanghai.myqcloud.com/beacon/downloads"
)
