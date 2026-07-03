package prometheus

import (
	"context"
	"fmt"
	"time"
)

// UnifiedAlert 统一告警格式（兼容 VMAlert 和 Prometheus）
type UnifiedAlert struct {
	Name        string            `json:"name"`         // 告警名称
	State       string            `json:"state"`        // 状态: firing, pending, resolved
	Severity    string            `json:"severity"`     // 严重级别: critical, warning, info
	Labels      map[string]string `json:"labels"`       // 标签
	Annotations map[string]string `json:"annotations"` // 注解
	ActiveAt    string            `json:"active_at"`    // 激活时间
	Value       string            `json:"value"`        // 当前值
	Source      string            `json:"source"`       // 数据来源: prometheus / vmalert
}

// UnifiedRule 统一规则格式
type UnifiedRule struct {
	Name           string            `json:"name"`           // 规则名称
	Group          string            `json:"group"`          // 所属规则组
	File           string            `json:"file"`           // 规则文件
	Query          string            `json:"query"`          // PromQL 表达式
	Health         string            `json:"health"`         // 评估健康: ok, err
	LastError      string            `json:"last_error"`     // 最后错误
	State          string            `json:"state"`          // 状态
	Labels         map[string]string `json:"labels"`         // 标签
	Annotations    map[string]string `json:"annotations"`    // 注解
	EvaluationTime float64           `json:"evaluation_time"` // 评估耗时
	LastEvaluation string            `json:"last_evaluation"`  // 最后评估时间
	Source         string            `json:"source"`          // 数据来源
}

// UnifiedTarget 统一采集目标格式
type UnifiedTarget struct {
	ScrapeURL        string  `json:"scrape_url"`        // 采集 URL
	Job              string  `json:"job"`               // 任务名称
	Instance         string  `json:"instance"`          // 实例
	Health           string  `json:"health"`            // 健康状态: up, down
	LastError        string  `json:"last_error"`        // 最后错误
	LastScrape       string  `json:"last_scrape"`       // 最后采集时间
	LastScrapeDuration float64 `json:"last_scrape_duration"` // 采集耗时
	Source           string  `json:"source"`            // 数据来源
}

// AggregateAlerts 聚合告警数据，统一为标准格式
func (c *Client) AggregateAlerts(ctx context.Context) ([]UnifiedAlert, error) {
	alerts, err := c.GetAlerts(ctx)
	if err != nil {
		return nil, err
	}

	unified := make([]UnifiedAlert, 0, len(alerts.Alerts))
	for _, a := range alerts.Alerts {
		severity := "info"
		if s, ok := a.Labels["severity"]; ok {
			severity = s
		}

		name := a.Labels["alertname"]
		if name == "" {
			name = "unknown"
		}

		unified = append(unified, UnifiedAlert{
			Name:        name,
			State:       a.State,
			Severity:    severity,
			Labels:      a.Labels,
			Annotations: a.Annotations,
			ActiveAt:    a.ActiveAt,
			Value:       a.Value,
			Source:      "prometheus",
		})
	}

	return unified, nil
}

// AggregateRules 聚合规则数据，统一为标准格式
func (c *Client) AggregateRules(ctx context.Context) ([]UnifiedRule, error) {
	rules, err := c.GetRules(ctx)
	if err != nil {
		return nil, err
	}

	var unified []UnifiedRule
	for _, group := range rules.Groups {
		for _, rule := range group.Rules {
			state := rule.State
			if state == "" {
				state = "ok"
			}

			unified = append(unified, UnifiedRule{
				Name:           rule.Name,
				Group:          group.Name,
				File:           group.File,
				Query:          rule.Query,
				Health:         rule.Health,
				LastError:      rule.LastError,
				State:          state,
				Labels:         rule.Labels,
				Annotations:    rule.Annotations,
				EvaluationTime: rule.EvaluationTime,
				LastEvaluation: rule.LastEvaluation,
				Source:         "prometheus",
			})
		}
	}

	return unified, nil
}

