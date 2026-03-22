package response

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

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

func GetPage(c *gin.Context) (int, int) {
	pageNum, err := strconv.Atoi(c.DefaultQuery("pageNum", "1"))
	if err != nil || pageNum < 1 {
		pageNum = 1
	}
	pageSize, err := strconv.Atoi(c.DefaultQuery("pageSize", "10"))
	if err != nil || pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}
	return pageNum, pageSize
}
