package utils

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"
)

var defaultPromtoolPath = getDefaultPromtoolPath()

// getDefaultPromtoolPath 从环境变量或本地路径获取 promtool 位置
func getDefaultPromtoolPath() string {
	// 优先从环境变量读
	if p := os.Getenv("PROMTOOL_PATH"); p != "" {
		return p
	}

	// 其次尝试系统 PATH 中的 promtool
	if p, err := exec.LookPath("promtool"); err == nil {
		return p
	}

	// 最后降级到本地路径
	return "./tools/promtool/promtool"
}

// ValidateRulesWithPromtool 调用 promtool 检查规则文件语法。
// 返回 promtool 输出和错误（若有）。
func ValidateRulesWithPromtool(promtoolPath string, rulesContent string) (string, error) {
	if promtoolPath == "" {
		promtoolPath = defaultPromtoolPath
	}

	// 检查 promtool 是否存在
	if _, err := os.Stat(promtoolPath); err != nil {
		return "", fmt.Errorf("promtool 未找到: %s (运行 backend/tools/download_promtool.sh 下载)", promtoolPath)
	}

	// 写入临时文件
	tmpFile, err := os.CreateTemp("", "rules-*.yaml")
	if err != nil {
		return "", fmt.Errorf("创建临时文件失败: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(rulesContent); err != nil {
		return "", fmt.Errorf("写入临时文件失败: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		return "", fmt.Errorf("关闭临时文件失败: %w", err)
	}

	// 设置超时避免 promtool 卡住
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, promtoolPath, "check", "rules", tmpFile.Name())
	output, err := cmd.CombinedOutput()

	if ctx.Err() == context.DeadlineExceeded {
		return string(output), fmt.Errorf("promtool 校验超时 (15s)")
	}

	if err != nil {
		return string(output), fmt.Errorf("promtool 校验失败: %w", err)
	}

	return string(output), nil
}
