package model

import "time"

type User struct {
	ID        uint      `gorm:"column:id;primaryKey" form:"id" json:"id" redis:"id"`
	Username  string    `gorm:"column:username" form:"user_name" json:"username" redis:"user_name"`
	Password  string    `gorm:"column:password" form:"password" json:"password" redis:"-"`
	NickName  string    `gorm:"column:nickname" form:"nick_name" json:"nickname" redis:"nickname"`
	AvatarUrl string    `redis:"avatar_url"`
	CreatedAt time.Time `redis:"created_at"`
}
