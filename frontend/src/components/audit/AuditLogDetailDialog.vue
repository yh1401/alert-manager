<template>
    <el-dialog :model-value="visible" title="审计日志详情" width="80%" :close-on-click-modal="false" @update:model-value="emit('close')">
        <div v-if="detail" class="detail-content">
            <el-descriptions :column="2" border>
                <el-descriptions-item label="日志ID">{{
                    detail.id
                }}</el-descriptions-item>
                <el-descriptions-item label="操作人">
                    <el-tag v-if="detail.user_id === 0" type="info" size="small">系统</el-tag>
                    <span v-else>{{ detail.username }} (ID:
                        {{ detail.user_id }})</span>
                </el-descriptions-item>
                <el-descriptions-item label="资源类型">
                    <el-tag v-if="detail.resource_type === 'rule'" type="primary" size="small">规则</el-tag>
                    <el-tag v-else type="warning" size="small">节点</el-tag>
                </el-descriptions-item>
                <el-descriptions-item label="资源名称">{{
                    detail.resource_name
                }}</el-descriptions-item>
                <el-descriptions-item label="操作类型">
                    <el-tag v-if="detail.action === 'create'" type="success" size="small">创建</el-tag>
                    <el-tag v-else-if="detail.action === 'update'" type="primary" size="small">更新</el-tag>
                    <el-tag v-else-if="detail.action === 'delete'" type="danger" size="small">删除</el-tag>
                    <el-tag v-else-if="detail.action === 'rollback'" type="warning" size="small">回滚</el-tag>
                    <el-tag v-else-if="detail.action === 'manual_sync'" type="primary" size="small">同步</el-tag>
                    <el-tag v-else type="info" size="small">{{
                        detail.action
                    }}</el-tag>
                </el-descriptions-item>
                <el-descriptions-item label="IP地址">{{
                    detail.ip_address
                }}</el-descriptions-item>
                <el-descriptions-item label="操作时间" :span="2">
                    {{ formatDateTime(detail.created_at) }}
                </el-descriptions-item>
                <el-descriptions-item label="操作描述" :span="2">
                    {{ detail.description }}
                </el-descriptions-item>
            </el-descriptions>

            <!-- 规则内容差异（行级彩色对比） -->
            <div v-if="isRuleWithFileDiff" class="change-comparison" style="margin-top: 20px">
                <h3>规则内容差异</h3>
                <div class="diff-wrapper">
                    <div class="diff-side">
                        <div class="diff-title diff-title--from">
                            变更前
                        </div>
                        <div class="code-lines">
                            <div v-for="(line, idx) in diffLeftLines" :key="'dl-' + idx"
                                :class="['code-line', lineClass(line.type)]">
                                <span class="line-no">{{ idx + 1 }}</span>
                                <span class="line-text">{{
                                    line.text
                                }}</span>
                            </div>
                        </div>
                    </div>
                    <div class="diff-side">
                        <div class="diff-title diff-title--to">变更后</div>
                        <div class="code-lines">
                            <div v-for="(line, idx) in diffRightLines" :key="'dr-' + idx"
                                :class="['code-line', lineClass(line.type)]">
                                <span class="line-no">{{ idx + 1 }}</span>
                                <span class="line-text">{{
                                    line.text
                                }}</span>
                            </div>
                        </div>
                    </div>
                </div>
            </div>

            <!-- JSON 级别对比（非规则或无文件内容时） -->
            <div v-else-if="detail.action !== 'create'" class="change-comparison" style="margin-top: 20px">
                <h3>变更前后对比</h3>
                <el-row :gutter="20">
                    <el-col :span="12">
                        <el-card shadow="hover">
                            <template #header>
                                <div style="
                                            display: flex;
                                            align-items: center;
                                            gap: 8px;
                                        ">
                                    <el-icon color="#909399">
                                        <Back />
                                    </el-icon>
                                    <span>变更前</span>
                                </div>
                            </template>
                            <pre class="json-content">{{
                                formatJSON(detail.old_value)
                            }}</pre>
                        </el-card>
                    </el-col>
                    <el-col :span="12">
                        <el-card shadow="hover">
                            <template #header>
                                <div style="
                                            display: flex;
                                            align-items: center;
                                            gap: 8px;
                                        ">
                                    <el-icon color="#67C23A">
                                        <Right />
                                    </el-icon>
                                    <span>变更后</span>
                                </div>
                            </template>
                            <pre class="json-content">{{
                                formatJSON(detail.new_value)
                            }}</pre>
                        </el-card>
                    </el-col>
                </el-row>
            </div>

            <!-- 仅创建操作显示新值 -->
            <div v-else class="change-comparison" style="margin-top: 20px">
                <h3>创建内容</h3>
                <el-card shadow="hover">
                    <pre class="json-content">{{
                        formatJSON(detail.new_value)
                    }}</pre>
                </el-card>
            </div>
        </div>

        <template #footer>
            <el-button v-if="detail &&
                detail.resource_type === 'rule' &&
                detail.action === 'delete'
                " type="warning" :loading="restoring" :icon="RefreshLeft" @click="emit('restore', detail)">
                从此记录恢复
            </el-button>
            <el-button @click="emit('close')">关闭</el-button>
        </template>
    </el-dialog>
</template>

