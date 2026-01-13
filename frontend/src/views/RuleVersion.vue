<template>
    <div class="rule-version-container">
        <!-- 顶部信息 -->
        <div
            style="
                margin-bottom: 20px;
                padding: 15px;
                background: #f5f7fa;
                border-radius: 4px;
            "
        >
            <el-row :gutter="20">
                <el-col :span="12">
                    <div><strong>规则名称：</strong>{{ ruleInfo.name }}</div>
                    <div><strong>节点 ID：</strong>{{ ruleInfo.node_id }}</div>
                    <div>
                        <strong>文件路径：</strong>{{ ruleInfo.file_path }}
                    </div>
                </el-col>
                <el-col :span="12">
                    <div><strong>当前版本：</strong>{{ ruleInfo.version }}</div>
                    <div>
                        <strong>最后更新：</strong
                        >{{ formatDate(ruleInfo.updated_at) }}
                    </div>
                </el-col>
            </el-row>
        </div>

        <!-- 版本列表 + 详情（左侧侧栏，右侧内容） -->
        <div class="version-layout">
            <!-- 左侧侧栏 -->
            <div class="panel sidebar">
                <div class="panel-header">版本历史</div>
                <div class="panel-body">
                    <div class="version-cards">
                        <div
                            v-for="v in versionList"
                            :key="v.id || `v-${v.version}`"
                            class="version-card"
                            :class="{
                                active:
                                    selectedVersion &&
                                    selectedVersion.version === v.version,
                            }"
                            @click="handleVersionSelect(v)"
                        >
                            <div class="version-card-top">
                                <div class="version-title">
                                    v{{ v.version }}
                                </div>
                                <el-tag
                                    v-if="v.is_current"
                                    size="small"
                                    type="success"
                                    >Current</el-tag
                                >
                            </div>
                            <div class="version-comment">
                                {{ v.comment || "Version update" }}
                            </div>
                            <div class="version-time">
                                {{ formatDate(v.created_at) }}
                            </div>
                        </div>
                    </div>
                </div>
            </div>

            <!-- 右侧详情与对比 -->
            <div class="panel main" v-if="selectedVersion">
                <div class="panel-header panel-header-actions">
                    <span>v{{ selectedVersion.version }} 详情</span>
                    <el-button
                        v-if="!selectedVersion.is_current"
                        type="primary"
                        size="small"
                        @click="handleRollback"
                        :loading="rollbackLoading"
                    >
                        回滚到此版本
                    </el-button>
                </div>

                <div class="panel-body">
                    <el-descriptions border :column="2">
                        <el-descriptions-item label="版本号"
                            >v{{
                                selectedVersion.version
                            }}</el-descriptions-item
                        >
                        <el-descriptions-item label="状态">
                            <el-tag
                                v-if="selectedVersion.is_current"
                                type="success"
                                >当前版本</el-tag
                            >
                            <el-tag v-else type="info">历史版本</el-tag>
                        </el-descriptions-item>
                        <el-descriptions-item label="更新时间">{{
                            formatDate(selectedVersion.created_at)
                        }}</el-descriptions-item>
                        <el-descriptions-item label="备注">{{
                            selectedVersion.comment
                        }}</el-descriptions-item>
                    </el-descriptions>

                    <div class="side-by-side">
                        <div class="side">
                            <div class="diff-title diff-title--from">
                                历史版本 v{{ selectedVersion.version }}
                            </div>
                            <div class="code-lines">
                                <div
                                    v-for="(line, idx) in leftLines"
                                    :key="'l-' + idx"
                                    :class="['code-line', lineClass(line.type)]"
                                >
                                    <span class="line-no">{{ idx + 1 }}</span>
                                    <span class="line-text">{{
                                        line.text
                                    }}</span>
                                </div>
                            </div>
                        </div>
                        <div class="side">
                            <div class="diff-title diff-title--to">
                                当前版本 v{{ ruleInfo.version }}
                            </div>
                            <div class="code-lines">
                                <div
                                    v-for="(line, idx) in rightLines"
                                    :key="'r-' + idx"
                                    :class="['code-line', lineClass(line.type)]"
                                >
                                    <span class="line-no">{{ idx + 1 }}</span>
                                    <span class="line-text">{{
                                        line.text
                                    }}</span>
                                </div>
                            </div>
                        </div>
                    </div>
                </div>
            </div>
            <div v-else class="panel main placeholder">
                请选择一个版本查看详情
            </div>
        </div>

        <!-- 回滚确认对话框 -->
        <el-dialog
            v-model="rollbackDialogVisible"
            title="确认回滚"
            width="400px"
        >
            <p>
                确定要回滚到
                <strong>v{{ selectedVersion?.version }}</strong> 吗？
            </p>
            <p style="color: #909399; font-size: 12px; margin-top: 10px">
                当前规则内容将被保存为历史版本，不可恢复。请谨慎操作。
            </p>
            <el-form :model="rollbackForm" style="margin-top: 15px">
                <el-form-item label="回滚说明">
                    <el-input
                        v-model="rollbackForm.comment"
                        type="textarea"
                        :rows="3"
                        placeholder="请输入回滚原因（可选）"
                    />
                </el-form-item>
            </el-form>
            <template #footer>
                <el-button @click="rollbackDialogVisible = false"
                    >取消</el-button
                >
                <el-button
                    type="danger"
                    @click="confirmRollback"
                    :loading="rollbackLoading"
                    >确认回滚</el-button
                >
            </template>
        </el-dialog>
    </div>
