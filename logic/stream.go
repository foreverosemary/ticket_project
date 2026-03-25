package logic

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"ticket/dao"
	"ticket/models"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// 队列名
const ticketKey = "ticket_stream"
const activityKey = "activity_stream"

// 消费组名
const groupName = "consume_group"

func InitStreamGroup(rdb *redis.Client) {
	ctx := context.Background()

	rdb.XAdd(ctx, &redis.XAddArgs{
		Stream: ticketKey,
		Values: map[string]interface{}{"init": "create"},
	})
	rdb.XAdd(ctx, &redis.XAddArgs{
		Stream: activityKey,
		Values: map[string]interface{}{"init": "create"},
	})
	if err := rdb.XGroupCreate(ctx, ticketKey, groupName, "$").Err(); err != nil {
		log.Printf("初始化消息队列失败:%v", err.Error())
	}
	if err := rdb.XGroupCreate(ctx, activityKey, groupName, "$").Err(); err != nil {
		log.Printf("初始化消息队列失败:%v", err.Error())
	}
}

func ProduceTicketMsg(rdb *redis.Client, orderId, activityId int64, need int) error {
	ctx := context.Background()
	log.Printf("[生产者] 准备发送消息: orderId=%d, activityId=%d, need=%d", orderId, activityId, need)
	args := &redis.XAddArgs{
		Stream: ticketKey,
		ID:     "*",
		Values: map[string]interface{}{
			"order_id":    orderId,
			"activity_id": activityId,
			"need":        need,
		},
	}
	id, err := rdb.XAdd(ctx, args).Result()
	if err != nil {
		log.Printf("[生产者] 消息发送失败: %v", err)
		return err
	}
	log.Printf("[生产者] 消息发送成功, StreamID: %s", id)
	return err
}

func ProduceActivityMsg(rdb *redis.Client, activityId int64) error {
	ctx := context.Background()
	args := &redis.XAddArgs{
		Stream: activityKey,
		ID:     "*",
		Values: map[string]interface{}{
			"activity_id": activityId,
		},
	}
	_, err := rdb.XAdd(ctx, args).Result()
	return err
}

func StartStreamConsumer(rdb *redis.Client) {
	ctx := context.Background()
	log.Println("[消费者] 启动消费者协程...")
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("[消费者] 协程崩溃重启中: %v", r)
				time.Sleep(time.Second)
				StartStreamConsumer(rdb)
			}
		}()

		for {
			// 先处理旧消息
			for {
				streams, err := rdb.XReadGroup(ctx, &redis.XReadGroupArgs{
					Group:    groupName,
					Consumer: "consumer",
					Streams:  []string{ticketKey, activityKey, "0", "0"},
					Count:    100,
				}).Result()
				if err != nil || len(streams) == 0 || countMessages(streams) == 0 {
					break
				}

				handleMessages(rdb, streams)
				log.Printf("处理完毕 pending 旧消息共: %v 条", countMessages(streams))
			}

			// 再处理新消息
			streams, err := rdb.XReadGroup(ctx, &redis.XReadGroupArgs{
				Group:    groupName,
				Consumer: "consumer",
				Streams:  []string{ticketKey, activityKey, ">", ">"},
				Count:    10,
				Block:    0,
			}).Result()
			// 阻塞读
			if err != nil {
				if !strings.Contains(err.Error(), "context canceled") {
					log.Printf("[消费者] XReadGroup 错误: %v", err)
				}
				time.Sleep(100 * time.Millisecond)
				continue
			}
			if len(streams) > 0 && countMessages(streams) > 0 {
				log.Printf("处理新消息")
				handleMessages(rdb, streams)
				time.Sleep(100 * time.Millisecond)
			}
		}
	}()
}

func countMessages(streams []redis.XStream) int {
	total := 0
	for _, s := range streams {
		total += len(s.Messages)
	}
	return total
}

func handleMessages(rdb *redis.Client, streams []redis.XStream) {
	ctx := context.Background()
	for _, stream := range streams {
		// 消息队列判断
		streamName := stream.Stream
		for _, msg := range stream.Messages {
			msgId := msg.ID
			var err error

			switch streamName {
			case ticketKey:
				orderIdStr, _ := msg.Values["order_id"].(string)
				activityIdStr, _ := msg.Values["activity_id"].(string)
				needStr, _ := msg.Values["need"].(string)

				orderId, _ := strconv.ParseInt(orderIdStr, 10, 64)
				activityId, _ := strconv.ParseInt(activityIdStr, 10, 64)
				need, _ := strconv.Atoi(needStr)
				err = createTicket(orderId, activityId, need)
			case activityKey:
				activityIdStr, _ := msg.Values["activity_id"].(string)
				activityId, _ := strconv.ParseInt(activityIdStr, 10, 64)
				err = delActivityInfo(activityId)
			}
			if err != nil {
				log.Printf("消息处理失败 [ID: %s]:%v", msgId, err)
			}
			rdb.XAck(ctx, streamName, groupName, msgId)
		}
	}
}

func createTicket(orderId, activityId int64, need int) error {
	if need <= 0 {
		log.Printf("警告: need=%d <= 0，跳过创建门票 (orderId=%d, activityId=%d)", need, orderId, activityId)
		return nil
	}
	db := dao.GetDB()
	tickets := make([]models.Ticket, need)
	now := time.Now().UnixMilli()
	for i := 0; i < need; i++ {
		tickets[i].OrderID = orderId
		tickets[i].ActivityID = activityId
		tickets[i].TicketNo = fmt.Sprintf("%d%d", now, i)
		tickets[i].Status = models.IV
	}

	// 创建事务
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&tickets).Error; err != nil {
			return err
		}

		result := tx.Model(&models.Activity{}).
			Where("id = ?", activityId).
			Update("stock", gorm.Expr("stock - ?", need))

		if result.Error != nil {
			return result.Error
		}

		if result.RowsAffected == 0 {
			return errors.New("活动不存在，更新库存失败")
		}

		return nil
	})
}

func delActivityInfo(activityId int64) error {
	db := dao.GetDB()
	batchSize := 500
	// 分批作废订单
	for {
		delTime := time.Now()
		result := db.Exec(`
			UPDATE orders o
			SET status = ?, deleted_at = ?
			WHERE o.id IN (
    			SELECT t.order_id 
    			FROM tickets t 
    			WHERE t.activity_id = ?
      			AND o.status != ? 
      			AND o.deleted_at IS NULL
			)
			LIMIT ?`,
			models.CL, delTime, activityId, models.CL, batchSize)

		if result.Error != nil {
			log.Printf("分批作废订单错误:%v", result.Error)
			return result.Error
		}
		if result.RowsAffected == 0 {
			break
		}

		// 停顿
		time.Sleep(50 * time.Millisecond)
	}

	// 分批作废门票
	for {
		delTime := time.Now()
		result := db.Exec(`
			UPDATE tickets 
			SET status = ?, deleted_at = ?
			WHERE activity_id = ? AND status != ? AND deleted_at IS NULL 
			LIMIT ?`,
			models.IV, delTime, activityId, models.IV, batchSize)

		if result.Error != nil {
			log.Printf("分批作废门票错误:%v", result.Error)
			return result.Error
		}
		if result.RowsAffected == 0 {
			break
		}

		// 停顿
		time.Sleep(50 * time.Millisecond)
	}

	return nil
}
