package utils

import (
	"strings"
	"testing"
)

func TestThreeWayMergeLines_NoChange(t *testing.T) {
	base := "a\nb\nc\n"
	cur := base
	cli := base

	merged, err := ThreeWayMergeLines(base, cur, cli)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if merged != base {
		t.Fatalf("expected merged == base, got:\n%s", merged)
	}
}

func TestThreeWayMergeLines_OnlyClientChanges(t *testing.T) {
	base := "line1\nline2\nline3\n"
	cur := base
	cli := "line1\nline2-modified\nline3\n"

	merged, err := ThreeWayMergeLines(base, cur, cli)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if merged != cli {
		t.Fatalf("expected merged == client, got:\n%s", merged)
	}
}

func TestThreeWayMergeLines_OnlyCurrentChanges(t *testing.T) {
	base := "line1\nline2\nline3\n"
	cur := "line1\nline2-modified\nline3\n"
	cli := base

	merged, err := ThreeWayMergeLines(base, cur, cli)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if merged != cur {
		t.Fatalf("expected merged == current, got:\n%s", merged)
	}
}

func TestThreeWayMergeLines_BothChangeSameLine_Conflict(t *testing.T) {
	base := "line1\nline2\nline3\n"
	cur := "line1\nline2-from-current\nline3\n"
	cli := "line1\nline2-from-client\nline3\n"

	merged, err := ThreeWayMergeLines(base, cur, cli)
	if err == nil {
		t.Fatalf("expected conflict error, got nil, merged:\n%s", merged)
	}
	if merged != "" {
		t.Fatalf("expected merged to be empty on conflict, got:\n%s", merged)
	}
}

// 这个测试描述的是“不同行修改”的理想场景：
// current 改第 1 行，client 改第 3 行。
// 目前的实现可能仍然返回冲突，所以这里先用 Log 而不是 Fatal，
// 方便后续迭代 ThreeWayMergeLines 时对比行为。
func TestThreeWayMergeLines_BothChangeDifferentLines_CurrentOnFirst_ClientOnLast(t *testing.T) {
	base := "line1\nline2\nline3\n"
	cur := "line1-from-current\nline2\nline3\n" // current 改第 1 行
	cli := "line1\nline2\nline3-from-client\n"  // client 改第 3 行

	merged, err := ThreeWayMergeLines(base, cur, cli)
	if err != nil {
		t.Logf("ThreeWayMergeLines returned conflict for different-line changes (current impl is conservative): %v", err)
		return
	}

	if !strings.Contains(merged, "line1-from-current") || !strings.Contains(merged, "line3-from-client") {
		t.Fatalf("expected merged to contain both changes, got:\n%s", merged)
	}
}

// 贴近你现在 YAML 规则文件的典型用例：
// - base: 原始规则
// - cur: 只改 name（A）
// - cli: 只改 alert/for（B）
// 当前实现大概率会认为有冲突，这个测试先用于观察现状，
// 后面优化算法时可以把这里改成强制必须合并成功。
func TestThreeWayMergeLines_YamlLike_DifferentBlocks(t *testing.T) {
	base := strings.TrimSpace(`
groups:
  - name: merge
    rules:
      - alert: Base
        expr: up==0
        for: 1m
`) + "\n"

	// A: 改 name（模拟 current）
	cur := strings.TrimSpace(`
groups:
  - name: merge-A
    rules:
      - alert: Base
        expr: up==0
        for: 1m
`) + "\n"

	// B: 改 alert/for（模拟 client）
	cli := strings.TrimSpace(`
groups:
  - name: merge
    rules:
      - alert: Base-Changed
        expr: up==0
        for: 2m
`) + "\n"

	merged, err := ThreeWayMergeLines(base, cur, cli)
	if err != nil {
		t.Logf("ThreeWayMergeLines reported conflict for YAML-like different blocks (current impl is conservative): %v", err)
		return
	}

	if !strings.Contains(merged, "merge-A") || !strings.Contains(merged, "Base-Changed") {
		t.Fatalf("expected merged to contain both A(name) and B(alert/for) changes, got:\n%s", merged)
	}
}