</template>

<script setup>
import { ref, reactive, onMounted, computed } from "vue";
import { useRoute, useRouter } from "vue-router";
import axios from "axios";
import { ElMessage } from "element-plus";
import * as Diff from "diff";

const props = defineProps({
    ruleId: Number,
});
const route = useRoute();
const router = useRouter();
// 统一的 ruleId 来源：优先 props，其次路由参数
const effectiveRuleId = computed(() => {
    if (props.ruleId) return props.ruleId;
    const paramId = parseInt(route.params.ruleId, 10);
    return isNaN(paramId) ? null : paramId;
});

const emit = defineEmits(["back"]);

// 数据
const ruleInfo = reactive({
    name: "",
    node_id: "",
    file_path: "",
    version: 0,
    updated_at: "",
});

const versionList = ref([]);
const selectedVersion = ref(null);
const diffLoading = ref(false);
const rollbackLoading = ref(false);
const rollbackDialogVisible = ref(false);

// 左右对比行数组
const leftLines = ref([]);
const rightLines = ref([]);

const rollbackForm = reactive({
    comment: "",
});

// 计算当前版本号
const currentVersionNum = computed(() => ruleInfo.version);

// 获取授权头
const getAuthHeaders = () => {
    return { Authorization: localStorage.getItem("token") };
};

// 格式化日期
const formatDate = (dateStr) => {
    if (!dateStr) return "-";
    const date = new Date(dateStr);
    return date.toLocaleString("zh-CN");
};

// 获取版本历史
const fetchVersions = async () => {
    try {
        const res = await axios.get("/api/rule/versions", {
            params: { id: effectiveRuleId.value },
            headers: getAuthHeaders(),
        });
        if (res.data.data) {
            versionList.value = res.data.data;

            // 设置规则信息
            const current = versionList.value.find((v) => v.is_current);
            if (current) {
                ruleInfo.name = current.name;
                ruleInfo.node_id = current.node_id;
                ruleInfo.file_path = current.file_path;
                ruleInfo.version = current.version;
                ruleInfo.updated_at = current.created_at;
            }

            // 默认选中当前版本
            if (current) {
                selectedVersion.value = current;
            }
            // 如果有历史版本，默认选择最新的历史版本进行对比
            const firstHistory = versionList.value.find((v) => !v.is_current);
            if (firstHistory) {
                selectedVersion.value = firstHistory;
            }

            const currentContent = current?.file_content || "";
            if (selectedVersion.value) {
                buildSideBySide(
                    selectedVersion.value.file_content,
                    currentContent,
                );
            }
        }
    } catch (err) {
        ElMessage.error("获取版本历史失败");
        console.error(err);
    }
};

// 选择版本
const handleVersionSelect = (data) => {
    if (data.version !== undefined) {
        selectedVersion.value = data;
        const current = versionList.value.find((v) => v.is_current);
        buildSideBySide(
            selectedVersion.value.file_content,
            current?.file_content || "",
        );
    }
};

