```
ticket_project
├── main.go             # 项目启动入口
├── config/             # 配置文件夹（存放数据库密码、Redis地址等）
│   └── config.yaml
│   └── config.go
│   └── order_loc.lua   # lua 脚本
├── controller/         # 接口层（处理接口请求）
│   ├── user_role.go    # 用户接口
│   ├── activity.go     # 活动接口
│   └── order.go        # 订单接口
│   └── ticket.go       # 票接口
├── logic/              # 业务层（抢票、并发、校验）
│   ├── user_logic.go
│   ├── activity_logic.go
│   ├── order_logic.go
│   └── ticket_logic.go
│   └── stream.go       # 消息队列 redis stream
├── models/             # 定义数据模型
│   ├── user.go         # 用户模型
│   └── role.go         # 角色模型
│   └── activity.go     # 活动模型
│   └── order.go        # 订单模型
│   └── ticket.go       # 票模型
├── dao/                # 数据库操作（GORM）
│   ├── init_mysql.go   # 初始化连接
│   └── init_redis.go   # 初始化 Redis 连接
├── router/             # 路由定义（定义 URL 到 Controller 的映射）
│   └── router.go
├── utils/              # 工具类
│   ├── midddle.go          # 中间件
│   └── response/
│       └── response.go     # 统一封装返回给前端的 JSON 格式
└── web/                # 前端（Vue3 项目）
    ├── 
    └── 
```
