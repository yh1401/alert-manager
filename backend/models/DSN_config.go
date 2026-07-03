package models

import "time"

// DSNConfig 用于解析 config.yaml
type DSNConfig struct {
	DSN         string             `yaml:"dsn"`
	Prometheus  PrometheusConfig   `yaml:"prometheus"`
}

// PrometheusConfig Prometheus 数据源连接配置
type PrometheusConfig struct {
	URL           string        `yaml:"url"`            // Prometheus HTTP API 地址
	Timeout       time.Duration `yaml:"timeout"`        // HTTP 请求超时
	RetryCount    int           `yaml:"retry_count"`     // 失败重试次数
	RetryInterval time.Duration `yaml:"retry_interval"`  // 重试基础间隔
	CacheTTL      time.Duration `yaml:"cache_ttl"`       // 缓存有效期
}
