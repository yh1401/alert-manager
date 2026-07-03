package global

import (
	"alert-manager-backend/prometheus"

	"gorm.io/gorm"
)

var DB *gorm.DB

// PrometheusClient 全局 Prometheus API 客户端实例
var PrometheusClient *prometheus.Client
