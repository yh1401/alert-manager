package cmd

import (
	"alert-manager-agent/config"
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"
)

func StartHeartbeat() {
	ticker := time.NewTicker(config.GlobalConfig.HeartbeatInt)
	defer ticker.Stop()

	for range ticker.C {
		currentIP := GetLocalIP()
		url := fmt.Sprintf("%s/api/agent/heartbeat?node_id=%s&ip_address=%s",
			config.GlobalConfig.BackendURL,
			config.GlobalConfig.NodeID,
			currentIP,
		)
		resp, err := http.Post(url, "application/json", nil)
		if err != nil {
			log.Printf("⚠️ 心跳上报失败: %v", err)
			continue
		}
		resp.Body.Close()
	}
}

// ReportSyncStatus 向后端上报拉取与重载结果
func ReportSyncStatus(configHash, fetchStatus, reloadStatus, errMsg string) {
	url := fmt.Sprintf("%s/api/agent/report_sync", config.GlobalConfig.BackendURL)

	nodeID, _ := strconv.Atoi(config.GlobalConfig.NodeID)
	payload := map[string]interface{}{
		"node_id":       nodeID,
		"config_hash":   configHash,
		"fetch_status":  fetchStatus,
		"reload_status": reloadStatus,
		"error_msg":     errMsg,
	}
	body, _ := json.Marshal(payload)

	client := &http.Client{Timeout: 5 * time.Second}
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("⚠️ 上报同步状态失败: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("⚠️ 上报同步状态返回非 200: %s", resp.Status)
	}
}
