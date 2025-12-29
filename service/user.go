package service

import (
	"go.uber.org/zap"
	"logo_api/dao/mysql"
	"logo_api/model"
	"logo_api/model/user/do"
	"logo_api/model/user/dto"
	"logo_api/model/user/vo"
)

func GetUserFromName(username string) (vo.UserInfoResp, error) {
	var (
		doUser do.UserDO
		voUser vo.UserInfoResp
		err    error
	)
	if doUser, err = mysql.GetUserFromName(username); err != nil {
		zap.L().Error("mysql.GetUserFromName() failed", zap.Error(err), zap.String("username", username))
		return vo.UserInfoResp{}, err
	}
	voUser.ID = doUser.ID
	voUser.Username = doUser.Username
	if doUser.Status == model.StatusActive {
		voUser.Status = model.StatusActiveStr
	} else if doUser.Status == model.StatusDeleted {
		voUser.Status = model.StatusDeletedStr
	}
	return voUser, err
}

func GetUserList(req dto.UserGetListReq) ([]dto.UserListDTO, int64, error) {
	var (
		dtoUsers   []dto.UserListDTO
		totalCount int64
		err        error
	)
	if dtoUsers, totalCount, err = mysql.GetUserList(req); err != nil {
		zap.L().Error("mysql.GetUserList() failed", zap.Error(err))
		return nil, 0, err
	}
	zap.L().Info("GetUserList success", zap.Int("pageSize", req.PageSize), zap.Int("page", req.Page),
		zap.String("keyword", req.Keyword), zap.String("sortBy", req.SortBy), zap.String("sortOrder", req.SortOrder))
	return dtoUsers, totalCount, nil
}

func InsertUser(req dto.UserInsertReq) (int, error) {
	var (
		insertId int
		err      error
	)
	if insertId, err = mysql.InsertUser(req); err != nil {
		zap.L().Error("mysql.InsertUser() failed", zap.Error(err))
		return -1, err
	}
	zap.L().Info("InsertUser() success", zap.String("username", req.Username))
	return insertId, nil
}
