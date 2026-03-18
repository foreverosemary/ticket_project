package controller

import (
	"strconv"
	"ticket/dao"
	"ticket/models"
	"ticket/utils/response"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func CreateActivity(c *gin.Context) {
	// 获取数据库及参数
	db := dao.GetDB()
	var newActivity models.Activity
	if err := c.ShouldBindJSON(&newActivity); err != nil {
		response.JsonErr(c, 400, "参数获取错误")
		return
	}

	newActivity.CreatorID = c.GetInt64("userId")

	// 检验合法性
	if err := newActivity.Verify(); err != nil {
		response.JsonErr(c, 400, err.Error())
		return
	}

	// 创建活动
	newActivity.SetStatus()
	newActivity.Stock = newActivity.Total
	if err := db.Create(&newActivity).Error; err != nil {
		response.JsonErr(c, 500, "创建活动失败，数据库错误")
		return
	}

	// 构建成功响应
	response.JsonOK(c, "创建活动成功", gin.H{
		"activityId": newActivity.ID,
		"status":     newActivity.Status,
		"createdAt":  newActivity.CreatedAt.Format(response.FmtTime),
	})
}

func UpdateAllActivity(c *gin.Context) {
	// 获取数据库及参数
	db := dao.GetDB().Unscoped()
	var input models.Activity
	if err := c.ShouldBindJSON(&input); err != nil {
		response.JsonErr(c, 400, "参数获取错误")
		return
	}

	activityId, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || activityId <= 0 {
		response.JsonErr(c, 400, "活动 ID 错误")
		return
	}

	// 检查活动是否存在
	var activity models.Activity
	if err := db.First(&activity, activityId).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			response.JsonErr(c, 404, "活动不存在")
			return
		}
		response.JsonErr(c, 500, "查询失败")
		return
	}

	// 参数检验
	if input.Total < activity.Total {
		response.JsonErr(c, 400, "只允许扩大总量")
		return
	}

	if err := input.Verify(); err != nil {
		response.JsonErr(c, 400, err.Error())
		return
	}

	// 更新活动
	if activity.Status != models.RM {
		input.SetStatus()
	} else {
		input.Status = models.RM
	}

	if err := db.Model(&activity).Updates(gin.H{
		"name":       input.Name,
		"content":    input.Content,
		"stock":      gorm.Expr("stock + ?", input.Total-activity.Total),
		"total":      input.Total,
		"start_time": input.StartTime,
		"end_time":   input.EndTime,
		"status":     input.Status,
	}).Error; err != nil {
		response.JsonErr(c, 500, "活动更新失败")
		return
	}

	// 构建成功响应
	db.First(&activity, activityId)
	response.JsonOK(c, "全量更新活动成功", gin.H{
		"activityId": activity.ID,
		"status":     activity.Status,
		"stock":      activity.Stock,
		"updatedAt":  activity.UpdatedAt.Format(response.FmtTime),
	})
}

func UpdatePartialActivity(c *gin.Context) {
	// 获取数据库及参数
	db := dao.GetDB().Unscoped()

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

	// 查询活动是否存在
	var activity models.Activity
	if err := db.First(&activity, activityId).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			response.JsonErr(c, 404, "活动不存在")
			return
		}
		response.JsonErr(c, 500, "查询失败")
		return
	}

	// 检验字段
	if dto.Total != nil && *dto.Total < activity.Total {
		response.JsonErr(c, 400, "只允许扩大总量")
		return
	}

	// 准备更新 Map
	oldTotal := activity.Total
	activity.ApplyUpdates(&dto)
	if err := activity.Verify(); err != nil {
		response.JsonErr(c, 400, err.Error())
		return
	}

	updates := make(map[string]interface{})
	if dto.Name != nil {
		updates["name"] = *dto.Name
	}
	if dto.Content != nil {
		updates["content"] = dto.Content
	}
	if dto.Total != nil {
		updates["total"] = *dto.Total
		updates["stock"] = gorm.Expr("stock + ?", *dto.Total-oldTotal)
	}
	if dto.StartTime != nil {
		updates["start_time"] = *dto.StartTime
	}
	if dto.EndTime != nil {
		updates["end_time"] = *dto.EndTime
	}
	if activity.Status != models.RM {
		updates["status"] = activity.Status
	}

	// 更新活动
	if err := db.Model(&activity).Updates(updates).Error; err != nil {
		response.JsonErr(c, 500, "更新失败")
		return
	}

	// 构建成功响应
	db.First(&activity, activityId) // 获取最新的 updatedAt
	response.JsonOK(c, "部分更新成功", gin.H{
		"activityId": activity.ID,
		"status":     activity.Status,
		"stock":      activity.Stock,
		"updatedAt":  activity.UpdatedAt.Format(response.FmtTime),
	})
}

func DeleteActivity(c *gin.Context) {
	// 获取数据库及参数
	db := dao.GetDB().Unscoped()
	activityId, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || activityId <= 0 {
		response.JsonErr(c, 400, "活动ID错误")
		return
	}

	// 检查活动是否存在
	var activity models.Activity
	if err := db.First(&activity, activityId).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			response.JsonErr(c, 404, "活动不存在")
			return
		}
		response.JsonErr(c, 500, "查询失败")
		return
	}

	if activity.Status == models.RM {
		response.JsonErr(c, 404, "活动重复删除")
		return
	}

	// 删除活动
	activity.Status = models.RM
	activity.DeletedAt = gorm.DeletedAt{
		Time:  time.Now(),
		Valid: true,
	}
	if err := db.Model(&activity).Updates(map[string]any{
		"status":     activity.Status,
		"deleted_at": activity.DeletedAt,
	}).Error; err != nil {
		response.JsonErr(c, 500, "删除失败")
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
	for _, aty := range activities {
		activityList = append(activityList, gin.H{
			"activityId": aty.ID,
			"name":       aty.Name,
			"total":      aty.Total,
			"stock":      aty.Stock,
			"status":     aty.Status,
			"startTime":  aty.StartTime.Format(response.FmtTime),
			"endTime":    aty.EndTime.Format(response.FmtTime),
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
	// 获取数据库及参数
	db := dao.GetDB().Unscoped()
	activityId, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || activityId <= 0 {
		response.JsonErr(c, 400, "活动ID错误")
		return
	}

	// 查询
	var activity models.Activity
	if err := db.First(&activity, activityId).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			response.JsonErr(c, 404, "活动不存在")
			return
		}
		response.JsonErr(c, 500, "查询失败")
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
