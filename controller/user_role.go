package controller

import (
	"errors"
	"strconv"
	"ticket/dao"
	"ticket/logic"
	"ticket/models"
	"ticket/utils/response"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

var (
	userLogic logic.UserLogic
)

// Public
func Register(c *gin.Context) {
	// 参数获取
	username := c.PostForm("username")
	password := c.PostForm("password")

	// 参数检查
	if len([]rune(username)) < 1 || len([]rune(username)) > 20 {
		response.JsonErr(c, 400, "用户名字符长度应在 1 ~ 20 以内")
		return
	}

	if len(password) < 6 || len(password) > 20 {
		response.JsonErr(c, 400, "密码长度应在 6 ~ 20 以内")
		return
	}

	// 调用逻辑层
	newUser, err := userLogic.Register(username, password)
	if err != nil {
		response.JsonErr(c, 400, err.Error())
	}

	// 成功注册响应
	response.JsonOK(c, "注册成功", gin.H{
		"userId":    newUser.ID,
		"username":  newUser.Username,
		"roleId":    newUser.RoleID,
		"createdAt": newUser.CreatedAt.Format(response.FmtTime),
	})
}

func Login(c *gin.Context) {
	// 数据库及参数获取
	username := c.PostForm("username")
	password := c.PostForm("password")

	// 调用逻辑层
	userInfo, err := userLogic.Login(c, username, password)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.JsonErr(c, 400, "用户不存在")
		}
		response.JsonErr(c, 400, "查询失败:"+err.Error())
	}

	// 成功登录响应
	response.JsonOK(c, "登录成功", gin.H{
		"userId":   userInfo["userId"],
		"username": userInfo["username"],
		"roleId":   userInfo["roleId"],
		"roleName": userInfo["roleName"],
		"token":    userInfo["token"],
	})
}

// Logined

func GetMyInfo(c *gin.Context) {
	// 获取数据库及参数
	db := dao.GetDB()
	roleID := c.GetInt("roleId")
	userID := c.GetInt64("userId")

	// 查找用户及角色
	var user models.User
	if err := db.First(&user, userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.JsonErr(c, 404, "指定用户不存在")
		} else {
			response.JsonErr(c, 400, "查询错误:"+err.Error())
		}
		return
	}

	var role models.Role
	if err := db.First(&role, roleID).Error; err != nil {
		response.JsonErr(c, 500, "角色信息不存在")
		return
	}

	response.JsonOK(c, "查询成功", gin.H{
		"userId":   user.ID,
		"username": user.Username,
		"role": response.RoleInfo{
			RoleId:   role.ID,
			RoleName: role.Name,
			RoleCode: role.Code,
		},
		"createdAt": user.CreatedAt.Format(response.FmtTime),
		"updatedAt": user.UpdatedAt.Format(response.FmtTime),
	})
}

func GetMyActivities(c *gin.Context) {
	var q models.ActivityQuery

	// 构建用户ID
	q.UserID = c.GetInt64("userId")

	// 构建活动ID
	q.ActivityID, _ = strconv.ParseInt(c.DefaultQuery("id", "0"), 10, 64)

	// 构建活动名称
	q.Name = c.DefaultQuery("name", "")

	// 状态条件(0-未开始 1-进行中 2-已结束 3-已下架)
	var statusList []int
	statusStr := c.QueryArray("status")
	for _, s := range statusStr {
		if st, err := strconv.Atoi(s); err == nil && st >= 0 && st < 3 {
			statusList = append(statusList, st)
		}
	}
	if len(statusList) == 0 {
		statusList = []int{0, 1, 2}
	}
	q.StatusList = statusList

	// 分页
	q.PageNum, q.PageSize = response.GetPage(c)

	// 调用逻辑层
	activityList, err := userLogic.GetMyActivities(q)
	if err != nil {
		response.JsonErr(c, 400, err.Error())
		return
	}

	// 成功响应
	var activities []gin.H
	for _, act := range activityList.Activities {
		activities = append(activities, gin.H{
			"activityId": act.ID,
			"name":       act.Name,
			"stock":      act.Stock,
			"status":     act.GetStatus(),
			"startTime":  act.StartTime.Format(response.FmtTime),
			"endTime":    act.EndTime.Format(response.FmtTime),
		})
	}

	response.JsonOK(c, "成功返回活动列表", gin.H{
		"activities": activities,
		"total":      activityList.Total,
		"pageNum":    q.PageNum,
		"pageSize":   q.PageSize,
	})
}

