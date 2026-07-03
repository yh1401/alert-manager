package prometheus

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"time"
)

// RetryConfig 重试配置
type RetryConfig struct {
	MaxRetries int           // 最大重试次数
	BaseDelay   time.Duration // 基础延迟
}

// withRetry 执行带指数退避重试的操作
func withRetry(ctx context.Context, config RetryConfig, operation func() error) error {
	var lastErr error

	for attempt := 0; attempt <= config.MaxRetries; attempt++ {
		// 检查上下文是否已取消
		if ctx.Err() != nil {
			return ctx.Err()
		}

		err := operation()
		if err == nil {
			return nil
		}
		lastErr = err

		// 最后一次不再等待
		if attempt == config.MaxRetries {
			break
		}

		// 指数退避 + 随机抖动
		delay := calculateBackoff(attempt, config.BaseDelay)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
		}
	}

	return fmt.Errorf("重试 %d 次后仍然失败: %w", config.MaxRetries, lastErr)
}

// calculateBackoff 计算指数退避延迟，增加随机抖动避免惊群效应
func calculateBackoff(attempt int, baseDelay time.Duration) time.Duration {
	if baseDelay <= 0 {
		baseDelay = time.Second
	}

	// 指数退避: base * 2^attempt
	backoff := float64(baseDelay) * math.Pow(2, float64(attempt))

	// 增加随机抖动 (0.5x ~ 1.5x)
	jitter := 0.5 + rand.Float64()

	return time.Duration(backoff * jitter)
}
