package prometheus

import (
	"context"
	"errors"
	"testing"
	"time"
)

// TestRetrySuccess 测试首次成功不需要重试
func TestRetrySuccess(t *testing.T) {
	config := RetryConfig{
		MaxRetries: 3,
		BaseDelay:   10 * time.Millisecond,
	}

	callCount := 0
	err := withRetry(context.Background(), config, func() error {
		callCount++
		return nil
	})

	if err != nil {
		t.Fatalf("不应返回错误: %v", err)
	}
	if callCount != 1 {
		t.Fatalf("应只调用 1 次, 实际 %d 次", callCount)
	}
}

// TestRetryFailAll 测试所有重试都失败
func TestRetryFailAll(t *testing.T) {
	config := RetryConfig{
		MaxRetries: 2,
		BaseDelay:   10 * time.Millisecond,
	}

	callCount := 0
	expectedErr := errors.New("connection refused")
	err := withRetry(context.Background(), config, func() error {
		callCount++
		return expectedErr
	})

	if err == nil {
		t.Fatal("应返回错误")
	}
	if callCount != 3 { // 初始 + 2 次重试
		t.Fatalf("应调用 3 次, 实际 %d 次", callCount)
	}
}

// TestRetrySuccessOnRetry 测试重试后成功
func TestRetrySuccessOnRetry(t *testing.T) {
	config := RetryConfig{
		MaxRetries: 3,
		BaseDelay:   10 * time.Millisecond,
	}

	callCount := 0
	err := withRetry(context.Background(), config, func() error {
		callCount++
		if callCount < 3 {
			return errors.New("temporary failure")
		}
		return nil
	})

	if err != nil {
		t.Fatalf("不应返回错误: %v", err)
	}
	if callCount != 3 {
		t.Fatalf("应调用 3 次, 实际 %d 次", callCount)
	}
}

// TestRetryContextCanceled 测试上下文取消
func TestRetryContextCanceled(t *testing.T) {
	config := RetryConfig{
		MaxRetries: 5,
		BaseDelay:   100 * time.Millisecond,
	}

	ctx, cancel := context.WithCancel(context.Background())

	// 立即取消上下文
	cancel()

	callCount := 0
	err := withRetry(ctx, config, func() error {
		callCount++
		return errors.New("failure")
	})

	if err == nil {
		t.Fatal("应返回错误")
	}
	// 上下文已取消，应只调用 0 或 1 次
	if callCount > 1 {
		t.Fatalf("上下文取消后不应继续重试, 调用了 %d 次", callCount)
	}
}
