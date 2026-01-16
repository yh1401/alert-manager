<template>
    <div class="tab-content" v-if="user">
        <div style="margin-bottom: 16px">
            <el-tag type="success">用户: {{ user.username }}</el-tag>
            <el-button type="primary" @click="emit('refresh')" icon="Refresh" style="margin-left: 10px">刷新权限</el-button>
        </div>
        <el-table :data="permissions" stripe border class="permission-table">
            <el-table-column prop="id" label="ID" width="80" />
            <el-table-column prop="resource_type" label="资源类型" width="120">
                <template #default="{ row }">
                    <el-tag :type="row.resource_type === 'rule'
                        ? 'primary'
                        : 'success'
                        ">{{ row.resource_type }}</el-tag>
                </template>
            </el-table-column>
            <el-table-column prop="resource_id" label="资源ID" width="100" />
            <el-table-column prop="action" label="权限">
                <template #default="{ row }">
                    <el-tag :type="row.action === 'write'
                        ? 'warning'
                        : 'info'
                        ">{{ row.action }}</el-tag>
                </template>
            </el-table-column>
            <el-table-column prop="created_at" label="创建时间" width="180" />
            <el-table-column label="操作" width="120">
                <template #default="{ row }">
                    <el-button size="small" type="danger" @click="emit('remove', row)">移除</el-button>
                </template>
            </el-table-column>
        </el-table>
    </div>
</template>

<script setup>
import { defineProps, defineEmits } from 'vue';
import { Refresh } from '@element-plus/icons-vue';

defineProps({
    user: {
        type: Object,
        default: null,
    },
    permissions: {
        type: Array,
        default: () => [],
    },
});

const emit = defineEmits(['refresh', 'remove']);
</script>

<style scoped>
.tab-content {
    padding: 20px;
    height: 100%;
    width: 100%;
    display: flex;
    flex-direction: column;
    overflow: auto;
}

.permission-table {
    flex: 1;
    width: 100%;
    overflow: auto;
    margin-bottom: 0;
    min-height: 200px;
}
</style>
