package cmd

import (
	"alert-manager-agent/config"
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type filePayload struct {
	FilePath string `json:"file_path"`
	Content  string `json:"content"`
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

func FetchConfig() {
	url := fmt.Sprintf("%s/api/agent/config_export?node_id=%s", config.GlobalConfig.BackendURL, config.GlobalConfig.NodeID)

	fetchStatus := "failed"
	reloadStatus := "skipped"
	errMsg := ""
	reportHash := config.GlobalConfig.LastRulesHash

	client := &http.Client{Timeout: 15 * time.Second}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Printf("❌ 创建请求失败: %v", err)
		errMsg = err.Error()
		ReportSyncStatus(reportHash, fetchStatus, reloadStatus, errMsg)
		return
	}

	if config.GlobalConfig.LastRulesHash != "" {
		ifNoneMatch := fmt.Sprintf("W/\"%s\"", config.GlobalConfig.LastRulesHash)
		req.Header.Set("If-None-Match", ifNoneMatch)
	}

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("❌ 拉取配置失败: %v", err)
		errMsg = err.Error()
		ReportSyncStatus(reportHash, fetchStatus, reloadStatus, errMsg)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotModified {
		log.Println("ℹ️ 配置未变化（304），跳过写入与重载")
		fetchStatus = "not_modified"
		reloadStatus = "skipped"
		ReportSyncStatus(reportHash, fetchStatus, reloadStatus, errMsg)
		return
	}

	if resp.StatusCode != http.StatusOK {
		errMsg = fmt.Sprintf("服务器返回错误: %s", resp.Status)
		log.Printf("❌ %s", errMsg)
		ReportSyncStatus(reportHash, fetchStatus, reloadStatus, errMsg)
		return
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("❌ 读取响应失败: %v", err)
		errMsg = err.Error()
		ReportSyncStatus(reportHash, fetchStatus, reloadStatus, errMsg)
		return
	}

	newHash := parseHashFromHeader(resp.Header.Get("ETag"), resp.Header.Get("X-Config-Hash"))
	if newHash == "" {
		newHash = computeHash(body)
	}
	reportHash = newHash

	if config.GlobalConfig.LastRulesHash != "" && newHash == config.GlobalConfig.LastRulesHash {
		log.Println("ℹ️ 配置未变化（200，哈希相同），跳过写入与重载")
		fetchStatus = "unchanged"
		reloadStatus = "skipped"
		ReportSyncStatus(reportHash, fetchStatus, reloadStatus, errMsg)
		return
	}

	var files []filePayload
	if err := json.Unmarshal(body, &files); err != nil {
		log.Printf("❌ 解析 JSON 失败: %v", err)
		errMsg = err.Error()
		ReportSyncStatus(reportHash, fetchStatus, reloadStatus, errMsg)
		return
	}

	if len(files) == 0 {
		log.Println("⚠️ 后端返回空配置，执行清理")
	}

	allowedSet := make(map[string]struct{})
	for _, p := range config.GlobalConfig.RulePaths {
		abs, _ := filepath.Abs(p)
		allowedSet[filepath.Clean(abs)] = struct{}{}
	}

	targetSet := make(map[string]struct{})
	for _, f := range files {
		fp := filepath.Clean(f.FilePath)
		if !isInManagedScope(fp, allowedSet) {
			log.Printf("⚠️ 跳过未授权路径: %s", fp)
			continue
		}
		targetSet[fp] = struct{}{}
		if err := atomicWrite(fp, []byte(f.Content), 0644); err != nil {
			log.Printf("❌ 写入文件失败 %s: %v", fp, err)
			errMsg = err.Error()
			ReportSyncStatus(reportHash, fetchStatus, reloadStatus, errMsg)
			return
		}
	}

	if err := cleanupExtraFiles(allowedSet, targetSet); err != nil {
		log.Printf("⚠️ 清理异常: %v", err)
	}

	config.GlobalConfig.LastRulesHash = newHash
	saveHashToFile(newHash)

	fetchStatus = "updated"
	log.Println("✅ 规则已更新，正在重载 vmalert...")
	ok, reloadErr := ReloadVMAlert()
	if ok {
		reloadStatus = "success"
	} else {
		reloadStatus = "failed"
		errMsg = reloadErr
	}

	ReportSyncStatus(reportHash, fetchStatus, reloadStatus, errMsg)
}

// CollectLocalRules 收集本地配置目录下的规则文件
func CollectLocalRules() ([]filePayload, error) {
	var results []filePayload
	for _, p := range config.GlobalConfig.RulePaths {
		info, err := os.Stat(p)
		if err != nil {
			log.Printf("⚠️ 跳过无法访问的路径 %s: %v", p, err)
			continue
		}
		if !info.IsDir() {
			// 单个文件
			content, err := os.ReadFile(p)
			if err != nil {
				log.Printf("⚠️ 读取文件失败 %s: %v", p, err)
				continue
			}
			abs, _ := filepath.Abs(p)
			results = append(results, filePayload{
				FilePath: abs,
				Content:  string(content),
			})
		} else {
			// 目录遍历
			err := filepath.Walk(p, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return nil
				}
				if !info.IsDir() && (strings.HasSuffix(path, ".yaml") || strings.HasSuffix(path, ".yml") || strings.HasSuffix(path, ".rule")) {
					content, err := os.ReadFile(path)
					if err != nil {
						log.Printf("⚠️ 读取文件失败 %s: %v", path, err)
						return nil
					}
					abs, _ := filepath.Abs(path)
					results = append(results, filePayload{
						FilePath: abs,
						Content:  string(content),
					})
				}
				return nil
			})
			if err != nil {
				log.Printf("⚠️ 遍历目录失败 %s: %v", p, err)
			}
		}
	}
	return results, nil
}

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

