package logic

import (
	"errors"
	"ticket/dao"
	"ticket/models"

	"gorm.io/gorm"
)

type TicketLogic struct{}

func (l *TicketLogic) GetTicketDetail(ticketId int64) (*models.Ticket, error) {
	db := dao.GetDB()

	var ticket models.Ticket
	if err := db.Model(&models.Ticket{}).
		Select("`tickets`.*, `activities`.`name` AS `activity_name`").
		Joins("LEFT JOIN `activities` ON `activities`.`id` = `tickets`.`activity_id`").
		First(&ticket, ticketId).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, err
		}
		return nil, errors.New(err.Error() + "查询错误")
	}

	return &ticket, nil
}

func (l *TicketLogic) VerifyTicket(ticketId int64) (*models.Ticket, error) {
	db := dao.GetDB()

	// 检验票是否存在
	var ticket models.Ticket
	if err := db.First(&ticket, ticketId).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
		return nil, errors.New("查询错误:" + err.Error())
	}

	// 验参
	switch ticket.Status {
	case models.US:
		return nil, errors.New("该票已被使用")
	case models.IV:
		return nil, errors.New("该票已作废")
	case models.UD:
		// 正常并继续执行
	default:
		return nil, errors.New("该票状态错误")
	}

	return &ticket, nil
}

func (l *TicketLogic) InvalidateTicket(ticketId int64) (*models.Ticket, error) {
	db := dao.GetDB()
	// 检验票是否存在
	var ticket models.Ticket
	if err := db.First(&ticket, ticketId).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, err
		}
		return nil, errors.New("查询错误:" + err.Error())
	}

	// 状态检验
	if ticket.Status == models.IV {
		return nil, errors.New("门票不可重复作废")
	}

	// 修改
	if err := db.Model(&models.Ticket{}).Where("id = ?", ticketId).Update("status", models.IV).Error; err != nil {
		return nil, errors.New("作废门票时出错:" + err.Error())
	}

	ticket.Status = models.IV
	return &ticket, nil
}

func (l *TicketLogic) GetTickets(q models.TicketQuery) (*models.TicketList, error) {
	db := dao.GetDB()
	queryDB := db.Model(&models.Ticket{}).Joins("LEFT JOIN `activities` ON `activities`.`id` = `tickets`.`activity_id`")

	if q.OrderID > 0 {
		queryDB = queryDB.Where("`tickets`.`order_id` = ?", q.OrderID)
	}
	if q.ActivityID > 0 {
		queryDB = queryDB.Where("`tickets`.`activity_id` = ?", q.ActivityID)
	}
	queryDB = queryDB.Where("`tickets`.`status` IN (?)", q.StatusList)

	var ticketList models.TicketList
	if err := queryDB.Select("COUNT(DISTINCT `tickets`.`id`)").Scan(&ticketList.Total).Error; err != nil {
		return nil, errors.New(err.Error() + "查询错误")
	}
	if err := queryDB.
		Limit(q.PageSize).Offset((q.PageNum - 1) * q.PageSize).
		Order("`tickets`.`status` ASC, `tickets`.`created_at` ASC").
		Select("`tickets`.*, `activities`.`name` AS `activity_name`").
		Find(&ticketList.Tickets).Error; err != nil {
		return nil, errors.New(err.Error() + "查询错误")
	}

	return &ticketList, nil
}

func (l *TicketLogic) VerifyTicketNO(ticketNo string) (*models.Ticket, error) {
	db := dao.GetDB()
	// 检验票是否存在
	var ticket models.Ticket
	if err := db.Model(&models.Ticket{}).
		Select("`tickets`.*, `activities`.`name` AS `activity_name`").
		Joins("LEFT JOIN `activities` ON `activities`.`id` = `tickets`.`activity_id`").
		Where("`ticket_no` = ?", ticketNo).
		First(&ticket).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, err
		}
		return nil, errors.New("查询错误:" + err.Error())
	}

	// 检票
	if ticket.Status != models.UD {
		return nil, errors.New("票无效-无法检票")
	}

	if err := db.Model(&models.Ticket{}).
		Where("id = ?", ticket.ID).
		Update("status", models.US).Error; err != nil {
		return nil, errors.New("检票失败")
	}

	ticket.Status = models.US
	return &ticket, nil
}
