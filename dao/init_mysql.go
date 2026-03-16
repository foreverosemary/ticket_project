package dao

import (
	"fmt"
	"sync"
	"ticket/config"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var (
	db       *gorm.DB
	mu_mysql sync.RWMutex
)

func InitMySQL() error {
	// 加载配置并首次连接
	cfg := config.GetConfig()
	if err := connectMySQL(cfg); err != nil {
		return err
	}

	config.AddConfigChangeCallback(func() {
		fmt.Println("检测到 MySQL 配置变更，开始重连...")
		newCfg := config.GetConfig()

		if err := connectMySQL(newCfg); err != nil {
			fmt.Printf("MySQL 重连失败: %v\n", err)
			return
		}

		fmt.Printf("MySQL 重连成功")
	})

	return nil
}

func connectMySQL(cfg *config.Config) error {
	dsn := cfg.MySQL.DSN()
	newDB, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		fmt.Printf("MySQL 连接失败: %v", err)
		return err
	}

	mu_mysql.Lock()
	oldDB := db
	db = newDB
	mu_mysql.Unlock()

	if oldDB != nil {
		sqlDB, err := oldDB.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	}

	fmt.Printf("MySQL 连接成功")

	return nil
}

func GetDB() *gorm.DB {
	mu_mysql.RLock()
	defer mu_mysql.RUnlock()
	return db
}
