package controller

import (
	"errors"
	"strconv"
	"ticket/logic"
	"ticket/models"
	"ticket/utils/response"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

var (
	ticketLogic logic.TicketLogic
)

func GetTicketDetail(c *gin.Context) {
	// 获取参数
	ticketId, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || ticketId <= 0 {
		response.JsonErr(c, 400, "票ID错误")
		return
	}

	userId := c.GetInt64("userId")
	roleId := c.GetInt("roleId")

	// 查询
	ticket, err := ticketLogic.GetTicketDetail(ticketId, userId, roleId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.JsonErr(c, 404, "票不存在")
			return
		}
		response.JsonErr(c, 400, err.Error())
		return
	}

	// 构建成功响应
	response.JsonOK(c, "成功返回票的详情", gin.H{
		"ticketId":     ticket.ID,
		"ticketNo":     ticket.TicketNo,
		"orderId":      ticket.OrderID,
		"activityId":   ticket.ActivityID,
		"activityName": ticket.ActivityName,
		"status":       ticket.Status,
	})
}

func VerifyTicket(c *gin.Context) {
	// 获取参数
	ticketId, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || ticketId <= 0 {
		response.JsonErr(c, 400, "票ID错误")
		return
	}

	ticket, err := ticketLogic.VerifyTicket(ticketId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.JsonErr(c, 404, "票不存在")
			return
		}
		response.JsonErr(c, 400, "查询失败:"+err.Error())
		return
	}

	// 构建成功响应
	response.JsonOK(c, "验票成功", gin.H{
		"ticketId": ticket.ID,
		"ticketNo": ticket.TicketNo,
		"status":   ticket.Status,
	})
}

func InvalidateTicket(c *gin.Context) {
	// 获取参数
	ticketId, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || ticketId <= 0 {
		response.JsonErr(c, 400, "票ID错误")
		return
	}

	ticket, err := ticketLogic.InvalidateTicket(ticketId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.JsonErr(c, 404, "票不存在")
			return
		}
		response.JsonErr(c, 400, "查询失败")
		return
	}

	// 构建成功响应
	response.JsonOK(c, "作废成功", gin.H{
		"ticketId": ticket.ID,
		"ticketNo": ticket.TicketNo,
		"status":   ticket.Status,
	})
}

func GetTickets(c *gin.Context) {
	// 订单 ID 条件
	var q models.TicketQuery
	q.OrderID, _ = strconv.ParseInt(c.DefaultQuery("orderId", "0"), 10, 64)

	// 活动 ID 条件
	q.ActivityID, _ = strconv.ParseInt(c.DefaultQuery("activityId", "0"), 10, 64)

	// 构建状态条件
	var statusList []int
	statusStr := c.QueryArray("status")
	for _, s := range statusStr {
		if st, err := strconv.Atoi(s); err == nil && st >= models.UD && st < models.IV {
			statusList = append(statusList, st)
		}
	}
	if len(statusList) == 0 {
		statusList = []int{models.UD, models.US}
	}
	q.StatusList = statusList

	// 分页构建
	q.PageNum, q.PageSize = response.GetPage(c)

	// 调用逻辑层
	ticketList, err := ticketLogic.GetTickets(q)
	if err != nil {
		response.JsonErr(c, 400, err.Error())
		return
	}

	// 构建成功响应
	var tickets []gin.H
	for _, ticket := range ticketList.Tickets {
		tickets = append(tickets, gin.H{
			"ticketId":     ticket.ID,
			"activityId":   ticket.ActivityID,
			"activityName": ticket.ActivityName,
			"status":       ticket.Status,
		})
	}
	response.JsonOK(c, "成功返回票列表", gin.H{
		"tickets":  tickets,
		"total":    ticketList.Total,
		"pageNum":  q.PageNum,
		"pageSize": q.PageSize,
	})
}

func VerifyTicketNO(c *gin.Context) {
	// 获取数据库及参数
	ticketNo := c.DefaultQuery("ticketNo", "")
	if ticketNo == "" {
		response.JsonErr(c, 400, "票号不应为空")
		return
	}

	ticket, err := ticketLogic.VerifyTicketNO(ticketNo)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.JsonErr(c, 404, "票不存在")
			return
		}
		response.JsonErr(c, 400, "查询失败")
		return
	}

	// 构建成功响应
	response.JsonOK(c, "检票成功", gin.H{
		"ticketId":     ticket.ID,
		"ticketNo":     ticket.TicketNo,
		"activityName": ticket.ActivityName,
		"status":       ticket.Status,
	})
}
