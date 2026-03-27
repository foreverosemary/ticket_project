package config

import (
	"fmt"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/go-sql-driver/mysql"
	"github.com/spf13/viper"
)

type Config struct {
	App   AppConfig   `mapstructure:"app"`
	MySQL MySQLConfig `mapstructure:"mysql"`
	Redis RedisConfig `mapstructure:"redis"`
}

type AppConfig struct {
	Mode     string `mapstructure:"mode"`
	LogLevel string `mapstructure:"logLevel"`
}

type MySQLConfig struct {
	Host            string `mapstructure:"host"`
	Port            int    `mapstructure:"port"`
	Username        string `mapstructure:"username"`
	Password        string `mapstructure:"password"`
	Database        string `mapstructure:"database"`
	Charset         string `mapstructure:"charset"`
	ParseTime       bool   `mapstructure:"parseTime"`
	Loc             string `mapstructure:"loc"`
	MaxOpenConns    int    `mapstructure:"maxOpenConns"`
	MaxIdleConns    int    `mapstructure:"maxIdleConns"`
	ConnMaxLifetime int    `mapstructure:"connMaxLifetime"`
}

type RedisConfig struct {
	Addr         string `mapstructure:"addr"`
	Password     string `mapstructure:"password"`
	DB           int    `mapstructure:"db"`
	PoolSize     int    `mapstructure:"poolSize"`
	MinIdleConns int    `mapstructure:"minIdleConns"`
}

func (m *MySQLConfig) DSN() string {
	cfg := mysql.NewConfig()
	cfg.User = m.Username
	cfg.Passwd = m.Password
	cfg.Net = "tcp"
	cfg.Addr = fmt.Sprintf("%s:%d", m.Host, m.Port)
	cfg.DBName = m.Database
	cfg.ParseTime = m.ParseTime
	cfg.Collation = "utf8mb4_general_ci"

	cfg.InterpolateParams = true

	// 处理时区
	if m.Loc == "" {
		cfg.Loc = time.Local
	} else {
		cfg.Loc, _ = time.LoadLocation(m.Loc)
	}

	return cfg.FormatDSN()
}

func (r *RedisConfig) URL() string {
	return fmt.Sprintf("redis://:%s@%s/%d", r.Password, r.Addr, r.DB)
}

var (
	cfg                   *Config
	configChangeCallbacks []func() // 配置变化回调函数列表
	mu                    sync.RWMutex
)

func AddConfigChangeCallback(fn func()) {
	mu.Lock()
	defer mu.Unlock()
	configChangeCallbacks = append(configChangeCallbacks, fn)
}

// 加载配置文件并解析到结构体
func Load() error {
	// 设置配置文件信息
	v := viper.New()
	v.SetConfigFile("./config/config.yaml")

	// 开启热监听
	v.WatchConfig()

	// 配置变化时自动重新加载
	v.OnConfigChange(func(e fsnotify.Event) {
		fmt.Println("配置文件已变更:", e.Name)

		mu.Lock()
		err := v.Unmarshal(cfg)
		mu.Unlock()

		if err != nil {
			fmt.Printf("重新加载失败: %v\n", err)
			return
		}

		// 锁内拷贝锁外执行
		mu.Lock()
		cp := make([]func(), len(configChangeCallbacks))
		copy(cp, configChangeCallbacks)
		mu.Unlock()

		for _, fn := range cp {
			fn()
		}

		fmt.Println("配置热更新成功")
	})

	// 读取配置文件
	if err := v.ReadInConfig(); err != nil {
		return fmt.Errorf("read config file error: %w", err)
	}

	// 解析到对应结构体
	mu.Lock()
	cfg = &Config{}
	if err := v.Unmarshal(cfg); err != nil {
		return fmt.Errorf("unmarshal config error: %w", err)
	}
	mu.Unlock()

	return nil
}

// 获取当前配置实例
func GetConfig() *Config {
	mu.RLock()
	defer mu.RUnlock()
	if cfg == nil {
		panic("配置未加载，请先调用 Load() 函数")
	}
	return cfg
}
