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
	NodeID    string `yaml:"node_id"`
	RulesHash string `yaml:"rules_hash"`
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

// 读取本地缓存的规则哈希（如果存在）
func LoadCachedHash() {
	state := loadStateFile()
	if state.RulesHash != "" {
		config.GlobalConfig.LastRulesHash = state.RulesHash
		log.Printf("ℹ️ 已加载本地规则哈希: %s", state.RulesHash)
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
