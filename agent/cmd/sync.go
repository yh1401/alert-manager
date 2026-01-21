package cmd

import (
	"alert-manager-agent/config"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type filePayload struct {
	FilePath string `json:"file_path"`
	Content  string `json:"content"`
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
		errMsg := fmt.Sprintf("服务器返回错误: %s", resp.Status)
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
