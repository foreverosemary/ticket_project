package models

import (
	"errors"
	"ticket/utils/response"
	"time"

	"gorm.io/gorm"
)

const NS, IP, ED, RM = 0, 1, 2, 3

type Activity struct {
	ID        int64              `json:"id"`
	Name      string             `gorm:"type:varchar(30);not null" json:"name"`
	Content   *string            `gorm:"type:text" json:"content"`
	Stock     int                `gorm:"default:0" json:"stock"`
	Total     int                `gorm:"default:0" json:"total"`
	StartTime response.LocalTime `gorm:"not null" json:"startTime"`
	EndTime   response.LocalTime `gorm:"not null;index:idx_status_end_time,priority:2" json:"endTime"`
	CreatorID int64              `gorm:"not null;index" json:"creatorId"`
	CreatedAt time.Time          `json:"createdAt"`
	UpdatedAt time.Time          `json:"updatedAt"`
	DeletedAt gorm.DeletedAt     `gorm:"index" json:"deletedAt"`
}

type ActivityQuery struct {
	UserID     int64
	ActivityID int64
	Name       string
	StatusList []int
	PageNum    int
	PageSize   int
}

type ActivityList struct {
	Activities []Activity
	Total      int64
}

func (Activity) TableName() string {
	return "activities"
}

func (a *Activity) Verify() error {
	// 名字长度 1 ~ 30 字
	if len([]rune(a.Name)) < 1 || len([]rune(a.Name)) > 30 {
		return errors.New("活动名称应为 1~30 字")
	}
	// 总量非负
	if a.Total < 0 {
		return errors.New("活动总量应为非负数")
	}
	// 开始时间 < 结束时间
	if !a.StartTime.Before(a.EndTime) {
		return errors.New("活动开始时间必须早于结束时间")
	}
	return nil
}

func (a *Activity) GetStatus() int {
	if !a.DeletedAt.Valid {
		return RM
	}
	if time.Now().Before(a.StartTime.ToTime()) {
		return NS
	} else if time.Now().Before(a.EndTime.ToTime()) {
		return IP
	} else {
		return ED
	}
}

// 用于部分更新
type UpdateActivityDTO struct {
	Name      *string             `json:"name"`
	Content   *string             `json:"content"`
	Total     *int                `json:"total"`
	StartTime *response.LocalTime `json:"startTime"`
	EndTime   *response.LocalTime `json:"endTime"`
}

func (a *Activity) ApplyUpdates(dto *UpdateActivityDTO) {
	if dto.Name != nil {
		a.Name = *dto.Name
	}
	if dto.Content != nil {
		a.Content = dto.Content
	}
	if dto.Total != nil {
		a.Total = *dto.Total
	}
	if dto.StartTime != nil {
		a.StartTime = *dto.StartTime
	}
	if dto.EndTime != nil {
		a.EndTime = *dto.EndTime
	}
}
