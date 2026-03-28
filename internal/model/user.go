package model

import "time"

type User struct {
	ID        uint   `gorm:"column:id;primaryKey"`
	Username  string `gorm:"column:username" form:"user_name" json:"username"`
	Password  string `gorm:"column:password" form:"password" json:"password"`
	NickName  string `gorm:"column:nickname" form:"nick_name" json:"nickname"`
	AvatarUrl string
	CreatedAt time.Time
}
