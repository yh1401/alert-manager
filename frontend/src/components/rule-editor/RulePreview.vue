<template>
    <div class="preview-section">
        <div class="section-title">规则预览</div>
        <div class="preview-panel">
            <div v-if="groups.length > 0" class="preview-content">
                <div v-for="(group, idx) in groups" :key="idx" class="rule-group-preview">
                    <h5>
                        {{ group.name || "未命名分组" }}
                    </h5>
                    <div v-for="(rule, rIdx) in group.rules" :key="rIdx" class="rule-item-preview">
                        <div class="rule-alert-name">
                            <el-tag type="warning" size="small">{{ rule.alert }}</el-tag>
                        </div>
                        <div class="rule-expr">
                            <strong>条件:</strong><br /><code>{{
                                rule.expr
                            }}</code>
                        </div>
                        <div class="rule-for" v-if="rule.for">
                            <strong>持续:</strong>
                            {{ rule.for }}
                        </div>
                        <div class="rule-labels" v-if="rule.labels">
                            <strong>标签:</strong><br />
                            <el-tag v-for="(
                                val, key
                            ) in rule.labels" :key="key" size="small" style="
                                margin-top: 4px;
                                margin-right: 4px;
                            ">{{ key }}:
                                {{ val }}</el-tag>
                        </div>
                    </div>
                </div>
            </div>
            <el-empty v-else description="暂无规则预览" :image-size="60"></el-empty>
        </div>
    </div>
</template>

<script setup>
import { defineProps } from 'vue';

defineProps({
    groups: {
        type: Array,
        default: () => [],
    },
});
</script>

<style scoped>
.preview-section {
    display: flex;
    flex-direction: column;
    gap: 12px;
    min-height: 550px;
    position: relative;
    flex: 1; /* Ensure it takes available space */
}

.section-title {
    display: flex;
    justify-content: space-between;
    align-items: center;
    font-size: 16px;
    font-weight: 600;
    color: #303133;
    padding: 0 0 8px 0;
    border-bottom: 2px solid #1890ff;
}

.preview-panel {
    padding: 16px;
    background: #f8f9fa;
    border-radius: 8px;
    max-height: 500px;
    overflow-y: auto;
    border: 1px solid #e4e7ed;
    flex: 1;
}

.preview-content {
    display: flex;
    flex-direction: column;
    gap: 12px;
}

.rule-group-preview {
    padding: 12px;
    background: white;
    border-radius: 8px;
    box-shadow: 0 2px 8px rgba(0, 0, 0, 0.05);
}

.rule-group-preview h5 {
    margin: 0 0 10px 0;
    color: #1890ff;
    font-size: 14px;
    font-weight: 600;
}

.rule-item-preview {
    padding: 10px;
    margin-bottom: 8px;
    background: #fafafa;
    border-left: 3px solid #1890ff;
    border-radius: 4px;
    font-size: 13px;
}

.rule-item-preview:last-child {
    margin-bottom: 0;
}

.rule-alert-name {
    margin-bottom: 6px;
}

.rule-expr {
    margin-bottom: 6px;
    font-size: 13px;
    line-height: 1.4;
}

.rule-expr code {
    background: #e6f7ff;
    padding: 2px 4px;
    border-radius: 3px;
    font-family: "Monaco", "Menlo", "Consolas", monospace;
    font-size: 12px;
    word-break: break-all;
}

.rule-for,
.rule-labels {
    margin-top: 6px;
    font-size: 13px;
    color: #606266;
}

@media (max-width: 1600px) {
    .preview-section {
        position: static;
        min-height: auto;
    }

    .preview-panel {
        max-height: 400px;
    }
}
</style>
