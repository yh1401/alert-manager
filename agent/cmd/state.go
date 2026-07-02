package cmd

import (
	"alert-manager-agent/config"
	"fmt"
	"log"
	"os"

	"gopkg.in/yaml.v3"
)

// Agent 状态结构体
type AgentState struct {
	NodeID        string `yaml:"node_id"`
	RulesHash     string `yaml:"rules_hash"`
	Port          int    `yaml:"port,omitempty"`
	ReloadBackend string `yaml:"reload_backend,omitempty"`
	ReloadURL     string `yaml:"reload_url,omitempty"`
}

// 加载状态文件
func loadStateFile() AgentState {
	var state AgentState
	data, err := os.ReadFile(config.GlobalConfig.StateFilePath)
	if err != nil {
		// 文件不存在或无法读取，返回空状态
		return state
	}
	if err := yaml.Unmarshal(data, &state); err != nil {
		log.Printf("⚠️ 无法解析状态文件: %v", err)
		return state
	}
	return state
}

// 保存状态文件
func saveStateFile(state AgentState) error {
	data, err := yaml.Marshal(state)
	if err != nil {
		return fmt.Errorf("无法序列化状态: %v", err)
	}
	return os.WriteFile(config.GlobalConfig.StateFilePath, data, 0644)
}

// 读取本地缓存的规则哈希（如果存在），并尝试恢复端口与重载配置
func LoadCachedHash() {
	state := loadStateFile()
	if state.RulesHash != "" {
		config.GlobalConfig.LastRulesHash = state.RulesHash
		log.Printf("ℹ️ 已加载本地规则哈希: %s", state.RulesHash)
	}
	// 如果状态文件包含端口信息，优先使用（仅当命令行未显式指定 port 时）
	if state.Port != 0 && config.GlobalConfig.Port == 0 {
		config.GlobalConfig.Port = state.Port
		log.Printf("ℹ️ 已从状态文件加载端口: %d", state.Port)
	}
	// 如果状态文件包含重载配置，且命令行未显式指定（ReloadURL 为空），则恢复
	if state.ReloadURL != "" && config.GlobalConfig.ReloadURL == "" {
		config.GlobalConfig.ReloadURL = state.ReloadURL
		config.GlobalConfig.ReloadBackend = state.ReloadBackend
		log.Printf("ℹ️ 已从状态文件加载重载后端: %s, URL: %s", state.ReloadBackend, state.ReloadURL)
	}
}

func saveHashToFile(hash string) {
	if hash == "" {
		return
	}
	state := loadStateFile()
	state.RulesHash = hash
	if err := saveStateFile(state); err != nil {
		log.Printf("⚠️ 无法写入状态文件: %v", err)
	}
}

// 将端口写入状态文件（如果 port == 0 则不写入）
// 导出为 SavePort 以便外部包调用
func SavePort(port int) {
	// port == 0 表示未分配，为了兼容历史文件不写入 0
	if port == 0 {
		return
	}
	state := loadStateFile()
	state.Port = port
	if err := saveStateFile(state); err != nil {
		log.Printf("⚠️ 无法写入状态文件（保存端口）: %v", err)
	}
}

// SaveReload 将重载后端和 URL 写入状态文件（非空时写入）
// backend 格式示例: "prometheus" | "vmalert" | "custom"
func SaveReload(backend string, url string) {
	// 如果两项都为空则不写入
	if backend == "" && url == "" {
		return
	}
	state := loadStateFile()
	state.ReloadBackend = backend
	state.ReloadURL = url
	if err := saveStateFile(state); err != nil {
		log.Printf("⚠️ 无法写入状态文件（保存重载信息）: %v", err)
	}
}
