package service

import (
	"context"
	"go.uber.org/zap"
	"logo_api/dao/mysql"
	"logo_api/dao/redis"
	"logo_api/model/user/dto"
	"logo_api/model/user/vo"
	"time"
)

func GetUserFromName(username string) (vo.UserInfoResp, error) {
	var (
		user dto.UserInfoDTO
		err  error
	)
	if user, err = mysql.GetUserFromName(username); err != nil {
		zap.L().Error("mysql.GetUserFromName() failed", zap.Error(err), zap.String("username", username))
		return vo.UserInfoResp{}, err
	}
	var resultUser vo.UserInfoResp
	resultUser.ID = user.ID
	resultUser.Username = user.Username
	resultUser.Status = user.Status
	return resultUser, err
}

func InsertUser(user dto.UserInsertDTO) error {
	var err error
	if err = mysql.InsertUser(user); err != nil {
		zap.L().Error("mysql.InsertUser() failed", zap.Error(err))
		return err
	}
	zap.L().Info("InsertUser() success", zap.String("username", user.Username))
	return nil
}

func GetUserList(req dto.UserGetListDTO) ([]vo.UserInfoResp, int64, error) {
	var (
		users      []dto.UserInfoDTO
		totalCount int64
		err        error
	)
	if users, totalCount, err = mysql.GetUserList(req); err != nil {
		zap.L().Error("mysql.GetUserList() failed", zap.Error(err))
		return nil, 0, err
	}
	zap.L().Info("GetUserList success", zap.Int("pageSize", req.PageSize), zap.Int("page", req.Page),
		zap.String("keyword", req.Keyword), zap.String("sortBy", req.SortBy), zap.String("sortOrder", req.SortOrder))

	// 把 ID 转成 string 类型防止精度问题
	var resultUsers []vo.UserInfoResp
	for _, user := range users {
		resultUsers = append(resultUsers, vo.UserInfoResp{
			ID:       user.ID,
			Username: user.Username,
			Status:   user.Status,
		})
	}
	return resultUsers, totalCount, nil
}

// StoreUserToken 是 Service 层的方法，用于将用户的当前有效 Token 存储到 Redis
// 实现了单点登录的业务逻辑。
// 它调用了 redis DAO 层中的 SetUserSessionToken 函数。
func StoreUserToken(ctx context.Context, userID int, token string, duration time.Duration) error {
	// 直接调用 DAO 层的封装函数
	return redis.SetUserSessionToken(ctx, userID, token, duration)
}
