package logic

import (
	"context"
	"errors"
	"strconv"
	"ticket/dao"
	"ticket/models"
	"ticket/utils/response"
	"time"

	"gorm.io/gorm"
)

type OrderLogic struct{}

func (l *OrderLogic) CreateOrder(c context.Context, activityId, userId int64, need int) (map[string]interface{}, error) {
	db := dao.GetDB().WithContext(c)
	rdb := dao.GetRDB()
	script := dao.Script

	// 检验活动状态
	var activity models.Activity
	if err := db.First(&activity, activityId).Error; err != nil {
		return nil, errors.New("活动查询失败" + err.Error())
	}
	if activity.GetStatus() == models.ED {
		return nil, errors.New("活动已结束")
	} else if activity.GetStatus() == models.RM {
		return nil, errors.New("活动已删除")
	}

	// 变量声明并执行脚本
	idStr := strconv.FormatInt(activityId, 10)
	keys := []string{"activity:stock:" + idStr, "activity:user:set:" + idStr}
	args := []interface{}{userId, need}

	res, err := script.Run(c, rdb, keys, args...).Int()
	if err != nil {
		return nil, errors.New("脚本执行错误:" + err.Error())
	} else if res == -1 {
		return nil, errors.New("不允许重复下单")
	} else if res == 0 {
		return nil, errors.New("库存不足")
	}

	// 创建订单
	var order models.Order
	order.UserID = userId
	if err := db.Create(&order).Error; err != nil {
		// 手动回滚
		rdb.IncrBy(c, keys[1], int64(need))
		rdb.SRem(c, keys[2], userId)
		return nil, errors.New("订单创建失败:" + err.Error())
	}

	// 发送消息
	for i := 1; i <= 5; i++ {
		if err := ProduceTicketMsg(order.ID, activityId, need); err != nil {
			if i == 5 {
				return nil, errors.New("消息发送失败:" + err.Error())
			}
			time.Sleep(50 * time.Millisecond)
			continue
		}
		break
	}

	// 返回信息
	return map[string]interface{}{
		"orderId":      order.ID,
		"activityId":   activity.ID,
		"activityName": activity.Name,
		"createdAt":    order.CreatedAt,
	}, nil
}

func (l *OrderLogic) UpdateOrder(c context.Context, orderId, userId int64, status int) error {
	db := dao.GetDB().Unscoped()
	rdb := dao.GetRDB()

	var order models.Order
	if err := db.First(&order, orderId).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return err
		}
		return errors.New("查询失败:" + err.Error())
	}

	if order.UserID != userId {
		return errors.New("不可修改其它人的订单")
	}

	if order.Status == models.CL {
		return errors.New("不可修改已取消订单")
	}
	if order.Status == int8(status) {
		return errors.New("订单状态已是目标状态")
	}

	var ticket models.Ticket
	if err := db.First(&ticket, "tickets.order_id = ?", orderId).Error; err != nil {
		return errors.New("查询失败:" + err.Error())
	}

	// 支付订单
	if status == models.PD {
		return db.Transaction(func(tx *gorm.DB) error {
			if e := tx.Model(&models.Order{}).
				Where("id = ?", orderId).
				Updates(map[string]interface{}{
					"status":   models.PD,
					"pay_time": time.Now(),
				}).Error; e != nil {
				return e
			}
			return nil
		})
	}

	// 取消订单
	var total int64
	if err := db.Model(&models.Ticket{}).Where("order_id = ?", orderId).Count(&total).Error; err != nil {
		return errors.New("统计门票失败:" + err.Error())
	}

	err := db.Transaction(func(tx *gorm.DB) error {
		if e := tx.Model(&models.Order{}).
			Where("id = ?", orderId).
			Updates(map[string]interface{}{
				"status":     status,
				"deleted_at": time.Now(),
			}).Error; e != nil {
			return e
		}

		if e := tx.Model(&models.Activity{}).
			Where("id = ?", ticket.ActivityID).
			Update("stock", gorm.Expr("stock + ?", total)).Error; e != nil {
			return e
		}

		// 设置门票作废
		if e := tx.Model(&models.Ticket{}).
			Where("order_id = ?", orderId).
			Updates(map[string]interface{}{
				"status":     models.IV,
				"deleted_at": time.Now(),
			}).Error; e != nil {
			return e
		}

		return nil
	})
	if err != nil {
		return errors.New("订单删除失败:" + err.Error())
	}

	// 修改缓存
	idStr := strconv.FormatInt(ticket.ActivityID, 10)
	rdb.IncrBy(c, "activity:stock:"+idStr, total)
	rdb.SRem(c, "activity:user:set:"+idStr, userId)

	return nil
}

func (l *OrderLogic) GetOrders(q models.OrderQuery) (*models.OrderList, error) {
	db := dao.GetDB().Unscoped()

	// 构建查询
	queryDB := db.Model(&models.Order{}).
		Joins("LEFT JOIN `tickets` ON `tickets`.`order_id` = `orders`.`id`").
		Joins("LEFT JOIN `activities` ON `activities`.`id` = `tickets`.`activity_id`")

	if q.OrderID >= 0 {
		queryDB = queryDB.Where("`orders`.`id` = ?", q.OrderID)
	}

	if q.UserID >= 0 {
		queryDB = queryDB.Where("`orders`.`user_id` = ?", q.UserID)
	}

	if q.ActivityID >= 0 {
		queryDB = queryDB.Where("`tickets`.`activity_id` = ?", q.ActivityID)
	}

	queryDB = queryDB.Where("`orders`.`status` IN (?)", q.StatusList)

	// 查询
	var orderList models.OrderList
	if err := queryDB.Distinct("`orders`.`id`").Count(&orderList.Total).Error; err != nil {
		return nil, errors.New("查询活动总数错误:" + err.Error())
	}

	if err := queryDB.Limit(q.PageSize).Offset((q.PageNum - 1) * q.PageSize).
		Select("`orders`.*, `activities`.`name` AS `activity_name`, `activities`.`id` AS `activity_id`").
		Find(&orderList.Orders).Error; err != nil {
		return nil, errors.New("查询错误" + err.Error())
	}

	return &orderList, nil
}

func (l *OrderLogic) GetOrderDetail(orderId int64) (map[string]interface{}, error) {
	db := dao.GetDB().
		Joins("LEFT JOIN `tickets` ON `tickets`.`activity_id` = `activities`.`id`").
		Joins("LEFT JOIN `activities` ON `activities`.`id` = `tickets`.`activity_id`")

	// 查询
	var order models.Order
	var tickets []models.Ticket

	if err := db.Where("id = ?", orderId).
		Select("`orders`.*, `activities`.`id` AS `activityId`, `activities`.`name` AS `activityName`").
		First(&order).Error; err != nil {
		return nil, errors.New("订单查询错误:" + err.Error())
	}

	if err := db.Where("order_id = ?", orderId).Find(&tickets).Error; err != nil {
		return nil, errors.New("订单对应的门票查询错误:" + err.Error())
	}

	// 返回
	var ticketInfo []map[string]interface{}
	for _, ticket := range tickets {
		ticketInfo = append(ticketInfo, map[string]interface{}{
			"ticketId": ticket.ID,
			"ticketNo": ticket.TicketNo,
		})
	}

	return map[string]interface{}{
		"orderId":      order.ID,
		"tickets":      ticketInfo,
		"status":       order.Status,
		"activityId":   order.ActivityId,
		"activityName": order.ActivityName,
		"createdAt":    order.CreatedAt.Format(response.FmtTime),
		"payTime":      order.PayTime,
	}, nil
}
