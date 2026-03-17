package controller

import (
	"strconv"
	"ticket/dao"
	"ticket/models"
	"ticket/utils/response"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func GetTicketDetail(c *gin.Context) {
	// 获取数据库及参数
	db := dao.GetDB()
	ticketId, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || ticketId <= 0 {
		response.JsonErr(c, 400, "票ID错误")
		return
	}

	// 查询
	var ticket models.Ticket
	if err := db.Model(&models.Ticket{}).
		Select("`tickets`.*, `activities`.`name` AS `activity_name`").
		Joins("LEFT JOIN `activities` ON `activities`.`id` = `ticketd`.`activity_id`").
		First(&ticket, ticketId).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			response.JsonErr(c, 404, "票不存在")
			return
		}
		response.JsonErr(c, 500, "查询失败")
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
	// 获取数据库及参数
	db := dao.GetDB()
	ticketId, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || ticketId <= 0 {
		response.JsonErr(c, 400, "票ID错误")
		return
	}

	// 检验票是否存在
	var ticket models.Ticket
	if err := db.First(&ticket, ticketId).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			response.JsonErr(c, 404, "票不存在")
			return
		}
		response.JsonErr(c, 500, "查询失败")
		return
	}

	// 验参
	switch ticket.Status {
	case 1:
		response.JsonErr(c, 400, "该票已被使用")
		return
	case 2:
		response.JsonErr(c, 400, "该票已作废")
		return
	case 0:
		// 正常并继续执行
	default:
		response.JsonErr(c, 400, "该票状态错误")
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
	// 获取数据库及参数
	db := dao.GetDB()
	ticketId, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || ticketId <= 0 {
		response.JsonErr(c, 400, "票ID错误")
		return
	}

	// 检验票是否存在
	var ticket models.Ticket
	if err := db.First(&ticket, ticketId).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			response.JsonErr(c, 404, "票不存在")
			return
		}
		response.JsonErr(c, 500, "查询失败")
		return
	}

	// 状态检验
	if ticket.Status == 2 {
		response.JsonErr(c, 400, "票已作废")
		return
	}

	// 修改
	if err := db.Model(&models.Ticket{}).Where("id = ?", ticketId).Update("status", 2).Error; err != nil {
		response.JsonErr(c, 500, "作废票错误")
		return
	}

	// 构建成功响应
	db.First(&ticket, ticketId) // 获取修改后的数据
	response.JsonOK(c, "作废成功", gin.H{
		"ticketId": ticket.ID,
		"ticketNo": ticket.TicketNo,
		"status":   ticket.Status,
	})
}

func GetTickets(c *gin.Context) {
	// 获取数据库
	db := dao.GetDB()
	queryDB := db.Model(&models.Ticket{}).
		Select("`tickets`.*, `activities`.`name` AS `activity_name`").
		Joins("LEFT JOIN `activities` ON `activities`.`id` = `tickets`.`activity_id`")

	// 构建订单 ID 条件
	orderId, _ := strconv.ParseInt(c.DefaultQuery("orderId", "0"), 10, 64)
	if orderId > 0 {
		queryDB = queryDB.Where("`tickets`.`order_id` = ?", orderId)
	}

	// 构建活动 ID 条件
	activityId, _ := strconv.ParseInt(c.DefaultQuery("activityId", "0"), 10, 64)
	if activityId > 0 {
		queryDB = queryDB.Where("`tickets`.`activity_id` = ?", activityId)
	}

	// 构建状态条件
	var statusList []int
	statusStr := c.QueryArray("status")
	for _, s := range statusStr {
		if st, err := strconv.Atoi(s); err == nil && st >= 0 && st < 3 {
			statusList = append(statusList, st)
		}
	}
	if len(statusList) == 0 {
		statusList = []int{0, 1}
	}
	queryDB = queryDB.Where("`tickets`.`status` IN (?)", statusList)

	// 分页构建
	pageNum, err := strconv.Atoi(c.DefaultQuery("pageNum", "1"))
	if err != nil || pageNum < 1 {
		pageNum = 1
	}
	pageSize, err := strconv.Atoi(c.DefaultQuery("pageSize", "10"))
	if err != nil || pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}

	// 查询
	var total int64
	if err := queryDB.Count(&total).Error; err != nil {
		response.JsonErr(c, 500, "查询失败")
		return
	}

	var tickets []models.Ticket
	offset := (pageNum - 1) * pageSize
	if err := queryDB.
		Limit(pageSize).Offset(offset).
		Order("`tickets`.`status` ASC, `tickets`.`created_at` ASC").
		Find(&tickets).Error; err != nil {
		response.JsonErr(c, 500, "查询失败")
		return
	}

	// 构建成功响应
	var ticketList []gin.H
	for _, ticket := range tickets {
		ticketList = append(ticketList, gin.H{
			"ticketId":     ticket.ID,
			"activityId":   ticket.ActivityID,
			"activityName": ticket.ActivityName,
			"status":       ticket.Status,
		})
	}
	response.JsonOK(c, "成功返回票列表", gin.H{
		"tickets":  ticketList,
		"total":    total,
		"pageNum":  pageNum,
		"pageSize": pageSize,
	})
}

func VerifyTicketNO(c *gin.Context) {
	// 获取数据库及参数
	db := dao.GetDB()
	ticketNo := c.DefaultQuery("ticketNo", "")
	if ticketNo == "" {
		response.JsonErr(c, 400, "票号不应为空")
		return
	}

	// 检验票是否存在
	var ticket models.Ticket
	if err := db.Model(&models.Ticket{}).
		Select("`tickets`.*, `activities`.`name` AS `activity_name`").
		Joins("LEFT JOIN `activities` ON `activities`.`id` = `tickets`.`activity_id`").
		Where("`ticket_no` = ?", ticketNo).
		First(&ticket).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			response.JsonErr(c, 404, "票不存在")
			return
		}
		response.JsonErr(c, 500, "查询失败")
		return
	}

	// 检票
	if ticket.Status != 0 {
		response.JsonErr(c, 400, "票无效-无法检票")
		return
	}

	if err := db.Model(&models.Ticket{}).
		Where("id = ?", ticket.ID).
		Update("status", 1).Error; err != nil {
		response.JsonErr(c, 500, "检票失败")
		return
	}

	// 构建成功响应
	db.First(&ticket, ticket.ID)
	response.JsonOK(c, "检票成功", gin.H{
		"ticketId":     ticket.ID,
		"ticketNo":     ticket.TicketNo,
		"activityName": ticket.ActivityName,
		"status":       ticket.Status,
	})
}