// AggregateTargets 聚合采集目标数据，统一为标准格式
func (c *Client) AggregateTargets(ctx context.Context) ([]UnifiedTarget, error) {
	targets, err := c.GetTargets(ctx)
	if err != nil {
		return nil, err
	}

	var unified []UnifiedTarget
	for _, entry := range targets.ActiveTargets {
		for _, t := range entry.Targets {
			job := entry.Labels["job"]
			if job == "" {
				job = t.ScrapePool
			}
			instance := t.Labels["instance"]
			if instance == "" {
				instance = t.ScrapeURL
			}

			unified = append(unified, UnifiedTarget{
				ScrapeURL:         t.ScrapeURL,
				Job:               job,
				Instance:          instance,
				Health:            t.Health,
				LastError:         t.LastError,
				LastScrape:        t.LastScrape,
				LastScrapeDuration: t.LastScrapeDuration,
				Source:            "prometheus",
			})
		}
	}

	return unified, nil
}

// GetOverview 获取 Prometheus 数据概览（用于仪表盘）
func (c *Client) GetOverview(ctx context.Context) (map[string]interface{}, error) {
	type result struct {
		key string
		val interface{}
		err error
	}

	ch := make(chan result, 3)

	// 并发获取三类数据
	go func() {
		alerts, err := c.AggregateAlerts(ctx)
		ch <- result{"alerts", alerts, err}
	}()

	go func() {
		rules, err := c.AggregateRules(ctx)
		ch <- result{"rules", rules, err}
	}()

	go func() {
		targets, err := c.AggregateTargets(ctx)
		ch <- result{"targets", targets, err}
	}()

	overview := make(map[string]interface{})
	var errs []error

	for i := 0; i < 3; i++ {
		r := <-ch
		if r.err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", r.key, r.err))
			overview[r.key] = []interface{}{}
		} else {
			overview[r.key] = r.val
		}
	}

	// 统计摘要
	alerts, _ := overview["alerts"].([]UnifiedAlert)
	rules, _ := overview["rules"].([]UnifiedRule)
	targets, _ := overview["targets"].([]UnifiedTarget)

	overview["summary"] = map[string]interface{}{
		"total_alerts":      len(alerts),
		"firing_alerts":     countFiring(alerts),
		"pending_alerts":   countPending(alerts),
		"total_rules":      len(rules),
		"healthy_rules":    countHealthyRules(rules),
		"failed_rules":     countFailedRules(rules),
		"total_targets":    len(targets),
		"up_targets":       countUpTargets(targets),
		"down_targets":     countDownTargets(targets),
		"timestamp":        time.Now().Format(time.RFC3339),
	}

	if len(errs) > 0 {
		return overview, fmt.Errorf("部分数据获取失败: %v", errs)
	}

	return overview, nil
}

func countFiring(alerts []UnifiedAlert) int {
	count := 0
	for _, a := range alerts {
		if a.State == "firing" {
			count++
		}
	}
	return count
}

func countPending(alerts []UnifiedAlert) int {
	count := 0
	for _, a := range alerts {
		if a.State == "pending" {
			count++
		}
	}
	return count
}

func countHealthyRules(rules []UnifiedRule) int {
	count := 0
	for _, r := range rules {
		if r.Health == "ok" {
			count++
		}
	}
	return count
}

func countFailedRules(rules []UnifiedRule) int {
	count := 0
	for _, r := range rules {
		if r.Health == "err" {
			count++
		}
	}
	return count
}

func countUpTargets(targets []UnifiedTarget) int {
	count := 0
	for _, t := range targets {
		if t.Health == "up" {
			count++
		}
	}
	return count
}

func countDownTargets(targets []UnifiedTarget) int {
	count := 0
	for _, t := range targets {
		if t.Health == "down" {
			count++
		}
	}
	return count
}
