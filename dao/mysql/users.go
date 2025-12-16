package mysql

import (
	"errors"
	"fmt"
	"go.uber.org/zap"
	"logo_api/model"
	"strings"

	// 确保 GORM 相关的导入可用
	"gorm.io/gorm"
)

// 假设 db 已经是 *gorm.DB 类型，并已在 Init 中初始化

// GetUserFromName 使用 GORM 查询单个用户
func GetUserFromName(username string) (model.User, error) {
	var result model.User

	// 使用 db.Where() 设置查询条件，然后用 First() 查找第一条记录
	// GORM 会自动将结果映射到 result
	err := db.Where("username = ? and status = ?", username, 1).First(&result).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// 如果找不到记录，返回一个明确的空结构体和 nil 错误，或者返回 gorm.ErrRecordNotFound
			return model.User{}, ErrUserNotFound
		}
		zap.L().Error("GetUserFromName() failed", zap.Error(err))
		return model.User{}, err
	}

	zap.L().Info("GetUserFromName() success", zap.String("username", username))
	return result, nil
}

// InsertUser 使用 GORM 插入用户
func InsertUser(user model.User) (err error) {
	// 使用 db.Create() 插入数据。GORM 会自动使用结构体的字段名和值来构建 INSERT 语句。
	err = db.Create(&user).Error

	if err != nil {
		zap.L().Error("InsertUser() failed", zap.Error(err))
		// 常见错误：唯一约束冲突等，需要根据实际情况处理
		return err
	}

	zap.L().Info("InsertUser() success", zap.String("username", user.Username))
	return nil
}

// GetUserList 使用 GORM 实现动态查询和分页
func GetUserList(page, pageSize int, keyword, sortBy, sortOrder string) (users []model.UserListResponse, totalCount int64, err error) {
	// 1. 初始化 GORM 查询构建器
	tx := db.Model(&model.User{})
	// 排除敏感字段 password，手动选择 id，username，status
	tx = tx.Select("id", "username", "status")
	// 保证只查询 status = 1 的用户
	tx = tx.Where("status = ?", 1)
	// 2. 动态构建 WHERE 条件（搜索/筛选）
	if keyword != "" {
		// 使用 LOWER() 函数强制不区分大小写匹配

		// 1. 将搜索关键字转为小写，并添加 %
		searchKeyword := "%" + strings.ToLower(keyword) + "%"

		// 2. 在 SQL 中对数据库字段也使用 LOWER() 函数
		// 假设您只搜索 username 字段
		tx = tx.Where("LOWER(username) LIKE ?", searchKeyword)

		// 如果需要同时搜索 username 和 title 字段，可以这样写：
		/*
		   tx = tx.Where("LOWER(username) LIKE ? OR LOWER(title) LIKE ?",
		       searchKeyword,
		       searchKeyword)
		*/
	}

	// 3. 动态构建 ORDER BY 排序子句
	if sortBy != "" {
		// 安全检查重要：确保 sortBy 字段是安全的
		allowedSortFields := map[string]bool{"id": true, "username": true}

		if allowedSortFields[sortBy] {

			// 1. 标准化排序方向
			order := "ASC" // 默认值
			upperSortOrder := strings.ToUpper(sortOrder)

			if upperSortOrder == "DESC" || upperSortOrder == "ASC" {
				// 只有当传入值为 "DESC" 或 "ASC" 时才使用它
				order = upperSortOrder
			} else {
				// 可选：如果传入非法值，记录警告并使用默认值 (ASC)
				zap.L().Warn("Invalid sortOrder value, defaulting to ASC", zap.String("received", sortOrder))
			}

			orderClause := fmt.Sprintf("%s %s", sortBy, order)
			// 使用 Order() 方法
			tx = tx.Order(orderClause)

		} else {
			zap.L().Warn("Invalid sortBy field ignored", zap.String("sortBy", sortBy))
		}
	} else {
		// 设置默认排序，例如按 ID 降序
		tx = tx.Order("id DESC")
	}

	// 4a. 获取总记录数 (在应用 LIMIT/OFFSET 之前)
	// 注意：Count 必须在 Select 之后，因为它只需要计算符合 Where 条件的总数
	if err = tx.Count(&totalCount).Error; err != nil {
		// 处理错误
		return nil, 0, err
	}

	// 4. 构建 LIMIT 和 OFFSET 分页子句
	if pageSize > 0 && page > 0 {
		offset := (page - 1) * pageSize

		// 使用 Limit() 和 Offset() 方法
		tx = tx.Limit(pageSize).Offset(offset)
	}

	// 5. 执行查询
	// Find 方法执行查询，并将结果映射到 users 切片
	// 此时，users 切片中的每个 model.User 实例的 Password 字段将是零值（空字符串）
	var userResponses []model.UserListResponse
	err = tx.Find(&userResponses).Error

	if err != nil {
		// GORM Find 方法找不到记录时，不会返回错误，只会返回空切片，
		// 只有当发生数据库连接或其他严重错误时才会返回 error。
		zap.L().Error("GetUserList() failed", zap.Error(err))
		return nil, 0, err
	}

	zap.L().Info("GetUserList() success", zap.Int("userListLength", len(userResponses)))
	// GORM 找到 0 条记录时，users 为空切片 []model.User{}，返回 nil 错误
	return userResponses, totalCount, nil
}
