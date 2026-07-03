package initialize

import (
	"alert-manager-backend/global"
	"alert-manager-backend/models"
	"alert-manager-backend/prometheus"
	"log"
	"os"

	"github.com/goccy/go-yaml"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Config 全局配置（从 config.yaml 读取后保留，供其他模块使用）
var Config models.DSNConfig

func InitDB() {
	// 读取 config.yaml
	data, err := os.ReadFile("./config/config.yaml")
	if err != nil {
		log.Fatal("读取 config.yaml 失败: ", err)
	}
	var conf models.DSNConfig
	if err := yaml.Unmarshal(data, &conf); err != nil {
		log.Fatal("解析 config.yaml 失败: ", err)
	}
	Config = conf

	dsn := conf.DSN
	// 环境变量优先级最高 (覆盖配置文件，方便 Docker 部署)
	if envDSN := os.Getenv("DB_DSN"); envDSN != "" {
		dsn = envDSN
		log.Println("使用环境变量 DB_DSN 覆盖配置")
	}

	// 初始化全局 DB
	global.DB, err = gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Error), // 只记录错误，不记录 "record not found" 警告
	})
	if err != nil {
		log.Fatal("连接数据库失败: ", err)
	}

	// 自动迁移表结构
	if err := global.DB.AutoMigrate(
		&models.Node{},
		&models.RuleGroup{},
		&models.RuleGroupVersion{},
		&models.User{},
		&models.UserRole{},
		&models.Permission{},
		&models.NodeSyncStatus{},
		&models.NodeSyncHistory{},
		&models.AuditLog{}, // 审计日志表
		&models.Tag{},      // 标签表
	); err != nil {
		log.Fatal("数据库迁移失败: ", err)
	}
	log.Println("GORM 连接数据库成功")

	// 初始化 Prometheus 客户端
	promConfig := conf.Prometheus
	// 环境变量覆盖 Prometheus URL
	if envURL := os.Getenv("PROMETHEUS_URL"); envURL != "" {
		promConfig.URL = envURL
		log.Println("使用环境变量 PROMETHEUS_URL 覆盖配置")
	}
	if promConfig.URL == "" {
		promConfig.URL = "http://localhost:9090"
	}
	global.PrometheusClient = prometheus.NewClient(promConfig)
	log.Printf("Prometheus 客户端已初始化: %s", promConfig.URL)
}