func GetMyOrders(c *gin.Context) {
	// 构建用户ID条件
	var q models.OrderQuery
	q.UserID = c.GetInt64("userId")

	// 活动ID条件
	q.ActivityID, _ = strconv.ParseInt(c.DefaultQuery("activityId", "0"), 10, 64)

	// 订单状态条件(0-未支付 1-已支付 2-已取消 3-已过期)
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
	q.StatusList = statusList

	// 分页
	q.PageNum, q.PageSize = response.GetPage(c)

	// 调用逻辑层
	orderList, err := userLogic.GetMyOrders(q)
	if err != nil {
		response.JsonErr(c, 400, err.Error())
		return
	}

	// 构建成功响应
	var orders []gin.H
	for _, order := range orderList.Orders {
		payTime := ""
		if order.PayTime != nil {
			payTime = order.PayTime.Format(response.FmtTime)
		}
		orders = append(orders, gin.H{
			"orderId":      order.ID,
			"status":       order.Status,
			"activityId":   order.ActivityId,
			"activityName": order.ActivityName,
			"createdAt":    order.CreatedAt.Format(response.FmtTime),
			"payTime":      payTime,
		})
	}

	response.JsonOK(c, "返回成功", gin.H{
		"orders":   orders,
		"total":    orderList.Total,
		"pageNum":  q.PageNum,
		"pageSize": q.PageSize,
	})
}

func GetMyTickets(c *gin.Context) {
	// 用户ID条件
	var q models.TicketQuery
	q.UserID = c.GetInt64("userId")

	// 订单ID条件
	q.OrderID, _ = strconv.ParseInt(c.DefaultQuery("orderId", "0"), 10, 64)

	// 构建活动ID条件
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
		statusList = []int{0}
	}
	q.StatusList = statusList

	// 分页
	q.PageNum, q.PageSize = response.GetPage(c)

	// 调用逻辑层
	ticketList, err := userLogic.GetMyTickets(q)
	if err != nil {
		response.JsonErr(c, 400, err.Error())
		return
	}

	// 构建成功响应体
	var tickets []gin.H
	for _, tkt := range ticketList.Tickets {
		tickets = append(tickets, gin.H{
			"ticketId":     tkt.ID,
			"activityId":   tkt.ActivityID,
			"activityName": tkt.ActivityName,
			"status":       tkt.Status,
		})
	}
	response.JsonOK(c, "成功返回票列表", gin.H{
		"tickets":  tickets,
		"total":    ticketList.Total,
		"pageNum":  q.PageNum,
		"pageSize": q.PageSize,
	})
}

func GetUserInfoByID(c *gin.Context) {
	// 获取参数
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.JsonErr(c, 400, "用户ID错误")
		return
	}

	// 调用逻辑层
	user, role, err := userLogic.GetUserInfoByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.JsonErr(c, 404, "用户不存在")
			return
		}
		response.JsonErr(c, 400, err.Error())
		return
	}

	response.JsonOK(c, "查询成功", gin.H{
		"userId":   user.ID,
		"username": user.Username,
		"role": response.RoleInfo{
			RoleId:   role.ID,
			RoleName: role.Name,
			RoleCode: role.Code,
		},
		"createdAt": user.CreatedAt.Format(response.FmtTime),
		"updatedAt": user.UpdatedAt.Format(response.FmtTime),
	})
}

// Admin

func GetRoles(c *gin.Context) {
	// 获取数据库及参数
	db := dao.GetDB()
	pageNum, pageSize := response.GetPage(c)

	// 查询总条数
	var total int64
	if err := db.Model(&models.Role{}).Count(&total).Error; err != nil {
		response.JsonErr(c, 500, "查询失败")
		return
	}

	// 分页查询
	var roles []models.Role
	offset := (pageNum - 1) * pageSize
	if err := db.Limit(pageSize).Offset(offset).Find(&roles).Error; err != nil {
		response.JsonErr(c, 500, "查询失败")
		return
	}

	// 成功响应
	response.JsonOK(c, "角色列表获取成功", gin.H{
		"roles":    roles,
		"total":    total,
		"pageNum":  pageNum,
		"pageSize": pageSize,
	})
}
