package models

import "time"

type Role struct {
	ID          int       `json:"id"`
	Name        string    `gorm:"type:varchar(20);not null" json:"name"`
	Code        string    `gorm:"type:varchar(20);not null;unique" json:"code"`
	Description string    `gorm:"type:varchar(255);default:''" json:"description"`
	Status      int8      `gorm:"type:tinyint;not null;default:1;index" json:"status"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

func (Role) TableName() string {
	return "roles"
}
