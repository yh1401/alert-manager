package main

import (
	"alert-manager-agent/cmd"
	"alert-manager-agent/config"
	"log"
	"os"
	"strings"
	"time"

	"github.com/natefinch/lumberjack"
)

func main() {
	config.ParseFlags()

	// 预加载本地规则哈希，便于与服务端 ETag 比对
	cmd.LoadCachedHash()

	// 确定重载 URL 的优先级：explicit reload_url > prometheus_url + "/-/reload" > vmalert_url + "/api/v1/rules/reload"
	// 如果用户在命令行中显式设置了 ReloadURL（或从状态文件恢复到了 ReloadURL），则使用该 URL。
	// 否则根据 prom/vm 的配置构建默认的重载 URL，并保存到状态文件以便重启恢复。
	if config.GlobalConfig.ReloadURL != "" {
		// 已显式提供或从状态文件恢复，标记为 custom（但仍保存以确保状态文件包含信息）
		config.GlobalConfig.ReloadBackend = "custom"
	} else if config.GlobalConfig.PrometheusURL != "" {
		config.GlobalConfig.ReloadURL = strings.TrimRight(config.GlobalConfig.PrometheusURL, "/") + "/-/reload"
		config.GlobalConfig.ReloadBackend = "prometheus"
	} else if config.GlobalConfig.VMAlertURL != "" {
		// VM 的重载路径使用 /api/v1/rules/reload（由你确认）
		config.GlobalConfig.ReloadURL = strings.TrimRight(config.GlobalConfig.VMAlertURL, "/") + "/api/v1/rules/reload"
		config.GlobalConfig.ReloadBackend = "vmalert"
	}

	// 将重载信息保存到状态文件（如果已解析出 ReloadURL）
	if config.GlobalConfig.ReloadURL != "" {
		cmd.SaveReload(config.GlobalConfig.ReloadBackend, config.GlobalConfig.ReloadURL)
	}

	if config.GlobalConfig.LogFilePath != "" {
		_, err := os.OpenFile(config.GlobalConfig.LogFilePath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
		lumberJackLogger := &lumberjack.Logger{
			Filename:   config.GlobalConfig.LogFilePath,
			MaxSize:    500, // megabytes
			MaxBackups: 3,
			MaxAge:     config.GlobalConfig.LogMaxAge, //days
			Compress:   true,                          // disabled by default
		}
		if err != nil {
			log.Fatalf("error opening log file: %v", err)
		}
		log.SetOutput(lumberJackLogger)

	}

	// 获取或注册身份 (NodeID)
	if err := cmd.GetOrRegisterIdentity(); err != nil {
		log.Fatalf("❌ Agent 初始化失败: %v", err)
	}

	// 如果端口已确定，则写入状态文件保存端口
	if config.GlobalConfig.Port != 0 {
		cmd.SavePort(config.GlobalConfig.Port)
	}

	// 在启动日志中输出端口信息
	if config.GlobalConfig.Port != 0 {
		log.Printf("Agent 启动... NodeID: %s, 后端: %s, 端口: %d", config.GlobalConfig.NodeID, config.GlobalConfig.BackendURL, config.GlobalConfig.Port)
	} else {
		log.Printf("Agent 启动... NodeID: %s, 后端: %s, 端口: 未分配 (port=0)", config.GlobalConfig.NodeID, config.GlobalConfig.BackendURL)
	}

	ticker := time.NewTicker(config.GlobalConfig.PollInterval)

	// 启动心跳上报协程
	go cmd.StartHeartbeat()

	// 立即执行一次
	cmd.FetchConfig()

	for range ticker.C {
		cmd.FetchConfig()
	}
}
