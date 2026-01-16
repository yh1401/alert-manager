<template>
    <div class="stats-cards" v-if="stats">
        <el-row :gutter="16">
            <el-col style="width: 20%; max-width: 20%; flex: 0 0 20%" v-for="(stat, index) in statsDisplay" :key="index">
                <el-card shadow="hover" class="stat-card">
                    <div class="stat-content">
                        <el-icon :size="32" :color="stat.color">
                            <component :is="stat.icon" />
                        </el-icon>
                        <div class="stat-info">
                            <div class="stat-value">
                                {{ stat.value }}
                            </div>
                            <div class="stat-label">
                                {{ stat.label }}
                            </div>
                        </div>
                    </div>
                </el-card>
            </el-col>
        </el-row>
    </div>
</template>

<script setup>
import { defineProps, computed } from 'vue';
import { DataAnalysis, Operation } from '@element-plus/icons-vue';

const props = defineProps({
    stats: {
        type: Object,
        default: () => null,
    },
});

const statsDisplay = computed(() => {
    if (!props.stats) return [];

    const actionMap = {
        create: { label: "创建操作", icon: "Plus", color: "#67C23A" },
        update: { label: "更新操作", icon: "Edit", color: "#409EFF" },
        delete: { label: "删除操作", icon: "Delete", color: "#F56C6C" },
        rollback: { label: "回滚操作", icon: "RefreshLeft", color: "#E6A23C" },
        manual_sync: { label: "同步操作", icon: "Sync", color: "#409EFF" },
    };

    const result = [];

    // 操作类型统计
    if (props.stats.action_stats && props.stats.action_stats.length > 0) {
        props.stats.action_stats.forEach((stat) => {
            const config = actionMap[stat.Action] || {
                label: stat.Action,
                icon: "Operation",
                color: "#909399",
            };
            result.push({
                label: config.label,
                value: stat.Count,
                icon: Operation,
                color: config.color,
            });
        });
    }

    // 填充空位（如果少于5个）
    while (result.length < 5) {
        result.push({
            label: "总操作数",
            value:
                props.stats.action_stats?.reduce(
                    (sum, s) => sum + s.Count,
                    0,
                ) || 0,
            icon: DataAnalysis,
            color: "#409EFF",
        });
        break;
    }

    return result.slice(0, 5);
});
</script>

<style scoped>
.stats-cards {
    margin-top: 20px;
}

.stat-card {
    cursor: pointer;
    transition: all 0.3s;
}

.stat-card:hover {
    transform: translateY(-4px);
}

.stat-content {
    display: flex;
    align-items: center;
    gap: 16px;
}

.stat-info {
    flex: 1;
}

.stat-value {
    font-size: 28px;
    font-weight: bold;
    color: #303133;
}

.stat-label {
    font-size: 14px;
    color: #909399;
    margin-top: 4px;
}
</style>
