package service

import (
	"errors"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"logo_api/dao/mysql"
	"logo_api/model/university/do"
	"logo_api/model/university/dto"
	"logo_api/model/university/vo"
	"logo_api/settings"
)

// GetUniversityFromName 根据单个 name 获取单个 university 对象
func GetUniversityFromName(name string) (vo.UniversityResp, error) {
	var (
		daoUniversity  do.University
		respUniversity vo.UniversityResp
		err            error
	)
	if daoUniversity, err = mysql.GetUniversityByName(name); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			zap.L().Error("GetUniversityFromName() failed because university could not found", zap.String("name", name), zap.Error(err))
			return vo.UniversityResp{}, errors.New("university not found")
		}
		zap.L().Error("mysql.GetUniversityByName() failed", zap.String("name", name), zap.Error(err))
		return vo.UniversityResp{}, err
	}
	respUniversity = vo.UniversityResp{
		Slug:             daoUniversity.Slug,
		ShortName:        daoUniversity.ShortName,
		Title:            daoUniversity.Title,
		Website:          daoUniversity.Website,
		FullNameEn:       daoUniversity.FullNameEn,
		Region:           daoUniversity.Region,
		Province:         daoUniversity.Province,
		City:             daoUniversity.City,
		HasVector:        daoUniversity.HasVector,
		ResourceCount:    daoUniversity.ResourceCount,
		CreatedTime:      daoUniversity.CreatedTime,
		UpdatedTime:      daoUniversity.UpdatedTime,
		Vis:              daoUniversity.Vis,
		Story:            daoUniversity.Story,
		MainVectorFormat: daoUniversity.MainVectorFormat,
		ComputationID:    daoUniversity.ComputationID,
	}
	zap.L().Info("getUniversityFromName() success", zap.String("name", name))
	return respUniversity, nil
}

// InsertUniversity 插入单个 University 对象
func InsertUniversity(reqUniversities []dto.UniversityInsertReq) error {

	daoUniversities := make([]settings.Universities, 0, len(reqUniversities))
	// 检查输入是否为空
	if len(reqUniversities) == 0 {
		zap.L().Warn("this req is empty", zap.Any("reqUniversities", reqUniversities))
		return nil
	}
	for _, reqU := range reqUniversities {
		newDaoUniversity := settings.Universities{
			Slug:          reqU.Slug,
			ShortName:     reqU.ShortName,
			Title:         reqU.Title,
			Website:       reqU.Website,
			FullNameEn:    reqU.FullNameEn,
			Region:        reqU.Region,
			Province:      reqU.Province,
			City:          reqU.City,
			HasVector:     0,
			ResourceCount: 0,
			// 处理可空字段
			Vis:              reqU.Vis,
			Story:            reqU.Story,
			MainVectorFormat: nil,
			ComputationID:    nil,

			// 处理日期字段
			// 让数据库自动更新
		}
		daoUniversities = append(daoUniversities, newDaoUniversity)
	}

	if err := mysql.InsertUniversities(daoUniversities); err != nil {
		zap.L().Error("mysql.InsertUniversities() failed", zap.Error(err))
		return err
	}
	zap.L().Info("InsertUniversities() success", zap.Int("success count", len(daoUniversities)))
	return nil
}

func UpdateUniversities(reqs []dto.UniversityUpdateReq) error {
	if err := mysql.UpdateUniversities(reqs); err != nil {
		zap.L().Error("mysql.UpdateUniversities() failed", zap.Error(err))
		return err
	}
	zap.L().Info("service.UpdateUniversities() success", zap.Int("count", len(reqs)))
	return nil
}
