package response

import (
	"time"

	"github.com/gin-gonic/gin"
)

const FmtTime = "2006-01-02 15:04:05"

type Response struct {
	Code      int         `json:"code"`
	Msg       string      `json:"msg"`
	Data      interface{} `json:"data"`
	Timestamp int64       `json:"timestamp"`
}

type RoleInfo struct {
	RoleId   int    `json:"roleId"`
	RoleName string `json:"roleName"`
	RoleCode string `json:"roleCode"`
}

func JsonErr(c *gin.Context, code int, msg string) {
	c.JSON(code, Response{
		Code:      code,
		Msg:       msg,
		Data:      nil,
		Timestamp: time.Now().UnixMilli(),
	})
}

func JsonOK(c *gin.Context, msg string, data interface{}) {
	c.JSON(200, Response{
		Code:      200,
		Msg:       msg,
		Data:      data,
		Timestamp: time.Now().UnixMilli(),
	})
}
