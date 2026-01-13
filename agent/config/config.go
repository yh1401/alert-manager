package config

import (
	"flag"
	"strings"
	"time"
)

// 支持多次 -rule_path 的自定义 flag 类型
type stringSliceFlag []string

func (s *stringSliceFlag) String() string {
	return strings.Join(*s, ",")
}
func (s *stringSliceFlag) Set(v string) error {
	*s = append(*s, v)
	return nil
}

// GlobalConfig 全局配置单例
var GlobalConfig = &Config{}

type Config struct {
	BackendURL    string
	NodeID        string
	VMAlertURL    string
	PollInterval  time.Duration
	HeartbeatInt  time.Duration
	RulesFilePath string   // 兼容旧字段，默认取 rule_paths[0]
	RulePaths     []string // 支持多规则文件/目录
	StateFilePath string
	LogFilePath   string
	LogMaxAge     int
	LastRulesHash string
}

func ParseFlags() {
	flag.StringVar(&GlobalConfig.BackendURL, "backend", "http://localhost:8080", "后端服务地址")
	flag.StringVar(&GlobalConfig.VMAlertURL, "vmalert_url", "http://localhost:8880", "vmalert 服务地址")
	flag.DurationVar(&GlobalConfig.PollInterval, "poll_interval", 10*time.Second, "配置拉取间隔")
	flag.DurationVar(&GlobalConfig.HeartbeatInt, "heartbeat_interval", 30*time.Second, "心跳上报间隔")

	// 默认包含一个路径，用户可多次 -rule_path 追加
	defaultRulePath := "./rules.yaml"
	rulePaths := stringSliceFlag{defaultRulePath}
	flag.Var(&rulePaths, "rule_path", "规则文件或目录，可多次指定")

	flag.StringVar(&GlobalConfig.StateFilePath, "state_file", "./agent-state.yaml", "Agent 状态文件（包含 NodeID 和规则哈希）")
	flag.StringVar(&GlobalConfig.LogFilePath, "log_file", "./agent.log", "日志文件路径")
	flag.IntVar(&GlobalConfig.LogMaxAge, "log_max_age", 7, "日志保留天数")

	flag.Parse()

	if len(rulePaths) == 0 {
		rulePaths = stringSliceFlag{defaultRulePath}
	}
	GlobalConfig.RulePaths = rulePaths
	GlobalConfig.RulesFilePath = rulePaths[0]
}
