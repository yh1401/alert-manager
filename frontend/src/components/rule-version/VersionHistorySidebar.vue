<template>
    <div class="panel sidebar">
        <div class="panel-header">版本历史</div>
        <div class="panel-body">
            <div class="version-cards">
                <div v-for="v in versions" :key="v.id || `v-${v.version}`" class="version-card" :class="{
                    active:
                        selectedVersion &&
                        selectedVersion.version === v.version,
                }" @click="emit('select', v)">
                    <div class="version-card-top">
                        <div class="version-title">
                            v{{ v.version }}
                        </div>
                        <el-tag v-if="v.is_current" size="small" type="success">Current</el-tag>
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
</template>

<script setup>
import { defineProps, defineEmits } from 'vue';

defineProps({
    versions: {
        type: Array,
        default: () => [],
    },
    selectedVersion: {
        type: Object,
        default: null,
    },
});

const emit = defineEmits(['select']);

const formatDate = (dateStr) => {
    if (!dateStr) return "-";
    const date = new Date(dateStr);
    return date.toLocaleString("zh-CN");
};
</script>

<style scoped>
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

.panel-body {
    padding: 16px;
    overflow: auto;
    flex: 1;
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
</style>
