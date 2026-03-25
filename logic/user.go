package logic

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"ticket/dao"
	"ticket/models"
	"ticket/utils"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type UserLogic struct{}

func (l *UserLogic) Register(username, password string) (*models.User, error) {
	db := dao.GetDB()

	// 检查重复
	if err := db.First(&models.User{}, "username = ?", username).Error; err == nil {
		return nil, errors.New("用户名已存在")
	}

	// 加密
	hashedPsw, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, errors.New("密码加密失败")
	}

	newUser := models.User{
		Username: username,
		Password: string(hashedPsw),
		RoleID:   models.RoleUser,
	}

	if err := db.Create(&newUser).Error; err != nil {
		return nil, err
	}
	return &newUser, nil
}

func (l *UserLogic) Login(c context.Context, username, password string) (map[string]interface{}, error) {
	db := dao.GetDB()
	rdb := dao.GetRDB()

	// 检查用户名
	var user models.User
	if err := db.First(&user, "username = ?", username).Error; err != nil {
		return nil, err
	}

	// 验证密码
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return nil, errors.New("密码错误")
	}

	// 查询角色
	roleKey := fmt.Sprintf("cache:role:%d", user.RoleID)
	roleName, err := rdb.Get(c, roleKey).Result()
	if err != nil {
		var role models.Role
		if err := db.First(&role, user.RoleID).Error; err != nil {
			return nil, errors.New("角色信息异常")
		}
		roleName = role.Name
		rdb.Set(c, roleKey, roleName, time.Hour*24)
	}

	// 生成 token
	token, err := utils.GenerateToken(user.ID, user.RoleID)
	if err != nil {
		return nil, errors.New("生成 Token 失败")
	}

	tokenKey := fmt.Sprintf("auth:token:%d", user.ID)
	rdb.Set(c, tokenKey, token, time.Hour*24)

	return map[string]interface{}{
		"userId":   user.ID,
		"username": user.Username,
		"roleId":   user.RoleID,
		"roleName": roleName,
		"token":    token,
	}, nil
}

func (l *UserLogic) GetMyActivities(q models.ActivityQuery) (*models.ActivityList, error) {
	db := dao.GetDB()

	// 构建查询
	queryDB := db.Model(&models.Activity{}).
		Joins("INNER JOIN `tickets` ON `tickets`.`activity_id` = `activities`.`id`").
		Joins("INNER JOIN `orders` ON `orders`.`id` = `tickets`.`order_id`").
		Where("`orders`.`user_id` = ?", q.UserID).
		Where("`orders`.`status` = ?", models.PD).
		Distinct()

	if q.ActivityID > 0 {
		queryDB = queryDB.Where("`activities`.`id` = ?", q.ActivityID)
	}
	if q.Name != "" {
		queryDB = queryDB.Where("`activities`.`name` LIKE ?", "%"+q.Name+"%")
	}
	// 动态构建状态条件
	var conds []string
	var args []interface{}
	now := time.Now()
	for _, s := range q.StatusList {
		switch s {
		case models.NS:
			conds = append(conds, "start_time > ?")
			args = append(args, now)
		case models.IP:
			conds = append(conds, "(start_time <= ? AND end_time > ?)")
			args = append(args, now, now)
		case models.ED:
			conds = append(conds, "end_time <= ?")
			args = append(args, now)
		case models.RM:
			conds = append(conds, "deleted_at IS NOT NULL")
		}
	}
	queryDB = queryDB.Where("("+strings.Join(conds, " OR ")+")", args...)

	// 查询
	var activityList models.ActivityList
	if err := queryDB.Distinct("`activities`.`id`").Count(&activityList.Total).Error; err != nil {
		return nil, errors.New("查询总数错误:" + err.Error())
	}

	if err := queryDB.Limit(q.PageSize).Offset((q.PageNum - 1) * q.PageSize).
		Order("`activities`.`start_time` ASC").
		Select("`activities`.*").
		Find(&activityList.Activities).Error; err != nil {
		return nil, errors.New("查询错误:" + err.Error())
	}

	return &activityList, nil
}

func (l *UserLogic) GetMyOrders(q models.OrderQuery) (*models.OrderList, error) {
	db := dao.GetDB()
	queryDB := db.Model(&models.Order{}).
		Where("`orders`.`user_id` = ?", q.UserID).
		Joins("LEFT JOIN `tickets` ON `tickets`.`order_id` = `orders`.`id`").
		Joins("LEFT JOIN `activities` ON `activities`.`id` = `tickets`.`activity_id`")

	if q.ActivityID > 0 {
		queryDB = queryDB.Where("`tickets`.`activity_id` = ?", q.ActivityID)
	}
	queryDB = queryDB.Where("`orders`.`status` IN (?)", q.StatusList)

	var orderList models.OrderList
	if err := queryDB.Distinct("`orders`.`id`").Count(&orderList.Total).Error; err != nil {
		return nil, errors.New("查询错误:" + err.Error())
	}

	if err := queryDB.Limit(q.PageSize).Offset((q.PageNum - 1) * q.PageSize).
		Order("`orders`.`created_at` DESC").
		Select("`orders`.*, `activities`.`id` AS `activity_id`, `activities`.`name` AS `activity_name`").
		Find(&orderList.Orders).Error; err != nil {
		return nil, errors.New("查询错误:" + err.Error())
	}

	return &orderList, nil
}

func (l *UserLogic) GetMyTickets(q models.TicketQuery) (*models.TicketList, error) {
	db := dao.GetDB()
	queryDB := db.Model(&models.Ticket{}).
		Joins("LEFT JOIN `orders` ON `orders`.`id` = `tickets`.`order_id`").
		Joins("LEFT JOIN `activities` ON `activities`.`id` = `tickets`.`activity_id`").
		Where("`orders`.`user_id` = ?", q.UserID)

	if q.OrderID > 0 {
		queryDB = queryDB.Where("`orders`.`id` = ?", q.OrderID)
	}

	if q.ActivityID > 0 {
		queryDB = queryDB.Where("`tickets`.`activity_id` = ?", q.ActivityID)
	}
	queryDB = queryDB.Where("`tickets`.`status` IN (?)", q.StatusList)

	var ticketList models.TicketList
	if err := queryDB.Session(&gorm.Session{}).
		Distinct("`tickets`.`id`").
		Count(&ticketList.Total).Error; err != nil {
		return nil, errors.New("查询错误:" + err.Error())
	}

	if err := queryDB.Session(&gorm.Session{}).
		Limit(q.PageSize).Offset((q.PageNum - 1) * q.PageSize).
		Select("`tickets`.*, `activities`.`name` AS `activity_name`").
		Order("`tickets`.`status` ASC, `tickets`.`updated_at` ASC").
		Find(&ticketList.Tickets).Error; err != nil {
		return nil, errors.New("查询错误:" + err.Error())
	}

	return &ticketList, nil
}

func (l *UserLogic) GetUserInfoByID(userId int64) (*models.User, *models.Role, error) {
	db := dao.GetDB()

	var user models.User
	if err := db.First(&user, userId).Error; err != nil {
		return nil, nil, errors.New("查询错误:" + err.Error())
	}

	var role models.Role
	if err := db.First(&role, user.RoleID).Error; err != nil {
		return nil, nil, errors.New("查询错误:" + err.Error())
	}

	return &user, &role, nil
}
