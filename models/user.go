package models

import "time"

type User struct {
	ID        int64     `json:"id"`
	Username  string    `gorm:"column:username;type:varchar(20);not null" json:"userName"`
	Password  string    `gorm:"column:password;type:varchar(255);not null" json:"-"`
	RoleID    int       `gorm:"column:role_id;not null;index:idx_role_id" json:"roleId"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func (User) TableName() string {
	return "users"
}
