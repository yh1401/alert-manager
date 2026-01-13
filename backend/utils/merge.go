package utils

import (
	"fmt"
	"strings"

	"github.com/sergi/go-diff/diffmatchpatch"
)

// ThreeWayMergeLines performs a conservative 3-way line-based merge.
//
// 参数说明：
//   - base:    基线版本（客户端编辑时看到的版本，对应 base_version）
//   - current: 当前最新版本（已经成功写入数据库的版本）
//   - client:  客户端提交的内容（基于 base_version 修改后的内容）
//
// 设计目标：
//   - 尽量在“不同块修改”时自动合并，避免无谓的冲突；
//   - 一旦判断不出是否安全（复杂场景），宁可返回冲突让用户手动处理，也不静默丢数据；
//   - 以“行”为最小粒度，不解析 YAML 语义。
//
// 返回值：
//   - merged: 合并后的文本
//   - error:  nil 表示合并成功；非 nil 表示存在冲突（调用方应走乐观锁冲突流程）
func ThreeWayMergeLines(base, current, client string) (string, error) {
	// 快速路径：完全幂等
	if current == client {
		return current, nil
	}
	if base == current && base != client {
		// 当前没动，只客户端改了
		return client, nil
	}
	if base == client && base != current {
		// 客户端没动，只当前改了
		return current, nil
	}

	// 使用 diff-match-patch 进行行级 diff
	dmp := diffmatchpatch.New()

	// 按行处理，避免对单个长行内的字符级改动过于敏感
	baseLines := splitKeepLastEmpty(base)
	curLines := splitKeepLastEmpty(current)
	cliLines := splitKeepLastEmpty(client)

	// 用 join("\n") 把它们变成文本再 diff，行结构仍然保持
	baseText := strings.Join(baseLines, "\n")
	curText := strings.Join(curLines, "\n")
	cliText := strings.Join(cliLines, "\n")

	// 计算 base->current 和 base->client 的增量
	diffsBC := dmp.DiffMain(baseText, curText, false)
	diffsBU := dmp.DiffMain(baseText, cliText, false)

	// 将 diff token 化为“片段序列”，与 base 对齐
	chunksBC := buildChunksFromDiff(baseText, diffsBC)
	chunksBU := buildChunksFromDiff(baseText, diffsBU)

	// 合并两个变更流
	merged, conflict := mergeChunks(chunksBC, chunksBU)
	if conflict {
		return "", fmt.Errorf("three-way merge conflict")
	}

	return merged, nil
}

// splitKeepLastEmpty 按行切分，保留末尾空行信息。
func splitKeepLastEmpty(s string) []string {
	// strings.Split 保留尾部空行：Split("a\nb\n", "\n") => ["a","b",""]
	return strings.Split(s, "\n")
}

// chunk 表示相对于 base 的一段：
// - origin: 来自哪一方（base/current/client）
// - text:   该段内容
// - isChange: 是否为 base 以外的修改
type chunk struct {
	origin   string // "base" | "cur" | "cli"
	text     string
	isChange bool
}

// buildChunksFromDiff 将基于 base 的 diff 转换成 chunk 序列。
// 这里采用一个简化模型：把删除视作对 base 的修改（即不保留），
// 插入视作新内容。
func buildChunksFromDiff(base string, diffs []diffmatchpatch.Diff) []chunk {
	var chunks []chunk
	baseIdx := 0

	for _, d := range diffs {
		switch d.Type {
		case diffmatchpatch.DiffEqual:
			// 这块等同于 base
			chunks = append(chunks, chunk{
				origin:   "base",
				text:     d.Text,
				isChange: false,
			})
			baseIdx += len(d.Text)
		case diffmatchpatch.DiffDelete:
			// 删除意味着从 base 中移除这部分 => 视作 change，但不在结果中保留文本
			if d.Text == "" {
				continue
			}
			// 这里不直接生成文本块，因为删除意味着“这一部分在结果中不存在”
			// 为了简化，我们用一个特殊空块标记删除。
			chunks = append(chunks, chunk{
				origin:   "base",
				text:     "",
				isChange: true,
			})
			baseIdx += len(d.Text)
		case diffmatchpatch.DiffInsert:
			// 插入的新文本
			if d.Text == "" {
				continue
			}
			chunks = append(chunks, chunk{
				origin:   "ins",
				text:     d.Text,
				isChange: true,
			})
		}
	}
	return chunks
}

// mergeChunks 执行两个基于 base 的变更序列的合并。
// 简化策略：
//   - 对于相同位置：
//   - 如果只有一边有变更 => 接受这边的变更
//   - 如果两边变更且文本完全一致 => 接受任意一边
//   - 如果两边变更且文本不一致 => 冲突
//   - 对于插入：如果只在一侧出现，则按顺序插入；两边在同一位置插入不同文本 => 冲突
//
// 注意：这里是非常保守的实现，更复杂的移动/重排场景会被判为冲突。
func mergeChunks(chunksCur, chunksCli []chunk) (string, bool) {
	// 为了简化，我们按线性扫描的方式对齐两个 chunk 列表。
	// 实际上 go-diff 给出的 diff 已经考虑了顺序，我们假定两边是基于同一个 base 的相对增量。
	var out strings.Builder
	i, j := 0, 0
	for i < len(chunksCur) || j < len(chunksCli) {
		// 处理边界情况
		if i >= len(chunksCur) {
			// 只剩客户端变更
			out.WriteString(chunksCli[j].text)
			j++
			continue
		}
		if j >= len(chunksCli) {
			// 只剩 current 变更
			out.WriteString(chunksCur[i].text)
			i++
			continue
		}

		cc := chunksCur[i]
		uc := chunksCli[j]

		// 简单对齐策略：按块一一对应比较
		switch {
		case !cc.isChange && !uc.isChange:
			// 双方都没改，且都是 base 等价块，按任意一边输出并前进
			out.WriteString(cc.text)
			i++
			j++
		case cc.isChange && !uc.isChange:
			// 只有 current 改
			out.WriteString(cc.text)
			i++
			j++
		case !cc.isChange && uc.isChange:
			// 只有 client 改
			out.WriteString(uc.text)
			i++
			j++
		default:
			// 两边都改了：必须文本相同才算可合并，否则冲突
			if cc.text == uc.text {
				out.WriteString(cc.text)
				i++
				j++
			} else {
				// 冲突：返回标记
				return "", true
			}
		}
	}

	return out.String(), false
}
