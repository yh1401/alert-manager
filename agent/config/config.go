package config

import (
	"flag"
	"fmt"
	"math/rand"
	"net"
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

// Config 保存 Agent 的运行时配置
type Config struct {
	// 基本信息
	Name       string
	BackendURL string
	NodeID     string

	// vmalert 与 prometheus 相关 URL（可选）
	VMAlertURL    string // vmalert 地址（当未指定 prometheus 时可作为重载后端）
	PrometheusURL string // Prometheus 地址（如果指定，则使用 Prometheus 的 /-/reload）
	ReloadURL     string // 最终的重载 URL（可由命令行显式指定）
	ReloadBackend string // 指示使用哪个后端进行重载："custom"|"prometheus"|"vmalert"

	// 时间间隔配置
	PollInterval time.Duration
	HeartbeatInt time.Duration

	// 规则路径 / 状态 / 日志
	RulesFilePath string   // 兼容旧字段，默认取 rule_paths[0]
	RulePaths     []string // 支持多规则文件/目录
	StateFilePath string
	LogFilePath   string
	LogMaxAge     int

	// 本地缓存的规则哈希
	LastRulesHash string

	// 可选监听端口。为 0 表示自动分配一个可用端口（区间 30000-65535）
	Port int
}

// ParseFlags 解析命令行参数并做一些基础初始化（如端口自动分配）
func ParseFlags() {
	flag.StringVar(&GlobalConfig.Name, "name", "", "Agent 启动时指定的名字，如果为空则使用 hostname")
	flag.StringVar(&GlobalConfig.BackendURL, "backend", "http://localhost:8080", "后端服务地址")
	flag.StringVar(&GlobalConfig.VMAlertURL, "vmalert_url", "http://localhost:8880", "vmalert 服务地址（当未指定 prometheus 时作为重载后端）")
	flag.StringVar(&GlobalConfig.PrometheusURL, "prometheus_url", "", "Prometheus 地址（可选），若指定则使用 Prometheus 的 /-/reload 来重载规则")
	flag.StringVar(&GlobalConfig.ReloadURL, "reload_url", "", "显式重载 URL（可选），优先级高于 prometheus_url 与 vmalert_url）")
	flag.DurationVar(&GlobalConfig.PollInterval, "poll_interval", 10*time.Second, "配置拉取间隔")
	flag.DurationVar(&GlobalConfig.HeartbeatInt, "heartbeat_interval", 30*time.Second, "心跳上报间隔")

	// 默认包含一个路径，用户可多次 -rule_path 追加
	defaultRulePath := "./rules.yaml"
	rulePaths := stringSliceFlag{defaultRulePath}
	flag.Var(&rulePaths, "rule_path", "规则文件或目录，可多次指定")

	flag.StringVar(&GlobalConfig.StateFilePath, "state_file", "./agent-state.yaml", "Agent 状态文件（包含 NodeID、规则哈希与重载信息）")
	flag.StringVar(&GlobalConfig.LogFilePath, "log_file", "./agent.log", "日志文件路径")
	flag.IntVar(&GlobalConfig.LogMaxAge, "log_max_age", 7, "日志保留天数")

	// 可选端口参数：0（默认）表示自动在 30000-65535 范围内选择一个空闲端口
	flag.IntVar(&GlobalConfig.Port, "port", 0, "Agent 使用的端口（可选）。设为 0 则在 30000-65535 范围内自动选择一个空闲端口")

	flag.Parse()

	if len(rulePaths) == 0 {
		rulePaths = stringSliceFlag{defaultRulePath}
	}
	GlobalConfig.RulePaths = rulePaths
	GlobalConfig.RulesFilePath = rulePaths[0]

	// 如果未指定端口，则尝试在 30000-65535 区间内找到一个空闲端口
	if GlobalConfig.Port == 0 {
		// 优先随机尝试若干次，随后顺序扫描作为回退
		rand.Seed(time.Now().UnixNano())
		const minPort = 30000
		const maxPort = 65535
		const randomTries = 200

		found := 0
		for i := 0; i < randomTries; i++ {
			p := rand.Intn(maxPort-minPort+1) + minPort
			if portAvailable(p) {
				found = p
				break
			}
		}

		if found == 0 {
			// 顺序扫描回退
			for p := minPort; p <= maxPort; p++ {
				if portAvailable(p) {
					found = p
					break
				}
			}
		}

		if found != 0 {
			GlobalConfig.Port = found
		} else {
			// 如果未能找到可用端口，保留 0（表示未分配），调用方可决定处理逻辑
			fmt.Println("⚠️ 未能在 30000-65535 范围内找到空闲端口，port 保持为 0")
		}
	}

	// 输出使用的端口，方便在启动日志中查看（主程序已在后续初始化日志系统）
	if GlobalConfig.Port != 0 {
		fmt.Printf("使用端口: %d\n", GlobalConfig.Port)
	} else {
		fmt.Printf("未指定端口，且未能分配到自动端口（port=0）\n")
	}
}

// portAvailable 尝试监听指定端口，若能成功绑定则认为端口可用
func portAvailable(port int) bool {
	addr := fmt.Sprintf(":%d", port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return false
	}
	_ = ln.Close()
	return true
}
