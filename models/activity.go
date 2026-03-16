package models

import (
	"errors"
	"time"

	"gorm.io/gorm"
)

type Activity struct {
	ID        int64          `json:"id"`
	Name      string         `gorm:"type:varchar(30);not null" json:"name"`
	Content   *string        `gorm:"type:text" json:"content"`
	Stock     int            `gorm:"default:0" json:"stock"`
	Status    int8           `gorm:"type:tinyint;not null;default:0;index:idx_status_end_time,priority:1" json:"status"`
	StartTime time.Time      `gorm:"not null" json:"startTime"`
	EndTime   time.Time      `gorm:"not null;index:idx_status_end_time,priority:2" json:"endTime"`
	CreatorID int64          `gorm:"not null;index" json:"creatorId"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deletedAt"`
}

func (Activity) TableName() string {
	return "activities"
}

func (a *Activity) Verify() error {
	// 名字长度 1 ~ 30 字
	if len([]rune(a.Name)) < 1 || len([]rune(a.Name)) > 30 {
		return errors.New("活动名称应为 1~30 字")
	}
	// 库存非负
	if a.Stock < 0 {
		return errors.New("库存应为非负数")
	}
	// 开始时间 < 结束时间
	if !a.StartTime.Before(a.EndTime) {
		return errors.New("活动开始时间必须早于结束时间")
	}
	return nil
}

func (a *Activity) SetStatus() {
	if time.Now().Before(a.StartTime) {
		a.Status = 0
	} else if time.Now().Before(a.EndTime) {
		a.Status = 1
	} else {
		a.Status = 2
	}
}

// 用于部分更新
type UpdateActivityDTO struct {
	Name      *string    `json:"name"`
	Content   *string    `json:"content"`
	Stock     *int       `json:"stock"`
	StartTime *time.Time `json:"startTime"`
	EndTime   *time.Time `json:"endTime"`
}

func (a *Activity) ApplyUpdates(dto *UpdateActivityDTO) {
	if dto.Name != nil {
		a.Name = *dto.Name
	}
	if dto.Content != nil {
		a.Content = dto.Content
	}
	if dto.Stock != nil {
		a.Stock = *dto.Stock
	}
	if dto.StartTime != nil {
		a.StartTime = *dto.StartTime
	}
	if dto.EndTime != nil {
		a.EndTime = *dto.EndTime
	}
	a.SetStatus()
}
