package service

import (
	"go.uber.org/zap"
	"logo_api/dao/mysql"
	"logo_api/model/university/do"
	"logo_api/model/university/dto"
	"logo_api/settings"
)

// GetUniversityFromName 根据单个 name 获取单个 university 对象
func GetUniversityFromName(name string) (do.University, error) {
	var (
		daoUniversity  do.University
		respUniversity do.University
		err            error
	)
	if daoUniversity, err = mysql.GetUniversityByName(name); err != nil {
		zap.L().Error("mysql.GetUniversityByName() failed", zap.Error(err))
		return do.University{}, err
	}
	respUniversity = do.University{
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

func UpdateUniversities(universities []do.University) error {
	if err := mysql.UpdateUniversities(universities); err != nil {
		zap.L().Error("mysql.UpdateUniversities() failed", zap.Error(err))
		return err
	}
	zap.L().Info("UpdateUniversities success", zap.Int("count", len(universities)))
	return nil
}
