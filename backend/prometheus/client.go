package prometheus

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"alert-manager-backend/models"
)

// Client Prometheus HTTP API 客户端
type Client struct {
	baseURL    string
	httpClient *http.Client
	cache      *Cache
	retry      RetryConfig
}

// NewClient 创建 Prometheus 客户端
func NewClient(config models.PrometheusConfig) *Client {
	timeout := config.Timeout
	if timeout <= 0 {
		timeout = 10 * time.Second
	}

	retryCount := config.RetryCount
	if retryCount <= 0 {
		retryCount = 3
	}

	retryInterval := config.RetryInterval
	if retryInterval <= 0 {
		retryInterval = 2 * time.Second
	}

	cacheTTL := config.CacheTTL
	if cacheTTL <= 0 {
		cacheTTL = 30 * time.Second
	}

	return &Client{
		baseURL: config.URL,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		cache: NewCache(cacheTTL),
		retry: RetryConfig{
			MaxRetries: retryCount,
			BaseDelay:   retryInterval,
		},
	}
}

// ---- Prometheus API 响应结构体 ----

// APIResponse Prometheus API 基础响应
type APIResponse struct {
	Status string          `json:"status"`
	Data   json.RawMessage `json:"data"`
	Error  string          `json:"error,omitempty"`
	Warnings []string       `json:"warnings,omitempty"`
}

// HealthResponse 健康检查响应
type HealthResponse struct {
	Status string `json:"status"`
}

