<template>
    <div class="editor-section">
        <div class="section-title">
            <span>规则内容编辑</span>
            <div class="validation-indicator" v-if="validationResult">
                <el-icon v-if="validationResult.success" class="success-icon">
                    <CircleCheck />
                </el-icon>
                <el-icon v-else class="error-icon">
                    <CircleClose />
                </el-icon>
            </div>
        </div>

        <textarea :value="modelValue" class="yaml-editor" placeholder="输入 YAML 格式的 Prometheus 告警规则..."
            @input="handleInput"></textarea>

        <div class="validation-panel" v-if="validationResult">
            <div class="validation-status" :class="
                validationResult.success
                    ? 'success'
                    : 'error'
            ">
                <el-icon v-if="validationResult.success">
                    <CircleCheck />
                </el-icon>
                <el-icon v-else>
                    <CircleClose />
                </el-icon>
                <span>{{
                    validationResult.success
                    ? "✅ 语法验证通过"
                    : "❌ 语法验证失败"
                }}</span>
            </div>
            <pre v-if="validationResult.output" class="validation-output">{{ validationResult.output }}</pre>
        </div>

        <div class="validation-loading" v-if="isValidating">
            <el-icon class="is-loading">
                <Loading />
            </el-icon>
            <span>验证中...</span>
        </div>
    </div>
</template>

<script setup>
import { defineProps, defineEmits } from 'vue';
import { CircleCheck, CircleClose, Loading } from "@element-plus/icons-vue";

const props = defineProps({
    modelValue: {
        type: String,
        default: "",
    },
    validationResult: {
        type: Object,
        default: null,
    },
    isValidating: {
        type: Boolean,
        default: false,
    },
});

const emit = defineEmits(['update:modelValue', 'change']);

const handleInput = (event) => {
    emit('update:modelValue', event.target.value);
    emit('change', event.target.value);
};
</script>

<style scoped>
.editor-section {
    display: flex;
    flex-direction: column;
    gap: 12px;
    min-height: 550px;
    flex: 1;
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

.validation-indicator {
    font-size: 20px;
}

.validation-indicator .success-icon {
    color: #67c23a;
}

.validation-indicator .error-icon {
    color: #f56c6c;
}

.yaml-editor {
    width: 100%;
    min-height: 450px;
    font-family: "Monaco", "Menlo", "Consolas", monospace;
    font-size: 14px;
    line-height: 1.6;
    padding: 16px;
    border: 1px solid #dcdfe6;
    border-radius: 8px;
    resize: vertical;
    background: #f8f9fa;
    transition: all 0.3s;
    box-sizing: border-box;
}

.yaml-editor:focus {
    outline: none;
    border-color: #1890ff;
    box-shadow: 0 0 0 2px rgba(24, 144, 255, 0.1);
}

.validation-panel {
    margin-top: 12px;
    padding: 12px;
    border-radius: 8px;
    background: #fff;
    border: 1px solid #e4e7ed;
}

.validation-status {
    display: flex;
    align-items: center;
    gap: 8px;
    font-weight: 600;
    margin-bottom: 8px;
}

.validation-status.success {
    color: #67c23a;
}

.validation-status.error {
    color: #f56c6c;
}

.validation-output {
    font-family: "Monaco", "Menlo", "Consolas", monospace;
    font-size: 12px;
    background: #f8f9fa;
    padding: 12px;
    border-radius: 6px;
    margin: 0;
    white-space: pre-wrap;
    word-wrap: break-word;
    max-height: 200px;
    overflow-y: auto;
}

.validation-loading {
    display: flex;
    align-items: center;
    gap: 8px;
    margin-top: 12px;
    color: #909399;
    font-size: 14px;
}

@media (max-width: 1600px) {
    .yaml-editor {
        min-height: 400px;
    }

    .editor-section {
        min-height: auto;
    }
}
</style>
