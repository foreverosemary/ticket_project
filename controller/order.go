package controller

import (
	"errors"
	"fmt"
	"strconv"
	"ticket/logic"
	"ticket/models"
	"ticket/utils/response"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

var (
	orderLogic logic.OrderLogic
)

func CreateOrder(c *gin.Context) {
	ctx := c.Request.Context()

	type Dto struct {
		ActivityId int64 `json:"activityId"`
		Need       int   `json:"need"`
	}

	var dto Dto
	if err := c.ShouldBindBodyWithJSON(&dto); err != nil {
		response.JsonErr(c, 400, err.Error())
		return
	}

	if dto.ActivityId <= 0 {
		response.JsonErr(c, 400, "活动ID错误")
		return
	}

	if dto.Need <= 0 || dto.Need > models.LIMIT {
		response.JsonErr(c, 400, fmt.Sprintf("所购票数应该为 1 ~ %v", models.LIMIT))
		return
	}

	userId := c.GetInt64("userId")

	// 调用逻辑层
	orderInfo, err := orderLogic.CreateOrder(ctx, dto.ActivityId, userId, dto.Need)
	if err != nil {
		response.JsonErr(c, 400, err.Error())
		return
	}

	// 构建成功响应
	response.JsonOK(c, "下单成功", map[string]interface{}{
		"order": orderInfo,
	})
}

func UpdateOrder(c *gin.Context) {
	// 获取数据库及参数
	ctx := c.Request.Context()
	orderId, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || orderId <= 0 {
		response.JsonErr(c, 400, "订单ID错误")
		return
	}

	type Dto struct {
		Status int `json:"status"`
	}
	var dto Dto
	if err := c.ShouldBindJSON(&dto); err != nil {
		response.JsonErr(c, 400, "参数格式错误")
		return
	}
	if dto.Status != models.PD && dto.Status != models.CL {
		response.JsonErr(c, 400, "不支持的订单操作")
		return
	}

	userId := c.GetInt64("userId")

	// 调用逻辑层
	if err := orderLogic.UpdateOrder(ctx, orderId, userId, dto.Status); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.JsonErr(c, 404, "未找到指定订单")
		} else {
			response.JsonErr(c, 400, err.Error())
		}
		return
	}

	// 构建成功响应
	response.JsonOK(c, "修改订单状态成功", gin.H{
		"orderId": orderId,
		"status":  dto.Status,
	})
}

func GetOrders(c *gin.Context) {
	// 获取参数
	var q models.OrderQuery
	orderId, _ := strconv.ParseInt(c.DefaultQuery("orderId", "0"), 10, 64)
	q.OrderID = orderId

	userId, _ := strconv.ParseInt(c.DefaultQuery("userId", "0"), 10, 64)
	q.UserID = userId

	activityId, _ := strconv.ParseInt(c.DefaultQuery("activityId", "0"), 10, 64)
	q.ActivityID = activityId

	var statusList []int
	statusStr := c.QueryArray("status")
	for _, s := range statusStr {
		if st, err := strconv.Atoi(s); err == nil && st >= models.UP && st <= models.CL {
			statusList = append(statusList, st)
		}
	}
	if len(statusList) == 0 {
		statusList = []int{models.UP, models.PD}
	}
	q.StatusList = statusList

	q.PageNum, q.PageSize = response.GetPage(c)

	// 调用逻辑层
	orderList, err := orderLogic.GetOrders(q)
	if err != nil {
		response.JsonErr(c, 400, err.Error())
	}

	// 构建成功响应
	var orders []gin.H
	for _, order := range orderList.Orders {
		orders = append(orders, gin.H{
			"orderId":      order.ID,
			"status":       order.Status,
			"activityId":   order.ActivityId,
			"activityName": order.ActivityName,
			"createdAt":    order.CreatedAt.Format(response.FmtTime),
			"payTime":      order.PayTime.Format(response.FmtTime),
		})
	}

	response.JsonOK(c, "成功返回订单列表", gin.H{
		"orders":   orders,
		"total":    orderList.Total,
		"pageNum":  q.PageNum,
		"pageSize": q.PageSize,
	})
}

func GetOrderDetail(c *gin.Context) {
	orderId, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || orderId <= 0 {
		response.JsonErr(c, 400, "订单ID错误")
		return
	}

	// 调用逻辑层
	orderInfo, err := orderLogic.GetOrderDetail(orderId)
	if err != nil {
		if errors.Is(gorm.ErrRecordNotFound, err) {
			response.JsonErr(c, 404, "指定活动不存在")
		} else {
			response.JsonErr(c, 400, err.Error())
		}
		return
	}

	// 构建成功响应
	response.JsonOK(c, "成功返回活动详情", gin.H{
		"order": orderInfo,
	})
}
