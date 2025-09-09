package model

type User struct {
	Id        int    `json:"id"`
	Name      string `json:"name"`
	CreatedAt int64  `json:"createdAt" gorm:"autoCreateTime;not null;comment:创建时间"`
}

// 表名
func (User) TableName() string {
	return "user"
}
