package controller

import (
	"strconv"
	"ticket/dao"
	"ticket/models"
	"ticket/utils"
	"ticket/utils/response"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// Public
func Register(c *gin.Context) {
	// 数据库及参数获取
	db := dao.GetDB()
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

	// 数据库查找
	if err := db.First(&models.User{}, "username = ?", username).Error; err == nil {
		response.JsonErr(c, 409, "用户名已存在")
		return
	}

	// 密码加密
	hashedPsw, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		response.JsonErr(c, 500, "密码加密失败")
		return
	}

	// 构造用户数据并插入
	newUser := models.User{
		Username: username,
		Password: string(hashedPsw),
		RoleID:   models.RoleUser,
	}
	if err := db.Create(&newUser).Error; err != nil {
		response.JsonErr(c, 500, "注册失败，数据库错误")
		return
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
	db := dao.GetDB()
	username := c.PostForm("username")
	password := c.PostForm("password")

	// 检查用户名是否存在
	var user models.User
	if err := db.First(&user, "username = ?", username).Error; err != nil {
		response.JsonErr(c, 401, "用户不存在")
		return
	}

	// 检查密码是否正确
	err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		response.JsonErr(c, 401, "密码错误")
		return
	}

	// 生成 Token
	token, err := utils.GenerateToken(user.ID, user.RoleID)
	if err != nil {
		response.JsonErr(c, 500, "生成 Token 失败")
		return
	}

	// 查询角色
	var role models.Role
	if err := db.First(&role, user.RoleID).Error; err != nil {
		response.JsonErr(c, 500, "角色信息不存在")
		return
	}

	// 成功登录响应
	response.JsonOK(c, "登录成功", gin.H{
		"userId":   user.ID,
		"username": user.Username,
		"roleId":   user.RoleID,
		"roleName": role.Name,
		"token":    token,
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
		response.JsonErr(c, 404, "指定用户不存在")
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
	// 获取数据库
	db := dao.GetDB()
	queryDB := db.Model(&models.Activity{})

	// 构建用户ID条件
	userId := c.GetInt64("userId")
	queryDB = queryDB.
		Joins("INNER JOIN `tickets` ON `tickets`.`activity_id` = `activities`.`id`").
		Joins("INNER JOIN `orders` ON `orders`.`id` = `tickets`.`order_id`").
		Where("`orders`.`user_id` = ?", userId).
		Where("`orders`.`status` = ?", models.PD).
		Not("`activities`.`status` = ?", models.RM).
		Distinct()

	// 构建活动ID条件
	activityId, _ := strconv.ParseInt(c.DefaultQuery("id", "0"), 10, 64)
	if activityId > 0 {
		queryDB = queryDB.Where("`activities`.`id` = ?", activityId)
	}

	// 构建活动名称条件
	name := c.DefaultQuery("name", "")
	if name != "" {
		queryDB = queryDB.Where("`activities`.`name` LIKE ?", "%"+name+"%")
	}

	// 构建状态条件(0-未开始 1-进行中 2-已结束 3-已下架)
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
	queryDB = queryDB.Where("`activities`.`status` IN (?)", statusList)

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
	if err := queryDB.Select("COUNT(DISTINCT `activities`.`id`)").Scan(&total).Error; err != nil {
		response.JsonErr(c, 500, "查询失败")
		return
	}
	var activities []models.Activity
	offset := (pageNum - 1) * pageSize
	if err := queryDB.
		Limit(pageSize).Offset(offset).
		Order("`activities`.`start_time` ASC").
		Find(&activities).Error; err != nil {
		response.JsonErr(c, 500, "查询失败")
		return
	}

	// 成功响应
	var activityList []gin.H
	for _, act := range activities {
		activityList = append(activityList, gin.H{
			"activityId": act.ID,
			"name":       act.Name,
			"stock":      act.Stock,
			"status":     act.Status,
			"startTime":  act.StartTime.Format(response.FmtTime),
			"endTime":    act.EndTime.Format(response.FmtTime),
		})
	}

	response.JsonOK(c, "成功返回活动列表", gin.H{
		"activities": activityList,
		"total":      total,
		"pageNum":    pageNum,
		"pageSize":   pageSize,
	})
}

func GetMyOrders(c *gin.Context) {
	// 获取数据库
	db := dao.GetDB()
	queryDB := db.Model(&models.Order{})

	// 构建用户ID条件
	userId := c.GetInt64("userId")
	queryDB = queryDB.
		Where("`orders`.`user_id` = ?", userId).
		Joins("LEFT JOIN `tickets` ON `tickets`.`order_id` = `orders`.`id`").
		Joins("LEFT JOIN `activities` ON `activities`.`id` = `tickets`.`activity_id`")

	// 构建活动ID条件
	activityId, _ := strconv.ParseInt(c.DefaultQuery("activityId", "0"), 10, 64)
	if activityId > 0 {
		queryDB = queryDB.Where("`tickets`.`activity_id` = ?", activityId)
	}

	// 构建订单状态条件(0-未支付 1-已支付 2-已取消 3-已过期)
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
	queryDB = queryDB.Where("`orders`.`status` IN (?)", statusList)

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
	if err := queryDB.Select("COUNT(DISTINCT `orders`.`id`)").Scan(&total).Error; err != nil {
		response.JsonErr(c, 500, "查询失败")
		return
	}
	var orders []models.Order
	offset := (pageNum - 1) * pageSize
	if err := queryDB.
		Limit(pageSize).Offset(offset).
		Order("`orders`.`created_at` DESC").
		Select("`orders`.*, `activities`.`id` AS `activity_id`, `activities`.`name` AS `activity_name`").
		Find(&orders).Error; err != nil {
		response.JsonErr(c, 500, "查询失败")
		return
	}

	// 构建成功响应
	var orderList []gin.H
	for _, order := range orders {
		orderList = append(orderList, gin.H{
			"orderId":      order.ID,
			"status":       order.Status,
			"activityId":   order.ActivityId,
			"activityName": order.ActivityName,
			"createdAt":    order.CreatedAt.Format(response.FmtTime),
			"payTime":      order.PayTime.Format(response.FmtTime),
		})
	}

	response.JsonOK(c, "返回成功", gin.H{
		"orders":   orderList,
		"total":    total,
		"pageNum":  pageNum,
		"pageSize": pageSize,
	})
}

func GetMyTickets(c *gin.Context) {
	// 获取数据库
	db := dao.GetDB()
	queryDB := db.Model(&models.Ticket{})

	// 构建用户ID条件
	userId := c.GetInt64("userId")
	queryDB = queryDB.
		Where("`tickets`.`user_id` = ?", userId).
		Joins("LEFT JOIN `activities` ON `activities`.`id` = `tickets`.`activity_id`")

	// 构建订单ID条件
	orderId, err := strconv.ParseInt(c.DefaultQuery("orderId", "0"), 10, 64)
	if err == nil && orderId > 0 {
		queryDB = queryDB.Where("`tickets`.`order_id` = ?", orderId)
	}

	// 构建活动ID条件
	activityId, err := strconv.ParseInt(c.DefaultQuery("activityId", "0"), 10, 64)
	if err == nil && activityId > 0 {
		queryDB = queryDB.Where("`tickets`.`activity_id` = ?", activityId)
	}

	// 构建状态条件
	var statusList []int
	statusStr := c.QueryArray("status")
	for _, s := range statusStr {
		if st, err := strconv.Atoi(s); err == nil && st >= 0 && st < 2 {
			statusList = append(statusList, st)
		}
	}
	if len(statusList) == 0 {
		statusList = []int{0}
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
		Select("`tickets`.*, `activities`.`name` AS `activity_name`").
		Order("`tickets`.`status` ASC, `tickets`.`updated_at` ASC").
		Find(&tickets).Error; err != nil {
		response.JsonErr(c, 500, "查询失败")
		return
	}

	// 构建成功响应体
	var ticketList []gin.H
	for _, tkt := range tickets {
		ticketList = append(ticketList, gin.H{
			"ticketId":     tkt.ID,
			"activityId":   tkt.ActivityID,
			"activityName": tkt.ActivityName,
			"status":       tkt.Status,
		})
	}
	response.JsonOK(c, "成功返回票列表", gin.H{
		"tickets":  ticketList,
		"total":    total,
		"pageNum":  pageNum,
		"pageSize": pageSize,
	})
}

func GetUserInfoByID(c *gin.Context) {
	// 获取数据库及参数
	db := dao.GetDB()
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.JsonErr(c, 400, "用户ID错误")
		return
	}

	// 查找用户及角色
	var user models.User
	if err := db.First(&user, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			response.JsonErr(c, 404, "用户不存在")
			return
		}
		response.JsonErr(c, 500, "查询失败")
		return
	}

	var role models.Role
	if err := db.First(&role, user.RoleID).Error; err != nil {
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

// Admin

func GetRoles(c *gin.Context) {
	// 获取数据库及参数
	db := dao.GetDB()
	pageNum, err := strconv.Atoi(c.DefaultQuery("pageNum", "1"))
	if err != nil || pageNum < 1 {
		pageNum = 1
	}

	pageSize, err := strconv.Atoi(c.DefaultQuery("pageSize", "10"))
	if err != nil || pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}

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
