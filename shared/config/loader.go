// Package config 提供统一配置加载。
//
// 设计要点：
//   - 每个服务一个 yaml：config/<service>.yaml，相对工作目录。
//   - 环境变量覆盖 yaml：使用 DOCTORS_<SERVICE>_<FIELD>_<SUB> 形式（自动映射）。
//   - 默认值在 Load 时填入；调用方可零成本使用。
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config 是全部服务的统一配置根。
type Config struct {
	Service string    `mapstructure:"service"`
	HTTP    HTTP      `mapstructure:"http"`
	DB      DB        `mapstructure:"db"`
	Redis   Redis     `mapstructure:"redis"`
	Kafka   Kafka     `mapstructure:"kafka"`
	Auth    Auth      `mapstructure:"auth"`
	Logging Logging   `mapstructure:"logging"`
	Admin   Admin     `mapstructure:"admin"`
}

// HTTP 是 HTTP 服务配置。
type HTTP struct {
	Addr            string `mapstructure:"addr"`
	ReadTimeoutSec  int    `mapstructure:"read_timeout_sec"`
	WriteTimeoutSec int    `mapstructure:"write_timeout_sec"`
}

// DB 是 PostgreSQL 连接配置。
type DB struct {
	DSN      string `mapstructure:"dsn"`
	MaxConns int32  `mapstructure:"max_conns"`
	MinConns int32  `mapstructure:"min_conns"`
}

// Redis 是缓存配置。
type Redis struct {
	Addr     string `mapstructure:"addr"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

// Kafka 是消息队列配置。
type Kafka struct {
	Brokers []string `mapstructure:"brokers"`
	GroupID string   `mapstructure:"group_id"`
}

// Auth 是鉴权配置。
type Auth struct {
	JWTSecret string        `mapstructure:"jwt_secret"`
	JWTTTL    time.Duration `mapstructure:"jwt_ttl"`
}

// Logging 是日志配置。
type Logging struct {
	Level string `mapstructure:"level"` // debug | info | warn | error
}

// Admin 是 admin-service 专用配置：内部服务 baseURL。
//
// v1 admin plan (2026-09-24)：admin 通过这些 URL 调 order/refund/escort/user。
type Admin struct {
	OrderBaseURL  string `mapstructure:"order_base_url"`
	RefundBaseURL string `mapstructure:"refund_base_url"`
	EscortBaseURL string `mapstructure:"escort_base_url"`
	UserBaseURL   string `mapstructure:"user_base_url"`
}

// Load 读取 config/<service>.yaml + 环境变量，解析为 *Config。
//
// 环境变量覆盖规则：DOCTORS_<UPPER_SNAKE>，例如 DOCTORS_AUTH_JWT_SECRET。
// viper.AutomaticEnv 自动按 key 路径匹配。
func Load(service string) (*Config, error) {
	v := viper.New()
	v.SetConfigName(service)
	v.SetConfigType("yaml")
	v.AddConfigPath("./config")
	v.AddConfigPath(".")

	// 默认值
	v.SetDefault("http.addr", ":8080")
	v.SetDefault("http.read_timeout_sec", 30)
	v.SetDefault("http.write_timeout_sec", 30)
	v.SetDefault("db.max_conns", 10)
	v.SetDefault("db.min_conns", 2)
	v.SetDefault("redis.db", 0)
	v.SetDefault("logging.level", "info")

	// 环境变量绑定
	v.SetEnvPrefix("DOCTORS")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("load config/%s.yaml: %w", service, err)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}
	cfg.Service = service
	return &cfg, nil
}

// MustLoad 是 Load 的 panic 版本，常用于 main 入口。
func MustLoad(service string) *Config {
	cfg, err := Load(service)
	if err != nil {
		panic(err)
	}
	return cfg
}

// loadPathForTest 暴露给测试定位 yaml 路径；正常代码不要使用。
func loadPathForTest(service string) string {
	return filepath.Join("config", service+".yaml")
}

// init 把配置目录加到 viper 默认搜索路径中。
func init() {
	// 允许通过环境变量切换 base path，便于测试
	if p := os.Getenv("DOCTORS_CONFIG_DIR"); p != "" {
		viper.AddConfigPath(p)
	}
}