package main

/*
import (
	"database/sql"
	"encoding/json"
	"fmt"
	"go.uber.org/zap"
	"logo_api/dao/mysql"
	"logo_api/logger"
	"logo_api/settings" // 假设 settings 包已导入
	"os"
	"path/filepath"
	"reflect" // 引入反射包用于调试
	"regexp"
	// 注意：time 包现在被隐式导入，因为 settings.UniversityResources 使用了 *time.Time
)

// --- 辅助函数：类型安全的 JSON 解析 ---

// unmarshalUniversityResources 专门读取并解析 university_resources.json 文件。
func unmarshalUniversityResources(filePath string) ([]settings.UniversityResources, error) {
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		zap.L().Error("无法获取资源的绝对路径", zap.Error(err))
		return nil, err
	}

	data, err := os.ReadFile(absPath)
	if err != nil {
		zap.L().Error("读取资源文件失败", zap.Error(err), zap.String("绝对路径", absPath))
		return nil, err
	}

	if len(data) == 0 {
		zap.L().Warn("资源文件内容为空", zap.String("绝对路径", absPath))
		return nil, nil
	}

	// 步骤 1: 检查是否存在空字符串 "" (防止 time.Parse 失败)
	emptyDateRegexCheck := regexp.MustCompile(`"last_update_time"\s*:\s*""`)
	if emptyDateRegexCheck.Match(data) {
		// 致命错误处理
		zap.L().Fatal("致命错误：'last_update_time' 字段发现空字符串 \"\"，程序已终止。",
			zap.String("文件", filePath),
			zap.String("检查项", "last_update_time: \"\""))
		return nil, nil
	}

	// 步骤 2: 处理 "last_update_time": "YYYY-MM-DD" 的情况
	// 替换为 RFC3339 格式所需的 "YYYY-MM-DDT00:00:00Z"
	dateRegex := regexp.MustCompile(`("last_update_time"\s*:\s*)"(\d{4}-\d{2}-\d{2})"`)
	processedData := dateRegex.ReplaceAll(data, []byte(`$1"$2T00:00:00Z"`))

	zap.L().Info("已对 university_resources.json 文件进行日期格式预处理",
		zap.Int("原始长度", len(data)),
		zap.Int("处理后长度", len(processedData)))

	// 核心修复：直接声明目标类型，消除 interface{} 歧义
	var resources []settings.UniversityResources

	// 增加反射调试信息！！！这是我们定位问题的关键
	targetType := reflect.TypeOf(&resources)
	zap.L().Debug("JSON 反序列化目标类型检查",
		zap.String("目标类型名", targetType.String()),
		zap.String("目标种类", targetType.Kind().String()),
		zap.String("元素类型名", targetType.Elem().String()),       // 切片类型名 []settings.UniversityResources
		zap.String("元素种类", targetType.Elem().Kind().String()), // 切片 Kind: slice
	)

	err = json.Unmarshal(processedData, &resources)
	if err != nil {
		zap.L().Error("解析 university_resources.json 数据失败", zap.Error(err), zap.String("绝对路径", absPath))
		return nil, err
	}

	return resources, nil
}

// unmarshalUniversities 专门读取并解析 universities.json 文件。
func unmarshalUniversities(filePath string) ([]settings.InitUniversities, error) {
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		zap.L().Error("无法获取大学列表的绝对路径", zap.Error(err))
		return nil, err
	}

	data, err := os.ReadFile(absPath)
	if err != nil {
		zap.L().Error("读取大学列表文件失败", zap.Error(err), zap.String("绝对路径", absPath))
		return nil, err
	}

	if len(data) == 0 {
		zap.L().Warn("大学列表文件内容为空", zap.String("绝对路径", absPath))
		return nil, nil
	}

	// 核心修复：直接声明目标类型，消除 interface{} 歧义
	var universities []settings.InitUniversities
	err = json.Unmarshal(data, &universities)
	if err != nil {
		zap.L().Error("解析 universities.json 数据失败", zap.Error(err), zap.String("绝对路径", absPath))
		return nil, err
	}

	return universities, nil
}

// main 函数现在使用类型安全的函数进行数据导入
func main() {
	// 1.加载配置
	if err := settings.Init(); err != nil {
		panic(fmt.Sprintf("settings.Init() failed: %s", err))
	}

	// 2.初始化日志
	if err := logger.Init(settings.Config.LogConfig); err != nil {
		panic(fmt.Sprintf("logger.Init() failed: %s", err))
	}
	zap.L().Info("Logger init success")

	// 3.初始化 MySQL
	if err := mysql.Init(settings.Config.MysqlConfig); err != nil {
		panic(fmt.Sprintf("mysql.Init() failed: %s", err))
	}

	// 4. 读取并解析 universities.json
	universitiesFilePath := "assets/universities.json"
	zap.L().Info("正在读取并解析 universities.json 文件...", zap.String("文件路径", universitiesFilePath))
	initUniversities, err := unmarshalUniversities(universitiesFilePath)
	if err != nil {
		zap.L().Error("读取或解析 universities.json 文件失败", zap.Error(err), zap.String("文件路径", universitiesFilePath))
		return
	}
	zap.L().Info("成功读取 universities.json 文件", zap.Int("数据条数", len(initUniversities)))
	// 🚨 增加检查：打印出被解析后的第一条数据
	if len(initUniversities) > 0 {
		firstUni := initUniversities[0]
		zap.L().Debug("DEBUG: 检查第一条记录的解析结果",
			zap.String("Slug", firstUni.Slug),
			zap.String("ShortName", firstUni.ShortName),
			zap.String("Title", firstUni.Title))
	}
	// 🚨 强制检查：如果发现 short_name 为空字符串，立即致命报错退出。
	for i, u := range initUniversities {
		if u.ShortName == "" {
			// 使用 Fatal 级别日志，强制程序停止并打印详细错误信息
			zap.L().Fatal("致命错误：发现 short_name 为空字符串的记录，程序终止。",
				zap.String("错误原因", "short_name 字段是数据库的唯一键，不允许为空字符串。请修复 universities.json 文件中的该记录。"),
				zap.String("可能的问题记录 slug", u.Slug),
				zap.Int("数组索引", i),
			)
			// 注意：Fatal 会自动调用 os.Exit(1)，所以后面的代码不会执行
		}
	}
	// 准备一个完整结构的切片，用于后续的 InitUniversitiesParams 调用
	universitiesForParams := make([]settings.Universities, len(initUniversities))
	for i, u := range initUniversities {
		// 将 InitUniversities 的字段映射到 Universities
		// 这样做的好处是，HasVector, ResourceCount 等字段会被初始化为零值(0)，
		// 不会覆盖数据库中可能存在的默认值，并准备好供 InitUniversitiesParams 写入。
		universitiesForParams[i] = settings.Universities{
			Slug:       u.Slug,
			ShortName:  u.ShortName,
			Title:      u.Title,
			Vis:        sql.NullString{String: u.Vis, Valid: u.Vis != ""},
			Website:    u.Website,
			FullNameEn: u.FullNameEn,
			Region:     u.Region,
			Province:   u.Province,
			City:       u.City,
			Story:      sql.NullString{String: u.Story, Valid: u.Story != ""},
		}
	}

	// 5. 读取并解析 university_resources.json
	resourcesFilePath := "assets/university_resources.json"
	zap.L().Info("正在读取并解析 university_resources.json 文件...", zap.String("文件路径", resourcesFilePath))
	universityResources, err := unmarshalUniversityResources(resourcesFilePath)
	if err != nil {
		zap.L().Error("读取或解析 university_resources.json 文件失败", zap.Error(err), zap.String("文件路径", resourcesFilePath))
		return
	}
	zap.L().Info("成功读取 university_resources.json 文件", zap.Int("数据条数", len(universityResources)))

	// 6. 执行数据库批量插入操作
	var curUniversities []settings.Universities
	if curUniversities, err = mysql.GetAllUniversities(); err != nil {
		zap.L().Error("mysql.GetAllUniversities()失败")
		return
	}
	if len(curUniversities) == 0 {
		zap.L().Info("universities 表中尚无数据，可以进行初始化插入！")
		zap.L().Info("开始向 universities 表插入数据。")
		if err := mysql.InitInsertUniversities(initUniversities); err != nil {
			zap.L().Error("universities 表批量插入数据失败", zap.Error(err))
			return
		}
		zap.L().Info(`universities 表批量插入数据成功。初始插入字段有："slug", "short_name", "title", "vis", "website", "full_name_en", "region", "province", "city", "story"`) // 原生字符串内部可以包含任何字符，包括双引号，而无需转义。
	} else {
		zap.L().Info("universities 表中已有初始化数据，无需插入！")
	}

	var curUniversityResources []settings.UniversityResources
	if curUniversityResources, err = mysql.GetAllUniversityResources(); err != nil {
		zap.L().Error("mysql.GetAllUniversities()失败！")
	}
	if len(curUniversityResources) == 0 {
		zap.L().Info("university_resources 表中尚无数据，可进行初始化查入！")
		zap.L().Info("开始向 university_resources 表 插入数据。")
		if err := mysql.InsertResource(universityResources); err != nil {
			zap.L().Error("university_resources 表批量插入数据失败", zap.Error(err))
			return
		}
		zap.L().Info("university_resources 表批量插入数据成功。")
	} else {
		zap.L().Info("university_resources 表已有初始化数据，无需插入！")
	}

	// 7.进行 universities 表 剩余字段初始化操作
	for _, curUniversity := range universitiesForParams {
		if err = mysql.InitUniversitiesParams(curUniversity); err != nil {
			zap.L().Error("mysql.InitUniversitiesParams(curUniversity) 失败！", zap.Error(err))
			return
		}
	}
	zap.L().Info("universities 表 剩余字段初始化操作完成！")

	zap.L().Info("所有数据导入任务已成功完成。")
}
*/