// AlertItem Prometheus 活跃告警
type AlertItem struct {
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`
	State       string            `json:"state"`
	ActiveAt    string            `json:"activeAt"`
	Value       string            `json:"value"`
}

// AlertsData 告警数据
type AlertsData struct {
	Alerts []AlertItem `json:"alerts"`
}

// RuleGroup Prometheus 规则评估状态
type RuleGroup struct {
	Name           string         `json:"name"`
	File           string         `json:"file"`
	Rules          []RuleEntry    `json:"rules"`
	Interval       float64        `json:"interval"`
	EvaluationTime float64        `json:"evaluationTime"`
	LastEvaluation string         `json:"lastEvaluation"`
}

// RuleEntry 单条规则评估状态
type RuleEntry struct {
	Name          string            `json:"name"`
	Query         string            `json:"query"`
	Health        string            `json:"health"`
	LastError     string            `json:"lastError,omitempty"`
	Type          string            `json:"type"`
	Labels        map[string]string `json:"labels"`
	Annotations   map[string]string `json:"annotations"`
	Alerts        []AlertItem       `json:"alerts"`
	State         string            `json:"state"`
	EvaluationTime float64          `json:"evaluationTime"`
	LastEvaluation string           `json:"lastEvaluation"`
}

// RulesData 规则数据
type RulesData struct {
	Groups []RuleGroup `json:"groups"`
}

// TargetHealth 目标健康状态
type TargetHealth struct {
	LastError     string `json:"lastError"`
	LastScrape    string `json:"lastScrape"`
	LastScrapeDuration float64 `json:"lastScrapeDuration"`
	Health        string `json:"health"`
}

// Target 目标
type Target struct {
	DiscoveredLabels map[string]string `json:"discoveredLabels"`
	Labels           map[string]string `json:"labels"`
	ScrapePool       string           `json:"scrapePool"`
	ScrapeURL        string           `json:"scrapeUrl"`
	GlobalURL        string           `json:"globalUrl"`
	LastError        string           `json:"lastError"`
	LastScrape       string           `json:"lastScrape"`
	LastScrapeDuration float64        `json:"lastScrapeDuration"`
	Health           string           `json:"health"`
}

// TargetEntry 目标条目
type TargetEntry struct {
	Labels  map[string]string `json:"labels"`
	Targets []Target         `json:"targets"`
}

// TargetsData 目标数据
type TargetsData struct {
	ActiveTargets  []TargetEntry `json:"activeTargets"`
	DroppedTargets []TargetEntry `json:"droppedTargets"`
}

// QueryResult 查询结果
type QueryResult struct {
	ResultType string          `json:"resultType"`
	Result     json.RawMessage `json:"result"`
}

// QueryData 查询数据
type QueryData struct {
	Metric map[string]string `json:"metric"`
	Value  [2]interface{}    `json:"value"` // [timestamp, value]
}

// RangeQueryData 范围查询数据
type RangeQueryData struct {
	Metric map[string]string `json:"metric"`
	Values [][2]interface{}  `json:"values"` // [[timestamp, value], ...]
}

// ---- API 方法 ----

// doRequest 执行 HTTP 请求并解析响应
func (c *Client) doRequest(ctx context.Context, method, path string, params url.Values, cacheKey string) (json.RawMessage, error) {
	// 尝试从缓存获取
	if cacheKey != "" {
		if data, ok := c.cache.Get(cacheKey); ok {
			if raw, ok := data.(json.RawMessage); ok {
				return raw, nil
			}
		}
	}

	var result json.RawMessage

	err := withRetry(ctx, c.retry, func() error {
		u := c.baseURL + path
		if params != nil {
			u += "?" + params.Encode()
		}

		req, err := http.NewRequestWithContext(ctx, method, u, nil)
		if err != nil {
			return fmt.Errorf("创建请求失败: %w", err)
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return fmt.Errorf("请求 Prometheus 失败: %w", err)
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("读取响应失败: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("Prometheus 返回非 200 状态码: %d, body: %s", resp.StatusCode, string(body))
		}

		var apiResp APIResponse
		if err := json.Unmarshal(body, &apiResp); err != nil {
			return fmt.Errorf("解析响应失败: %w", err)
		}

		if apiResp.Status != "success" {
			return fmt.Errorf("Prometheus API 错误: %s", apiResp.Error)
		}

		result = apiResp.Data
		return nil
	})

	if err != nil {
		return nil, err
	}

	// 写入缓存
	if cacheKey != "" {
		c.cache.Set(cacheKey, result)
	}

	return result, nil
}

// Health 检查 Prometheus 服务健康状态
func (c *Client) Health(ctx context.Context) (*HealthResponse, error) {
	cacheKey := "health"

	if data, ok := c.cache.Get(cacheKey); ok {
		if h, ok := data.(*HealthResponse); ok {
			return h, nil
		}
	}

	u := c.baseURL + "/-/healthy"

	req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求 Prometheus 失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Prometheus 不健康, 状态码: %d", resp.StatusCode)
	}

	health := &HealthResponse{
		Status: string(body),
	}

	c.cache.Set(cacheKey, health)
	return health, nil
}

// GetAlerts 获取当前活跃的告警列表
func (c *Client) GetAlerts(ctx context.Context) (*AlertsData, error) {
	data, err := c.doRequest(ctx, "GET", "/api/v1/alerts", nil, "alerts")
	if err != nil {
		return nil, err
	}

	var alerts AlertsData
	if err := json.Unmarshal(data, &alerts); err != nil {
		return nil, fmt.Errorf("解析告警数据失败: %w", err)
	}

	return &alerts, nil
}

// GetRules 获取规则评估状态
func (c *Client) GetRules(ctx context.Context) (*RulesData, error) {
	data, err := c.doRequest(ctx, "GET", "/api/v1/rules", nil, "rules")
	if err != nil {
		return nil, err
	}

	var rules RulesData
	if err := json.Unmarshal(data, &rules); err != nil {
		return nil, fmt.Errorf("解析规则数据失败: %w", err)
	}

	return &rules, nil
}

// GetTargets 获取采集目标健康状态
func (c *Client) GetTargets(ctx context.Context) (*TargetsData, error) {
	data, err := c.doRequest(ctx, "GET", "/api/v1/targets", nil, "targets")
	if err != nil {
		return nil, err
	}

	var targets TargetsData
	if err := json.Unmarshal(data, &targets); err != nil {
		return nil, fmt.Errorf("解析目标数据失败: %w", err)
	}

	return &targets, nil
}

// Query 执行即时查询
func (c *Client) Query(ctx context.Context, query string) (*QueryResult, error) {
	params := url.Values{}
	params.Set("query", query)

	// 即时查询不缓存，确保数据实时性
	data, err := c.doRequest(ctx, "GET", "/api/v1/query", params, "")
	if err != nil {
		return nil, err
	}

	var result QueryResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("解析查询结果失败: %w", err)
	}

	return &result, nil
}

// QueryRange 执行范围查询
func (c *Client) QueryRange(ctx context.Context, query string, start, end time.Time, step string) (*QueryResult, error) {
	params := url.Values{}
	params.Set("query", query)
	params.Set("start", strconv.FormatInt(start.Unix(), 10))
	params.Set("end", strconv.FormatInt(end.Unix(), 10))
	params.Set("step", step)

	// 范围查询不缓存
	data, err := c.doRequest(ctx, "GET", "/api/v1/query_range", params, "")
	if err != nil {
		return nil, err
	}

	var result QueryResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("解析范围查询结果失败: %w", err)
	}

	return &result, nil
}

// ClearCache 清除客户端缓存
func (c *Client) ClearCache() {
	c.cache.Clear()
}
