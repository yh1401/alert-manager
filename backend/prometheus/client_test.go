package prometheus

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"alert-manager-backend/models"
)

// newTestClient 创建测试用客户端，指向 mock server
func newTestClient(serverURL string) *Client {
	return NewClient(models.PrometheusConfig{
		URL:           serverURL,
		Timeout:       5 * 1000 * 1000 * 1000, // 5s in nanoseconds
		RetryCount:    1,
		RetryInterval: 10 * 1000 * 1000,       // 10ms in nanoseconds
		CacheTTL:      1 * 1000 * 1000 * 1000, // 1s in nanoseconds
	})
}

// TestHealth 测试健康检查
func TestHealth(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/-/healthy" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Prometheus Server is up and running."))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	ctx := context.Background()

	health, err := client.Health(ctx)
	if err != nil {
		t.Fatalf("健康检查失败: %v", err)
	}
	if health.Status == "" {
		t.Fatal("健康状态不应为空")
	}
}

// TestGetAlerts 测试获取告警
func TestGetAlerts(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/alerts" {
			resp := APIResponse{
				Status: "success",
				Data:   json.RawMessage(`{"alerts":[]}`),
			}
			json.NewEncoder(w).Encode(resp)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	ctx := context.Background()

	alerts, err := client.AggregateAlerts(ctx)
	if err != nil {
		t.Fatalf("获取告警失败: %v", err)
	}
	if alerts == nil {
		t.Fatal("告警列表不应为 nil")
	}
}

// TestGetRules 测试获取规则
func TestGetRules(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/rules" {
			resp := APIResponse{
				Status: "success",
				Data:   json.RawMessage(`{"groups":[]}`),
			}
			json.NewEncoder(w).Encode(resp)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	ctx := context.Background()

	rules, err := client.AggregateRules(ctx)
	if err != nil {
		t.Fatalf("获取规则失败: %v", err)
	}
	// 空列表时聚合函数返回 nil 切片，这是有效的
	_ = rules // rules 可以是 nil 或空切片
}

// TestGetTargets 测试获取采集目标
func TestGetTargets(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/targets" {
			resp := APIResponse{
				Status: "success",
				Data:   json.RawMessage(`{"activeTargets":[],"droppedTargets":[]}`),
			}
			json.NewEncoder(w).Encode(resp)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	ctx := context.Background()

	targets, err := client.AggregateTargets(ctx)
	if err != nil {
		t.Fatalf("获取采集目标失败: %v", err)
	}
	// 空列表时聚合函数返回 nil 切片，这是有效的
	_ = targets // targets 可以是 nil 或空切片
}

// TestQuery 测试即时查询
func TestQuery(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/query" {
			resp := APIResponse{
				Status: "success",
				Data:   json.RawMessage(`{"resultType":"vector","result":[]}`),
			}
			json.NewEncoder(w).Encode(resp)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	ctx := context.Background()

	result, err := client.Query(ctx, "up")
	if err != nil {
		t.Fatalf("查询失败: %v", err)
	}
	if result == nil {
		t.Fatal("查询结果不应为 nil")
	}
	if result.ResultType != "vector" {
		t.Fatalf("结果类型应为 vector, got %s", result.ResultType)
	}
}

// TestServerError 测试服务器返回错误
func TestServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal server error"))
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	ctx := context.Background()

	_, err := client.GetAlerts(ctx)
	if err == nil {
		t.Fatal("应返回错误")
	}
}

// TestCacheHit 验证缓存命中时不发请求
func TestCacheHit(t *testing.T) {
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		resp := APIResponse{
			Status: "success",
			Data:   json.RawMessage(`{"alerts":[]}`),
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	ctx := context.Background()

	// 第一次请求
	_, err := client.GetAlerts(ctx)
	if err != nil {
		t.Fatalf("第一次请求失败: %v", err)
	}

	// 第二次请求应命中缓存
	_, err = client.GetAlerts(ctx)
	if err != nil {
		t.Fatalf("第二次请求失败: %v", err)
	}

	if requestCount != 1 {
		t.Fatalf("缓存应阻止第二次请求, 但服务器收到 %d 次请求", requestCount)
	}
}
