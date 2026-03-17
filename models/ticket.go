package models

import (
	"time"

	"gorm.io/gorm"
)

type Ticket struct {
	ID         int64          `json:"id"`
	TicketNo   string         `gorm:"type:varchar(32);not null;unique" json:"ticketNo"`
	ActivityID int64          `gorm:"not null;index" json:"activityId"`
	OrderID    int64          `gorm:"index" json:"orderId"`
	Status     int8           `gorm:"type:tinyint;not null;default:0;index:idx_status_deleted_at,priority:1" json:"status"`
	CreatedAt  time.Time      `json:"createdAt"`
	UpdatedAt  time.Time      `json:"updatedAt"`
	DeletedAt  gorm.DeletedAt `gorm:"index:idx_status_deleted_at,priority:2" json:"deletedAt"`

	// 临时字段
	ActivityName string `gorm:"-" json:"activityName"`
	UserID       int64  `gorm:"-" json:"userId"`
}

func (Ticket) TableName() string {
	return "tickets"
}
