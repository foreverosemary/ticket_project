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
	actLogic logic.ActivityLogic
)

func CreateActivity(c *gin.Context) {
	// 获取参数
	ctx := c.Request.Context()
	var newActivity models.Activity
	if err := c.ShouldBindJSON(&newActivity); err != nil {
		response.JsonErr(c, 400, "参数获取错误:"+err.Error())
		return
	}

	newActivity.CreatorID = c.GetInt64("userId")

	// 调用逻辑层
	activity, err := actLogic.CreateActivity(ctx, newActivity)
	if err != nil {
		response.JsonErr(c, 400, err.Error())
		return
	}

	// 构建成功响应
	response.JsonOK(c, "创建活动成功", gin.H{
		"activityId": activity.ID,
		"status":     activity.GetStatus(),
		"createdAt":  activity.CreatedAt.Format(response.FmtTime),
	})
}

func UpdateAllActivity(c *gin.Context) {
	// 获取参数
	ctx := c.Request.Context()
	activityId, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || activityId <= 0 {
		response.JsonErr(c, 400, "活动 ID 错误")
		return
	}

	var input models.Activity
	if err := c.ShouldBindJSON(&input); err != nil {
		response.JsonErr(c, 400, "参数获取错误:"+err.Error())
		return
	}

	// 调用逻辑层
	updatedAct, err := actLogic.UpdateAllActivity(ctx, activityId, input)

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
		"status":     updatedAct.GetStatus(),
		"stock":      updatedAct.Stock,
		"updatedAt":  updatedAct.UpdatedAt.Format(response.FmtTime),
	})
}

func UpdatePartialActivity(c *gin.Context) {
	// 获取参数
	ctx := c.Request.Context()
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
	updatedAct, err := actLogic.UpdatePartialActivity(ctx, activityId, dto)
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
		"status":     updatedAct.GetStatus(),
		"stock":      updatedAct.Stock,
		"updatedAt":  updatedAct.UpdatedAt.Format(response.FmtTime),
	})
}

func DeleteActivity(c *gin.Context) {
	// 获取参数
	ctx := c.Request.Context()
	activityId, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || activityId <= 0 {
		response.JsonErr(c, 400, "活动ID错误")
		return
	}

	// 调用逻辑层
	activity, err := actLogic.DeleteActivity(ctx, activityId)
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
		"status":     activity.GetStatus(),
		"deletedAt":  activity.DeletedAt.Time.Format(response.FmtTime),
	})
}

func GetActivities(c *gin.Context) {
	var q models.ActivityQuery
	ctx := c.Request.Context()
	// 活动 ID 条件
	activityId, _ := strconv.ParseInt(c.DefaultQuery("id", "0"), 10, 64)
	q.ActivityID = activityId

	// 活动名称条件
	name := c.DefaultQuery("name", "")
	q.Name = name

	// 活动状态条件
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
	q.StatusList = statusList

	// 分页
	q.PageNum, q.PageSize = response.GetPage(c)

	// 调用逻辑层
	activityList, err := actLogic.GetActivities(ctx, q)
	if err != nil {
		response.JsonErr(c, 400, err.Error())
		return
	}

	// 构建成功响应
	var activities []gin.H
	for _, act := range activityList.Activities {
		activities = append(activities, gin.H{
			"activityId": act.ID,
			"name":       act.Name,
			"total":      act.Total,
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

func GetActivityDetail(c *gin.Context) {
	// 获取参数
	ctx := c.Request.Context()
	activityId, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || activityId <= 0 {
		response.JsonErr(c, 400, "活动ID错误")
		return
	}

	// 调用逻辑层
	activity, err := actLogic.GetActivityDetail(ctx, activityId)
	if err != nil {
		response.JsonErr(c, 400, err.Error())
		return
	}

	// 鉴权
	if activity.GetStatus() == models.RM && c.GetInt("roleId") == models.RoleUser {
		response.JsonErr(c, 403, "活动已下架")
		return
	}

	// 构建成功响应
	response.JsonOK(c, "成功返回活动详情", gin.H{
		"activityId": activity.ID,
		"name":       activity.Name,
		"total":      activity.Total,
		"stock":      activity.Stock,
		"status":     activity.GetStatus(),
		"startTime":  activity.StartTime.Format(response.FmtTime),
		"endTime":    activity.EndTime.Format(response.FmtTime),
		"content":    activity.Content,
	})
}