// 构建左右行对比
const buildSideBySide = (oldText, newText) => {
    const parts = Diff.diffLines(oldText || "", newText || "");
    const left = [];
    const right = [];

    for (let i = 0; i < parts.length; i++) {
        const part = parts[i];
        const lines = part.value.split("\n");
        if (lines.length && lines[lines.length - 1] === "") lines.pop();

        if (!part.added && !part.removed) {
            lines.forEach((l) => {
                left.push({ text: l, type: "unchanged" });
                right.push({ text: l, type: "unchanged" });
            });
            continue;
        }

        if (part.removed) {
            const next = parts[i + 1];
            if (next && next.added) {
                const oldLines = lines;
                const newLines = next.value.split("\n");
                if (newLines.length && newLines[newLines.length - 1] === "")
                    newLines.pop();
                const maxLen = Math.max(oldLines.length, newLines.length);
                for (let k = 0; k < maxLen; k++) {
                    left.push({ text: oldLines[k] || "", type: "modified" });
                    right.push({ text: newLines[k] || "", type: "modified" });
                }
                i++;
            } else {
                lines.forEach((l) => {
                    left.push({ text: l, type: "deleted" });
                    right.push({ text: "", type: "empty" });
                });
            }
            continue;
        }

        if (part.added) {
            lines.forEach((l) => {
                left.push({ text: "", type: "empty" });
                right.push({ text: l, type: "added" });
            });
            continue;
        }
    }

    leftLines.value = left;
    rightLines.value = right;
};

// 行样式映射
const lineClass = (type) => {
    switch (type) {
        case "added":
            return "line-added";
        case "deleted":
            return "line-deleted";
        case "modified":
            return "line-modified";
        default:
            return "";
    }
};

// 回滚
const handleRollback = () => {
    rollbackForm.comment = "";
    rollbackDialogVisible.value = true;
};

// 确认回滚
const confirmRollback = async () => {
    rollbackLoading.value = true;
    try {
        // 强制转换为数字，确保JSON中是数字类型而非字符串
        const ruleId = parseInt(effectiveRuleId.value, 10);
        const version = parseInt(selectedVersion.value.version, 10);

        if (isNaN(ruleId) || isNaN(version)) {
            ElMessage.error("无效的规则ID或版本号");
            rollbackLoading.value = false;
            return;
        }

        await axios.post(
            "/api/rule/rollback",
            {
                id: ruleId,
                version: version,
                comment: rollbackForm.comment || `Rollback to v${version}`,
            },
            {
                headers: getAuthHeaders(),
            },
        );
        ElMessage.success("回滚成功");
        rollbackDialogVisible.value = false;
        // 刷新版本列表
        fetchVersions();
    } catch (err) {
        ElMessage.error(err.response?.data?.error || "回滚失败");
    } finally {
        rollbackLoading.value = false;
    }
};

onMounted(() => {
    if (effectiveRuleId.value) {
        fetchVersions();
    } else {
        // 未拿到 ruleId，提示或静默处理
        ElMessage.error("无法识别规则ID，请从列表页进入");
    }
});
</script>

<style scoped>
.rule-version-container {
    padding: 20px;
}