func ReloadVMAlert() (bool, string) {
	url := fmt.Sprintf("%s/-/reload", config.GlobalConfig.VMAlertURL)
	resp, err := http.Post(url, "", nil)
	if err != nil {
		msg := fmt.Sprintf("重载 vmalert 失败: %v", err)
		log.Printf("⚠️ %s", msg)
		return false, msg
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		msg := fmt.Sprintf("vmalert 重载返回非 200 状态: %s", resp.Status)
		log.Printf("⚠️ %s", msg)
		return false, msg
	}

	log.Println("✅ vmalert 重载成功")
	return true, ""
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

func parseHashFromHeader(etag string, headerHash string) string {
	if headerHash != "" {
		return strings.Trim(headerHash, "\"")
	}
	if etag == "" {
		return ""
	}
	clean := strings.TrimSpace(etag)
	clean = strings.TrimPrefix(clean, "W/")
	clean = strings.Trim(clean, "\"")
	return clean
}

func computeHash(data []byte) string {
	sum := sha256.Sum256(data)
	return fmt.Sprintf("%x", sum[:])
}

func isInManagedScope(filePath string, scopes map[string]struct{}) bool {
	abs, _ := filepath.Abs(filePath)
	for scope := range scopes {
		info, err := os.Stat(scope)
		if err != nil {
			continue
		}
		if info.IsDir() {
			if strings.HasPrefix(abs, scope+string(os.PathSeparator)) || abs == scope {
				return true
			}
		} else {
			if abs == scope {
				return true
			}
		}
	}
	return false
}

func atomicWrite(path string, data []byte, perm os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, perm); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func cleanupExtraFiles(scopes map[string]struct{}, keep map[string]struct{}) error {
	for scope := range scopes {
		info, err := os.Stat(scope)
		if err != nil {
			continue
		}
		if info.IsDir() {
			err = filepath.Walk(scope, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return nil
				}
				if info.IsDir() {
					return nil
				}
				abs := filepath.Clean(path)
				if _, ok := keep[abs]; !ok {
					if rmErr := os.Remove(abs); rmErr != nil {
						log.Printf("⚠️ 删除文件失败 %s: %v", abs, rmErr)
					} else {
						log.Printf("🧹 删除多余文件: %s", abs)
					}
				}
				return nil
			})
			if err != nil {
				return err
			}
		} else {
			abs := filepath.Clean(scope)
			if _, ok := keep[abs]; !ok {
				if err := os.WriteFile(abs, []byte{}, info.Mode()); err != nil {
					log.Printf("⚠️ 清空文件失败 %s: %v", abs, err)
				} else {
					log.Printf("🧹 清空文件: %s", abs)
				}
			}
		}
	}
	return nil
}

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
