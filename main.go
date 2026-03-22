package main

import (
	"log"
	"ticket/config"
	"ticket/dao"
	"ticket/router"
)

func main() {

	// 加载配置文件
	if err := config.Load(); err != nil {
		log.Fatalf("配置加载失败：%v", err)
	}

	// 初始化 MySQL
	if err := dao.InitMySQL(); err != nil {
		log.Fatalf("MySQL 错误: %v", err)
	}

	// 初始化 Redis
	if err := dao.InitRedis(); err != nil {
		log.Fatalf("Redis 错误: %v", err)
	}

	// 初始化路由
	router.InitRouter()
	r := router.GetRouter()

	// 启动服务
	log.Println("服务启动成功并运行在 :8080 端口")
	if err := r.Run(":8080"); err != nil {
		log.Fatalf("服务启动失败: %v", err)
	}
}
