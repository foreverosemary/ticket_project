package logic

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"ticket/dao"
	"ticket/models"
	"time"

	"gorm.io/gorm"
)

type ActivityLogic struct{}

func (l *ActivityLogic) CreateActivity(c context.Context, input models.Activity) (*models.Activity, error) {
	db := dao.GetDB().WithContext(c)
	rdb := dao.GetRDB()

	// 检验
	if err := input.Verify(); err != nil {
		return nil, err
	}

	input.Stock = input.Total
	if err := db.Create(&input).Error; err != nil {
		return nil, errors.New("活动创建失败")
	}

	// 添加缓存
	idStr := strconv.FormatInt(input.ID, 10)
	rdb.Set(c, "activity:stock:"+idStr, input.Stock, 0)

	// 获取更新并返回
	var activity models.Activity
	if err := db.First(&activity, input.ID).Error; err != nil {
		return nil, errors.New("活动创建成功但返回数据失败")
	}
	return &activity, nil
}

func (l *ActivityLogic) UpdateAllActivity(c context.Context, id int64, input models.Activity) (*models.Activity, error) {
	db := dao.GetDB().WithContext(c).Unscoped()
	rdb := dao.GetRDB()

	// 检验是否存在
	var activity models.Activity
	if err := db.First(&activity, id).Error; err != nil {
		return nil, err
	}

	// 校验
	if input.Total < activity.Total {
		return nil, errors.New("活动总量只允许扩大")
	}
	if err := input.Verify(); err != nil {
		return nil, err
	}

	// 更新活动
	diff := input.Total - activity.Total
	err := db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&activity).Updates(map[string]interface{}{
			"name":       input.Name,
			"content":    input.Content,
			"stock":      gorm.Expr("stock + ?", diff),
			"total":      input.Total,
			"start_time": input.StartTime.ToTime(),
			"end_time":   input.EndTime.ToTime(),
		}).Error; err != nil {
			return errors.New("活动更新失败:" + err.Error())
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	// 维护缓存同步
	idStr := strconv.FormatInt(id, 10)
	rdb.Del(c, "activity:detail:"+idStr)
	if diff != 0 {
		rdb.IncrBy(c, "activity:stock:"+idStr, int64(diff))
	}

	// 获取最新结果
	if err := db.First(&activity, id).Error; err != nil {
		return nil, errors.New("更新活动成功但返回数据失败")
	}
	return &activity, nil
}

