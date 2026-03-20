package controller

import (
	"strconv"
	"ticket/dao"
	"ticket/models"
	"ticket/utils/response"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func CreateOrder(c *gin.Context) {

}

func UpdateOrder(c *gin.Context) {
	// 获取数据库及参数
	db := dao.GetDB().Unscoped()
	orderId, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || orderId < 0 {
		response.JsonErr(c, 400, "订单ID错误")
		return
	}

	type Dto struct {
		Status int `json:"status"`
	}
	var dto Dto

	err = c.ShouldBindJSON(&dto)
	if err != nil || (dto.Status != models.PD && dto.Status != models.CL) {
		response.JsonErr(c, 400, "订单状态错误")
		return
	}

	// 检验参数
	var order models.Order
	if err := db.First(&order, orderId).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			response.JsonErr(c, 404, "订单不存在")
			return
		}
		response.JsonErr(c, 500, "查询失败")
		return
	}

	if order.Status == models.CL {
		response.JsonErr(c, 400, "不可修改已取消订单")
		return
	}

	// 更新

	// 构建成功响应
	response.JsonOK(c, "修改订单状态成功", gin.H{
		"orderId": orderId,
		"status":  dto.Status,
	})
}

func GetOrders(c *gin.Context) {

}

func GetOrderDetail(c *gin.Context) {

}