.panel {
    border: none;
    border-radius: 12px;
    background: #fff;
    display: flex;
    flex-direction: column;
    height: 65vh;
    box-shadow: 0 2px 12px rgba(0, 0, 0, 0.08);
}
.panel-header {
    padding: 16px 20px;
    border-bottom: 1px solid #f0f0f0;
    font-weight: 600;
    background: linear-gradient(135deg, #f0f5ff 0%, #e6f0ff 100%);
    color: #1890ff;
    border-radius: 12px 12px 0 0;
}
.panel-header-actions {
    display: flex;
    justify-content: space-between;
    align-items: center;
}
.panel-body {
    padding: 16px;
    overflow: auto;
    flex: 1;
}

.placeholder {
    display: flex;
    align-items: center;
    justify-content: center;
    color: #909399;
}

.version-tree {
    padding: 8px;
}
.version-node {
    display: flex;
    align-items: flex-start;
    gap: 10px;
    padding: 10px 8px;
    border-bottom: 1px solid #f0f0f0;
}
.version-badge {
    background: linear-gradient(135deg, #52c41a 0%, #73d13d 100%);
    color: #fff;
    padding: 3px 10px;
    border-radius: 6px;
    font-size: 12px;
    line-height: 18px;
    font-weight: 500;
}
.version-info {
    flex: 1;
}
.version-title {
    font-weight: 600;
    color: #1890ff;
}
.version-comment {
    font-size: 12px;
    color: #909399;
    margin-top: 4px;
}
.version-time {
    font-size: 11px;
    color: #bfbfbf;
    margin-top: 2px;
}

.version-cards {
    display: flex;
    flex-direction: column;
    gap: 10px;
}
.version-card {
    border: 1px solid #e8e8e8;
    border-radius: 8px;
    padding: 12px 14px;
    cursor: pointer;
    transition: all 0.3s ease;
    background: #fafafa;
}
.version-card:hover {
    border-color: #1890ff;
    box-shadow: 0 4px 12px rgba(24, 144, 255, 0.15);
    transform: translateY(-2px);
}
.version-card.active {
    border-color: #1890ff;
    background: linear-gradient(135deg, #e6f0ff 0%, #f0f7ff 100%);
    box-shadow: 0 4px 12px rgba(24, 144, 255, 0.2);
}
.version-card-top {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 6px;
}

.version-layout {
    display: flex;
    gap: 16px;
}
.sidebar {
    width: 300px;
    flex: 0 0 300px;
}
.main {
    flex: 1;
}
.side-by-side {
    display: flex;
    gap: 16px;
}
.side {
    flex: 1;
}
.diff-title {
    padding: 10px 14px;
    border-radius: 8px;
    margin-bottom: 10px;
    font-weight: 600;
    box-shadow: 0 2px 8px rgba(0, 0, 0, 0.06);
}
.diff-title--from {
    background: linear-gradient(135deg, #fff1f0 0%, #ffccc7 100%);
    color: #cf1322;
}
.diff-title--to {
    background: linear-gradient(135deg, #f6ffed 0%, #d9f7be 100%);
    color: #389e0d;
}
.code-lines {
    border: 1px solid #e8e8e8;
    border-radius: 8px;
    background: #fff;
    max-height: 45vh;
    overflow: auto;
    box-shadow: 0 2px 8px rgba(0, 0, 0, 0.04);
}
.code-line {
    display: flex;
    gap: 12px;
    padding: 4px 12px;
    font-family:
        ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas,
        "Liberation Mono", "Courier New", monospace;
    transition: background 0.2s ease;
}
.code-line:hover {
    background: #fafafa;
}
.line-no {
    width: 40px;
    color: #999;
    text-align: right;
    font-weight: 500;
}
.line-text {
    flex: 1;
    white-space: pre-wrap;
    word-break: break-word;
}
.line-added {
    background: linear-gradient(
        90deg,
        rgba(183, 235, 143, 0.15) 0%,
        transparent 100%
    );
}
.line-deleted {
    background: linear-gradient(
        90deg,
        rgba(255, 189, 189, 0.15) 0%,
        transparent 100%
    );
}
.line-modified {
    background: linear-gradient(
        90deg,
        rgba(145, 213, 255, 0.15) 0%,
        transparent 100%
    );
}

:deep(.el-button--primary) {
    background: linear-gradient(135deg, #1890ff 0%, #096dd9 100%);
    border: none;
    border-radius: 8px;
    transition: all 0.3s ease;
    box-shadow: 0 4px 12px rgba(24, 144, 255, 0.3);
}

:deep(.el-button--primary:hover) {
    transform: translateY(-2px);
    box-shadow: 0 6px 16px rgba(24, 144, 255, 0.4);
}

:deep(.el-button--danger) {
    background: linear-gradient(135deg, #ff4d4f 0%, #cf1322 100%);
    border: none;
    border-radius: 8px;
    transition: all 0.3s ease;
}

:deep(.el-tag) {
    border-radius: 6px;
    padding: 4px 12px;
    font-weight: 500;
}
</style>
