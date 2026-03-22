package response

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

const FmtTime = "2006-01-02 15:04:05"

type LocalTime time.Time

// 解析前端传过来的时间字符串
func (t *LocalTime) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		return nil
	}
	// 去除引号
	var str string
	if err := json.Unmarshal(data, &str); err != nil {
		return err
	}

	// 确保按照服务器本地时区解析
	parseTime, err := time.ParseInLocation(FmtTime, str, time.Local)
	if err != nil {
		return err
	}
	*t = LocalTime(parseTime)
	return nil
}

// 返回给前端标准时间格式
func (t LocalTime) MarshalJSON() ([]byte, error) {
	// 修改: 增加空值处理
	if time.Time(t).IsZero() {
		return []byte("null"), nil
	}
	return []byte(fmt.Sprintf(`"%s"`, time.Time(t).Format(FmtTime))), nil
}

// 存入数据库
func (t LocalTime) Value() (driver.Value, error) {
	if time.Time(t).IsZero() {
		return nil, nil
	}
	return time.Time(t), nil
}

// 从数据库读取
func (t *LocalTime) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	tm, ok := value.(time.Time)
	if !ok {
		return fmt.Errorf("expected time.Time, got %T", value)
	}
	*t = LocalTime(tm)
	return nil
}

// 普适性封装
func (t LocalTime) Time() time.Time {
	return time.Time(t)
}

// 常用方法封装
func (t LocalTime) Before(u LocalTime) bool {
	return t.Time().Before(u.Time())
}

func (t LocalTime) After(u LocalTime) bool {
	return t.Time().After(u.Time())
}

func (t LocalTime) IsZero() bool {
	return t.Time().IsZero()
}

func (t LocalTime) Format(layout string) string {
	return time.Time(t).Format(layout)
}

func (t LocalTime) ToTime() time.Time {
	return time.Time(t)
}