func (l *ActivityLogic) UpdatePartialActivity(c context.Context, id int64, dto models.UpdateActivityDTO) (*models.Activity, error) {
	db := dao.GetDB().WithContext(c).Unscoped()
	rdb := dao.GetRDB()

	var activity models.Activity
	if err := db.First(&activity, id).Error; err != nil {
		return nil, err
	}

	// 部分更新事务
	oldTotal := activity.Total
	activity.ApplyUpdates(&dto)
	if err := activity.Verify(); err != nil {
		return nil, err
	}

	diff := 0
	if dto.Total != nil {
		diff = *dto.Total - oldTotal
		if diff < 0 {
			return nil, errors.New("只允许扩大容量")
		}
	}

	err := db.Transaction(func(tx *gorm.DB) error {
		updates := make(map[string]interface{})
		if dto.Name != nil {
			updates["name"] = *dto.Name
		}
		if dto.Content != nil {
			updates["content"] = dto.Content
		}
		if dto.StartTime != nil {
			updates["start_time"] = *dto.StartTime
		}
		if dto.EndTime != nil {
			updates["end_time"] = *dto.EndTime
		}
		if dto.Total != nil {
			updates["total"] = *dto.Total
			updates["stock"] = gorm.Expr("stock + ?", diff)
		}
		if err := tx.Model(&activity).Updates(updates).Error; err != nil {
			return errors.New("活动更新失败")
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	// 维护缓存同步
	idStr := strconv.FormatInt(id, 10)
	rdb.Del(c, "activity:detail:"+idStr)
	if diff != 0 {
		rdb.IncrBy(c, "activity:stock:"+idStr, int64(diff))
	}

	// 获取最新结果
	if err := db.First(&activity, id).Error; err != nil {
		return nil, errors.New("更新活动成功但返回数据失败")
	}
	return &activity, nil
}

func (l *ActivityLogic) DeleteActivity(c context.Context, id int64) (*models.Activity, error) {
	db := dao.GetDB().WithContext(c)
	rdb := dao.GetRDB()

	// 检验是否存在
	var activity models.Activity
	if err := db.First(&activity, id).Error; err != nil {
		return nil, err
	}

	if activity.GetStatus() == models.RM {
		return nil, errors.New("活动重复删除")
	}

	// 删除活动事务
	err := db.Transaction(func(tx *gorm.DB) error {
		return tx.Model(&activity).Updates(map[string]interface{}{
			"status":     models.RM,
			"deleted_at": time.Now(),
		}).Error
	})

	if err != nil {
		return nil, errors.New("活动状态更新失败")
	}

	// 删除缓存
	idStr := strconv.FormatInt(id, 10)
	rdb.Del(c, "activity:detail:"+idStr)
	rdb.Del(c, "activity:stock:"+idStr)

	// 发送消息
	for i := 1; i <= 5; i++ {
		if err := ProduceActivityMsg(rdb, activity.ID); err != nil {
			if i == 5 {
				return nil, errors.New("消息发送失败:" + err.Error())
			}
			time.Sleep(50 * time.Millisecond)
			continue
		}
		break
	}

	// 获取最新结果
	if err := db.Unscoped().First(&activity, id).Error; err != nil {
		return nil, errors.New("删除活动成功但是返回数据失败" + err.Error())
	}

	return &activity, nil
}

func (l *ActivityLogic) GetActivities(c context.Context, q models.ActivityQuery) (*models.ActivityList, error) {
	db := dao.GetDB().WithContext(c).Unscoped()

	// 构建查询
	queryDB := db.Model(&models.Activity{})

	if q.ActivityID > 0 {
		queryDB = queryDB.Where("id = ?", q.ActivityID)
	}

	if q.Name != "" {
		queryDB = queryDB.Where("name LIKE ?", "%"+q.Name+"%")
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
	queryDB = queryDB.Where("("+strings.Join(conds, "OR")+")", args...)

	// 查询
	var activityList models.ActivityList
	if err := queryDB.Count(&activityList.Total).Error; err != nil {
		return nil, errors.New("查询失败:" + err.Error())
	}

	if err := queryDB.
		Limit(q.PageSize).Offset((q.PageNum - 1) * q.PageSize).
		Order("start_time ASC").
		Find(&activityList.Activities).Error; err != nil {
		return nil, errors.New("查询活动列表失败:" + err.Error())
	}

	return &activityList, nil
}

func (l *ActivityLogic) GetActivityDetail(c context.Context, id int64) (*models.Activity, error) {
	db := dao.GetDB().WithContext(c).Unscoped()
	rdb := dao.GetRDB()

	// 先查 Redis
	var activity models.Activity
	idStr := strconv.FormatInt(id, 10)
	cacheKey := "activity:detail:" + idStr
	isFromCache := false
	val, err := rdb.Get(c, cacheKey).Result()
	if val == "NULL" {
		return nil, errors.New("活动不存在")
	}
	if err == nil {
		if e := json.Unmarshal([]byte(val), &activity); e == nil {
			isFromCache = true
		}
	}

	// 再查数据库
	if !isFromCache {
		if err := db.First(&activity, id).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				rdb.Set(c, cacheKey, "NULL", 2*time.Minute)
				return nil, errors.New("活动不存在")
			}
			return nil, errors.New("查询失败")
		}

		// 缓存非下架活动
		if activity.GetStatus() != models.RM {
			data, err := json.Marshal(activity)
			if err == nil {
				rdb.Set(c, cacheKey, data, 10*time.Minute)
			}
		}
	}

	stockVal, err := rdb.Get(c, "activity:stock:"+idStr).Result()
	if err == nil {
		if s, e := strconv.Atoi(stockVal); e == nil {
			activity.Stock = s
		}
	}

	return &activity, nil
}
