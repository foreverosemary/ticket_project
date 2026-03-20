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
	actLogic logic.ActivityLogic
)

func CreateActivity(c *gin.Context) {
	// 获取参数
	var newActivity models.Activity
	if err := c.ShouldBindJSON(&newActivity); err != nil {
		response.JsonErr(c, 400, "参数获取错误")
		return
	}

	newActivity.CreatorID = c.GetInt64("userId")

	// 调用逻辑层
	activity, err := actLogic.CreateActivity(c, newActivity)
	if err != nil {
		response.JsonErr(c, 400, err.Error())
		return
	}

	// 构建成功响应
	response.JsonOK(c, "创建活动成功", gin.H{
		"activityId": activity.ID,
		"status":     activity.Status,
		"createdAt":  activity.CreatedAt.Format(response.FmtTime),
	})
}

func UpdateAllActivity(c *gin.Context) {
	// 获取参数
	activityId, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || activityId <= 0 {
		response.JsonErr(c, 400, "活动 ID 错误")
		return
	}

	var input models.Activity
	if err := c.ShouldBindJSON(&input); err != nil {
		response.JsonErr(c, 400, "参数获取错误")
		return
	}

	// 调用逻辑层
	updatedAct, err := actLogic.UpdateAllActivity(c, activityId, input)

	// 处理错误
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.JsonErr(c, 404, "活动不存在")
			return
		}
		response.JsonErr(c, 400, err.Error())
		return
	}

	// 构建成功响应
	response.JsonOK(c, "全量更新活动成功", gin.H{
		"activityId": updatedAct.ID,
		"status":     updatedAct.Status,
		"stock":      updatedAct.Stock,
		"updatedAt":  updatedAct.UpdatedAt.Format(response.FmtTime),
	})
}

func UpdatePartialActivity(c *gin.Context) {
	// 获取参数
	var dto models.UpdateActivityDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		response.JsonErr(c, 400, "参数获取错误")
		return
	}

	// 活动 ID 校验
	activityId, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || activityId <= 0 {
		response.JsonErr(c, 400, "活动 ID 错误")
		return
	}

	// 调用逻辑层
	updatedAct, err := actLogic.UpdatePartialActivity(c, activityId, dto)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.JsonErr(c, 404, "活动不存在")
			return
		}
		response.JsonErr(c, 400, err.Error())
		return
	}

	// 构建成功响应
	response.JsonOK(c, "部分更新成功", gin.H{
		"activityId": updatedAct.ID,
		"status":     updatedAct.Status,
		"stock":      updatedAct.Stock,
		"updatedAt":  updatedAct.UpdatedAt.Format(response.FmtTime),
	})
}

func DeleteActivity(c *gin.Context) {
	// 获取参数
	activityId, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || activityId <= 0 {
		response.JsonErr(c, 400, "活动ID错误")
		return
	}

	// 调用逻辑层
	activity, err := actLogic.DeleteActivity(c, activityId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.JsonErr(c, 404, "活动不存在")
			return
		}
		response.JsonErr(c, 400, err.Error())
		return
	}

	// 构建成功响应
	response.JsonOK(c, "删除成功", gin.H{
		"activityId": activity.ID,
		"status":     activity.Status,
		"deletedAt":  activity.DeletedAt.Time.Format(response.FmtTime),
	})
}

func GetActivities(c *gin.Context) {
	// 获取数据库
	db := dao.GetDB().Unscoped()
	queryDB := db.Model(&models.Activity{})

	// 活动 ID 条件构建
	activityId, _ := strconv.ParseInt(c.DefaultQuery("id", "0"), 10, 64)
	if activityId > 0 {
		queryDB = queryDB.Where("`activities`.`id` = ?", activityId)
	}

	// 活动名称条件构建
	name := c.DefaultQuery("name", "")
	if name != "" {
		queryDB = queryDB.Where("`activities`.`name` LIKE ?", "%"+name+"%")
	}

	// 活动状态条件构建
	roleId := c.GetInt64("roleId")
	var statusList []int
	statusStr := c.QueryArray("status")
	for _, s := range statusStr {
		st, err := strconv.Atoi(s)
		if err != nil || st < models.NS || st > models.RM {
			continue
		}
		if st == models.RM && roleId == models.RoleUser {
			response.JsonErr(c, 403, "无权限获取已下架的活动")
			return
		}
		statusList = append(statusList, st)
	}
	if len(statusList) == 0 {
		statusList = []int{models.ED, models.IP, models.NS}
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
	if err := queryDB.Count(&total).Error; err != nil {
		response.JsonErr(c, 500, "查询失败")
		return
	}

	var activities []models.Activity
	offset := (pageNum - 1) * pageSize
	if err := queryDB.
		Limit(pageSize).Offset(offset).
		Order("`activities`.`status` ASC, `activities`.`start_time` ASC").
		Find(&activities).Error; err != nil {
		response.JsonErr(c, 500, "查询失败")
		return
	}

	// 构建成功响应
	var activityList []gin.H
	for _, act := range activities {
		activityList = append(activityList, gin.H{
			"activityId": act.ID,
			"name":       act.Name,
			"total":      act.Total,
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

func GetActivityDetail(c *gin.Context) {
	// 获取参数
	activityId, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || activityId <= 0 {
		response.JsonErr(c, 400, "活动ID错误")
		return
	}

	// 调用逻辑层
	activity, err := actLogic.GetActivityDetail(c, activityId)
	if err != nil {
		response.JsonErr(c, 400, err.Error())
		return
	}

	// 鉴权
	if activity.Status == models.RM && c.GetInt("roleId") == models.RoleUser {
		response.JsonErr(c, 403, "活动已下架")
		return
	}

	// 构建成功响应
	response.JsonOK(c, "成功返回活动详情", gin.H{
		"activityId": activity.ID,
		"name":       activity.Name,
		"total":      activity.Total,
		"stock":      activity.Stock,
		"status":     activity.Status,
		"startTime":  activity.StartTime.Format(response.FmtTime),
		"endTime":    activity.EndTime.Format(response.FmtTime),
		"content":    activity.Content,
	})
}
