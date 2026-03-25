package logic

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"ticket/dao"
	"ticket/models"
	"time"

	"github.com/redis/go-redis/v9"
)

// 队列名
const ticketKey = "ticket_stream"
const activityKey = "activity_stream"

// 消费组名
const groupName = "consume_group"

var (
	rdb = dao.GetRDB()
	ctx = context.Background()
	db  = dao.GetDB()
)

func InitStreamGroup() {
	_ = rdb.XGroupCreate(ctx, ticketKey, groupName, "$").Err()
	_ = rdb.XGroupCreate(ctx, activityKey, groupName, "$").Err()
}

func ProduceTicketMsg(orderId, activityId int64, need int) error {
	args := &redis.XAddArgs{
		Stream: ticketKey,
		ID:     "*",
		Values: map[string]interface{}{
			"order_id":    orderId,
			"activity_id": activityId,
			"need":        need,
		},
	}
	_, err := rdb.XAdd(ctx, args).Result()
	return err
}

func ProduceActivityMsg(activityId int64) error {
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

func StartStreamConsumer() {
	go func() {
		for {
			// 先处理旧消息
			streams, err := rdb.XReadGroup(ctx, &redis.XReadGroupArgs{
				Group:    groupName,
				Consumer: "consumer",
				Streams:  []string{ticketKey, activityKey, "0", "0"},
				Count:    10,
			}).Result()
			if err == nil && len(streams) > 0 {
				handleMessages(streams)
				continue // 重复处理旧消息
			}

			// 再处理新消息
			streams, err = rdb.XReadGroup(ctx, &redis.XReadGroupArgs{
				Group:    groupName,
				Consumer: "consumer",
				Streams:  []string{ticketKey, activityKey, ">", ">"},
				Block:    0,
			}).Result()
			// 阻塞读
			if err == nil && len(streams) > 0 {
				handleMessages(streams)
			}
		}
	}()
}

func handleMessages(streams []redis.XStream) {
	for _, stream := range streams {
		// 消息队列判断
		streamName := stream.Stream
		for _, msg := range stream.Messages {
			msgId := msg.ID
			var err error

			getField := func(key string) string {
				if v, ok := msg.Values[key]; ok {
					return v.(string)
				}
				return ""
			}

			switch streamName {
			case ticketKey:
				orderId, _ := strconv.ParseInt(getField("order_id"), 10, 64)
				activityId, _ := strconv.ParseInt(getField("activity_id"), 10, 64)
				need, _ := strconv.Atoi(getField("need"))
				err = createTicket(orderId, activityId, need)
			case activityKey:
				activityId := msg.Values["activity_id"].(int64)
				err = delActivityInfo(activityId)
			}
			if err != nil {
				log.Printf("消息处理失败 [ID: %s]:%v", msgId, err)
				return
			}
			rdb.XAck(ctx, streamName, groupName, msgId)
		}
	}
}

func createTicket(orderId, activityId int64, need int) error {
	tickets := make([]models.Ticket, need)
	now := time.Now().UnixMilli()
	for i := 0; i < need; i++ {
		tickets[i].OrderID = orderId
		tickets[i].ActivityID = activityId
		tickets[i].TicketNo = fmt.Sprintf("%d%d", now, i)
		tickets[i].Status = models.IV
	}
	return db.Create(&tickets).Error
}

func delActivityInfo(activityId int64) error {
	batchSize := 500
	// 分批作废订单
	for {
		delTime := time.Now()
		result := db.Exec(`
		UPDATE orders
			SET status = ?, deleted_at = ?
			WHERE activity_id = ? AND status != ? AND deleted_at IS NULL
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
