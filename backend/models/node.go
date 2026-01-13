package models

import "time"

// Node 对应数据库中的 nodes 表
// 用于后端记录所有注册的 Agent 信息
type Node struct {
	ID            int       `gorm:"primaryKey"`
	Name          string    `gorm:"column:name"`
	IPAddress     string    `gorm:"column:ip_address"`
	SecretKey     string    `gorm:"column:secret_key"`     // 通信密钥，用于 Agent 认证
	LastHeartbeat time.Time `gorm:"column:last_heartbeat"` // 最后心跳时间
	CreatedAt     time.Time `gorm:"column:created_at"`
	UpdatedAt     time.Time `gorm:"column:updated_at"`
}

func (Node) TableName() string {
	return "nodes"
}
