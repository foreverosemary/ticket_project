package models

import (
	"time"

	"gorm.io/gorm"
)

const UP, PD, CL = 0, 1, 2
const LIMIT = 10

type Order struct {
	ID        int64          `json:"id"`
	UserID    int64          `gorm:"not null;index:idx_user_status_delete,priority:1" json:"userId"`
	Status    int8           `gorm:"type:tinyint;not null;default:0;index:idx_user_status_delete,priority:2" json:"status"`
	PayTime   *time.Time     `json:"payTime,omitempty"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	DeletedAt gorm.DeletedAt `gorm:"index:idx_user_status_delete,priority:3" json:"deletedAt"`

	// 临时字段
	ActivityId   int64  `gorm:"->" json:"activityId"`
	ActivityName string `gorm:"->" json:"activityName"`
}

type OrderQuery struct {
	UserID     int64
	ActivityID int64
	OrderID    int64
	StatusList []int
	PageNum    int
	PageSize   int
}

type OrderList struct {
	Orders []Order
	Total  int64
}

func (Order) TableName() string {
	return "orders"
}