<script setup>
import { defineProps, defineEmits, computed } from 'vue';
import { Back, Right, RefreshLeft } from '@element-plus/icons-vue';
import * as Diff from 'diff';

const props = defineProps({
    visible: Boolean,
    detail: Object,
    restoring: Boolean,
});

const emit = defineEmits(['close', 'restore']);

const formatDateTime = (dateString) => {
    if (!dateString) return '-';
    const date = new Date(dateString);
    return date.toLocaleString('zh-CN', {
        year: 'numeric',
        month: '2-digit',
        day: '2-digit',
        hour: '2-digit',
        minute: '2-digit',
        second: '2-digit',
        hour12: false,
    });
};

const formatJSON = (jsonString) => {
    if (!jsonString) return '无数据';
    try {
        const obj =
            typeof jsonString === 'string'
                ? JSON.parse(jsonString)
                : jsonString;
        return JSON.stringify(obj, null, 2);
    } catch (e) {
        return jsonString;
    }
};

const getFileContent = (value) => {
    if (!value) return null;
    if (typeof value === 'string') {
        try {
            const obj = JSON.parse(value);
            return obj?.file_content || obj?.content || null;
        } catch (e) {
            return null;
        }
    }
    if (typeof value === 'object') {
        return value.file_content || value.content || null;
    }
    return null;
};

const isRuleWithFileDiff = computed(() => {
    const d = props.detail;
    if (!d || d.resource_type !== 'rule') return false;
    const oldContent = getFileContent(d.old_value);
    const newContent = getFileContent(d.new_value);
    return !!(oldContent || newContent);
});

const buildSideBySide = (oldText, newText) => {
    const parts = Diff.diffLines(oldText || '', newText || '');
    const left = [];
    const right = [];

    for (let i = 0; i < parts.length; i++) {
        const part = parts[i];
        const lines = part.value.split('\n');
        if (lines.length && lines[lines.length - 1] === '') lines.pop();

        if (!part.added && !part.removed) {
            lines.forEach((l) => {
                left.push({ text: l, type: 'unchanged' });
                right.push({ text: l, type: 'unchanged' });
            });
            continue;
        }

        if (part.removed) {
            const next = parts[i + 1];
            if (next && next.added) {
                const oldLines = lines;
                const newLines = next.value.split('\n');
                if (newLines.length && newLines[newLines.length - 1] === '')
                    newLines.pop();
                const maxLen = Math.max(oldLines.length, newLines.length);
                for (let k = 0; k < maxLen; k++) {
                    left.push({ text: oldLines[k] || '', type: 'modified' });
                    right.push({ text: newLines[k] || '', type: 'modified' });
                }
                i++;
            } else {
                lines.forEach((l) => {
                    left.push({ text: l, type: 'deleted' });
                    right.push({ text: '', type: 'empty' });
                });
            }
            continue;
        }

        if (part.added) {
            lines.forEach((l) => {
                left.push({ text: '', type: 'empty' });
                right.push({ text: l, type: 'added' });
            });
        }
    }

    return { left, right };
};

const diffLeftLines = computed(() => {
    if (!isRuleWithFileDiff.value) return [];
    const d = props.detail;
    return buildSideBySide(
        getFileContent(d?.old_value),
        getFileContent(d?.new_value),
    ).left;
});

const diffRightLines = computed(() => {
    if (!isRuleWithFileDiff.value) return [];
    const d = props.detail;
    return buildSideBySide(
        getFileContent(d?.old_value),
        getFileContent(d?.new_value),
    ).right;
});

const lineClass = (type) => {
    switch (type) {
        case 'added':
            return 'line-added';
        case 'deleted':
            return 'line-deleted';
        case 'modified':
            return 'line-modified';
        default:
            return '';
    }
};
</script>

<style scoped>
.detail-content {
    max-height: 70vh;
    overflow-y: auto;
}

.change-comparison h3 {
    margin: 0 0 16px 0;
    font-size: 16px;
    color: #303133;
}

.json-content {
    max-height: 400px;
    overflow-y: auto;
    background-color: #f5f7fa;
    padding: 12px;
    border-radius: 4px;
    font-size: 13px;
    line-height: 1.6;
    margin: 0;
    white-space: pre-wrap;
    word-wrap: break-word;
}

.diff-wrapper {
    display: flex;
    gap: 16px;
    flex-wrap: wrap;
}

.diff-side {
    flex: 1;
    min-width: 320px;
}

.code-lines {
    border: 1px solid #e8e8e8;
    border-radius: 8px;
    background: #fff;
    max-height: 420px;
    overflow: auto;
    box-shadow: 0 2px 8px rgba(0, 0, 0, 0.04);
}

.code-line {
    display: flex;
    gap: 12px;
    padding: 4px 12px;
    font-family: ui-monospace,
    SFMono-Regular,
    Menlo,
    Monaco,
    Consolas,
    'Liberation Mono',
    'Courier New',
    monospace;
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
    background: linear-gradient(90deg,
            rgba(183, 235, 143, 0.15) 0%,
            transparent 100%);
}

.line-deleted {
    background: linear-gradient(90deg,
            rgba(255, 189, 189, 0.15) 0%,
            transparent 100%);
}

.line-modified {
    background: linear-gradient(90deg,
            rgba(145, 213, 255, 0.15) 0%,
            transparent 100%);
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
</style>
