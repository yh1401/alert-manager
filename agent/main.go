package main

import (
	"alert-manager-agent/cmd"
	"alert-manager-agent/config"
	"log"
	"os"
	"time"

	"github.com/natefinch/lumberjack"
)

func main() {
	config.ParseFlags()

	// 预加载本地规则哈希，便于与服务端 ETag 比对
	cmd.LoadCachedHash()

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

	log.Printf("Agent 启动... NodeID: %s, 后端: %s", config.GlobalConfig.NodeID, config.GlobalConfig.BackendURL)

	ticker := time.NewTicker(config.GlobalConfig.PollInterval)

	// 启动心跳上报协程
	go cmd.StartHeartbeat()

	// 立即执行一次
	cmd.FetchConfig()

	for range ticker.C {
		cmd.FetchConfig()
	}
}
