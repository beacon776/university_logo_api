package do

// UserDO user 表的映射
type UserDO struct {
	ID       int    `gorm:"primaryKey;column:id;autoIncrement" json:"id"`
	Username string `gorm:"column:username" json:"username"`

	// json:"-" 忽略 json 映射
	Password string `gorm:"column:password" json:"-"`
	Status   int    `gorm:"column:status" json:"status"`
}
