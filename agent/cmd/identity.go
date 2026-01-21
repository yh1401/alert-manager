package cmd

import (
	"alert-manager-agent/config"
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
)

// GetOrRegisterIdentity 获取或注册身份
func GetOrRegisterIdentity() error {
	// 1. 尝试读取本地状态文件
	state := loadStateFile()
	if state.NodeID != "" {
		config.GlobalConfig.NodeID = state.NodeID
		log.Printf("✅ 身份已加载，NodeID: %s", config.GlobalConfig.NodeID)
		return nil
	}

	// 2. 本地无身份，向后端注册
	log.Println("⚠️ 未找到本地身份，正在向后端注册...")
	hostname, _ := os.Hostname()
	name := config.GlobalConfig.Name
	if name == "" {
		name = hostname
	}
	ip := GetLocalIP()

	// 收集本地规则文件
	files, err := CollectLocalRules()
	if err != nil {
		log.Printf("⚠️ 收集本地规则失败: %v", err)
	}

	reqBody := map[string]interface{}{
		"hostname":   name,
		"ip_address": ip,
		"files":      files,
	}
	jsonBody, _ := json.Marshal(reqBody)

	resp, err := http.Post(config.GlobalConfig.BackendURL+"/api/agent/register", "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		return fmt.Errorf("注册请求失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("注册失败，状态码: %d", resp.StatusCode)
	}

	var result struct {
		NodeID int `json:"node_id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("解析注册响应失败: %v", err)
	}

	// 3. 保存身份到状态文件
	idStr := strconv.Itoa(result.NodeID)
	state.NodeID = idStr
	if err := saveStateFile(state); err != nil {
		log.Printf("⚠️ 警告: 无法写入状态文件: %v", err)
	}

	config.GlobalConfig.NodeID = idStr
	log.Printf("✅ 注册成功! 分配 NodeID: %s", idStr)
	return nil
}

func GetLocalIP() string {
	addrs, _ := net.InterfaceAddrs()
	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() && ipnet.IP.To4() != nil {
			return ipnet.IP.String()
		}
	}
	return ""
}
