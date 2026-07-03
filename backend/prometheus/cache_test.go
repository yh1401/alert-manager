package prometheus

import (
	"testing"
	"time"
)

// TestCacheSetGet 测试缓存的写入和读取
func TestCacheSetGet(t *testing.T) {
	cache := NewCache(5 * time.Second)

	// 测试写入和读取
	cache.Set("key1", "value1")

	val, ok := cache.Get("key1")
	if !ok {
		t.Fatal("缓存未命中: key1")
	}
	if val != "value1" {
		t.Fatalf("缓存值不匹配: expected value1, got %v", val)
	}
}

// TestCacheMiss 测试缓存未命中
func TestCacheMiss(t *testing.T) {
	cache := NewCache(5 * time.Second)

	_, ok := cache.Get("nonexistent")
	if ok {
		t.Fatal("不应命中不存在的 key")
	}
}

// TestCacheExpiry 测试缓存过期
func TestCacheExpiry(t *testing.T) {
	cache := NewCache(50 * time.Millisecond)

	cache.Set("key1", "value1")

	// 立即读取应命中
	_, ok := cache.Get("key1")
	if !ok {
		t.Fatal("缓存应命中")
	}

	// 等待过期
	time.Sleep(60 * time.Millisecond)

	_, ok = cache.Get("key1")
	if ok {
		t.Fatal("缓存应已过期")
	}
}

// TestCacheClear 测试清空缓存
func TestCacheClear(t *testing.T) {
	cache := NewCache(5 * time.Second)

	cache.Set("key1", "value1")
	cache.Set("key2", "value2")

	cache.Clear()

	_, ok1 := cache.Get("key1")
	_, ok2 := cache.Get("key2")

	if ok1 || ok2 {
		t.Fatal("缓存清空后不应命中任何 key")
	}
}

// TestCalculateBackoff 测试指数退避计算
func TestCalculateBackoff(t *testing.T) {
	baseDelay := time.Second

	// 测试不同 attempt 的退避时间
	for attempt := 0; attempt < 5; attempt++ {
		delay := calculateBackoff(attempt, baseDelay)

		// 退避时间应大于 0
		if delay <= 0 {
			t.Fatalf("attempt %d: 退避时间应大于 0, got %v", attempt, delay)
		}

		// 退避时间不应超过 base * 2^attempt * 1.5（最大抖动）
		maxExpected := time.Duration(float64(baseDelay) * float64(int(1)<<attempt) * 1.5)
		if delay > maxExpected {
			t.Fatalf("attempt %d: 退避时间 %v 超过最大值 %v", attempt, delay, maxExpected)
		}
	}
}

// TestCalculateBackoffZeroDelay 测试零延迟的默认值
func TestCalculateBackoffZeroDelay(t *testing.T) {
	delay := calculateBackoff(0, 0)
	if delay <= 0 {
		t.Fatal("零延迟应使用默认值")
	}
}
