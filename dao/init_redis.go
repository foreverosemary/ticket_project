package dao

import (
	"context"
	"fmt"
	"sync"
	"ticket/config"

	"github.com/redis/go-redis/v9"
)

var (
	rdb      *redis.Client
	ctx      = context.Background()
	mu_redis sync.RWMutex
)

func InitRedis() error {
	// 加载配置并连接
	cfg := config.GetConfig()
	if err := connectRedis(cfg); err != nil {
		return err
	}

	// 注册配置热更新
	config.AddConfigChangeCallback(func() {
		fmt.Println("检测到 Redis 配置变更，开始重连...")
		newCfg := config.GetConfig()

		// 重新连接
		if err := connectRedis(newCfg); err != nil {
			fmt.Printf("Redis 重连失败: %v\n", err)
			return
		}
	})

	fmt.Printf("Redis连接成功: %v\n", rdb)
	return nil
}

func connectRedis(cfg *config.Config) error {
	// 解析 url 并创建客户端
	opt, err := redis.ParseURL(cfg.Redis.URL())
	if err != nil {
		return err
	}
	newRdb := redis.NewClient(opt)

	// 测试连接
	if err := newRdb.Ping(ctx).Err(); err != nil {
		newRdb.Close()
		return err
	}

	// 替换客户端
	mu_redis.Lock()
	oldRdb := rdb
	rdb = newRdb
	mu_redis.Unlock()

	if oldRdb != nil {
		_ = oldRdb.Close()
	}

	return nil
}

func GetRDB() *redis.Client {
	mu_redis.RLock()
	defer mu_redis.RUnlock()
	return rdb
}
