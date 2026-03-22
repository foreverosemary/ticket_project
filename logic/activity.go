package logic

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"ticket/dao"
	"ticket/models"
	"time"

	"gorm.io/gorm"
)

type ActivityLogic struct{}

func (actLogic *ActivityLogic) CreateActivity(c context.Context, input models.Activity) (*models.Activity, error) {
	db := dao.GetDB()
	rdb := dao.GetRDB()

	// 检验
	if err := input.Verify(); err != nil {
		return nil, err
	}

	input.SetStatus()
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

func (actLogic *ActivityLogic) UpdateAllActivity(c context.Context, id int64, input models.Activity) (*models.Activity, error) {
	db := dao.GetDB().Unscoped()
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
		if activity.Status != models.RM {
			input.SetStatus()
		} else {
			input.Status = models.RM
		}

		if err := tx.Model(&activity).Updates(map[string]interface{}{
			"name":       input.Name,
			"content":    input.Content,
			"stock":      gorm.Expr("stock + ?", diff),
			"total":      input.Total,
			"start_time": input.StartTime.ToTime(),
			"end_time":   input.EndTime.ToTime(),
			"status":     input.Status,
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

func (actLogic *ActivityLogic) UpdatePartialActivity(c context.Context, id int64, dto models.UpdateActivityDTO) (*models.Activity, error) {
	db := dao.GetDB().Unscoped()
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
		if activity.Status != models.RM {
			updates["status"] = activity.Status
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

func (actLogic *ActivityLogic) asyncCleanup(id int64, delTime gorm.DeletedAt) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	db := dao.GetDB().WithContext(ctx)
	batchSize := 500

	// 分批作废订单
	for {
		result := db.Exec(`
			UPDATE orders
			SET status = ?, deleted_at = ?, updated_at = ?
			WHERE id = ? AND status != ? AND deleted_at IS NULL
			LIMIT ?`,
			models.CL, delTime.Time, time.Now(), id, models.CL, batchSize)

		if result.Error != nil {
			fmt.Println(result.Error)
			return
		}
		if result.RowsAffected == 0 {
			break
		}

		// 停顿
		time.Sleep(20 * time.Millisecond)
	}

	// 分批作废门票
	// 2. 分批作废门票
	for {
		result := db.Exec(`
			UPDATE tickets 
			SET status = ?, deleted_at = ?, updated_at = ? 
			WHERE activity_id = ? AND status != ? AND deleted_at IS NULL 
			LIMIT ?`,
			models.IV, delTime.Time, time.Now(), id, models.IV, batchSize)

		if result.Error != nil {
			fmt.Println(result.Error)
			return
		}
		if result.RowsAffected == 0 {
			break
		}

		// 停顿
		time.Sleep(20 * time.Millisecond)
	}
}

func (actLogic *ActivityLogic) DeleteActivity(c context.Context, id int64) (*models.Activity, error) {
	db := dao.GetDB().WithContext(c).Unscoped()
	rdb := dao.GetRDB()

	// 检验是否存在
	var activity models.Activity
	if err := db.First(&activity, id).Error; err != nil {
		return nil, err
	}

	if activity.Status == models.RM {
		return nil, errors.New("活动重复删除")
	}

	deletedTime := gorm.DeletedAt{Time: time.Now(), Valid: true}

	// 删除活动事务
	err := db.Transaction(func(tx *gorm.DB) error {
		return tx.Model(&activity).Updates(map[string]interface{}{
			"status":     models.RM,
			"deleted_at": deletedTime,
		}).Error
	})

	if err != nil {
		return nil, errors.New("活动状态更新失败")
	}

	// 删除缓存
	idStr := strconv.FormatInt(id, 10)
	rdb.Del(c, "activity:detail:"+idStr)
	rdb.Del(c, "activity:stock:"+idStr)

	// 异步
	go actLogic.asyncCleanup(id, deletedTime)

	// 获取最新结果
	if err := db.Unscoped().First(&activity, id).Error; err != nil {
		return nil, errors.New("删除活动成功但是返回数据失败" + err.Error())
	}

	return &activity, nil
}

func (actLogic *ActivityLogic) GetActivities(c context.Context, q models.ActivityQuery) (*models.ActivityList, error) {
	db := dao.GetDB().Unscoped()

	// 构建查询
	queryDB := db.Model(&models.Activity{})

	if q.ActivityID > 0 {
		queryDB = queryDB.Where("id = ?", q.ActivityID)
	}

	if q.Name != "" {
		queryDB = queryDB.Where("name LIKE ?", "%"+q.Name+"%")
	}

	queryDB = queryDB.Where("status IN (?)", q.StatusList)

	// 查询
	var activityList models.ActivityList
	if err := queryDB.Session(&gorm.Session{NewDB: true}).
		Count(&activityList.Total).Error; err != nil {
		return nil, errors.New("查询失败:" + err.Error())
	}

	if err := queryDB.
		Limit(q.PageSize).
		Offset((q.PageNum - 1) * q.PageSize).
		Order("status ASC, start_time ASC").
		Find(&activityList.Activities).Error; err != nil {
		return nil, errors.New("查询活动列表失败:" + err.Error())
	}

	return &activityList, nil
}

func (actLogic *ActivityLogic) GetActivityDetail(c context.Context, id int64) (*models.Activity, error) {
	db := dao.GetDB().Unscoped()
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
		if activity.Status != models.RM {
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
