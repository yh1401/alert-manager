<template>
    <div class="tab-content">
        <el-button type="primary" @click="emit('refresh')" icon="Refresh" style="margin-bottom: 16px">刷新</el-button>
        <el-table :data="users" stripe border class="permission-table">
            <el-table-column prop="id" label="ID" width="80" />
            <el-table-column prop="username" label="用户名" />
            <el-table-column prop="role" label="角色">
                <template #default="{ row }">
                    <el-tag :type="row.role === 'admin'
                        ? 'danger'
                        : 'info'
                        ">{{ row.role }}</el-tag>
                </template>
            </el-table-column>
            <el-table-column label="操作" width="300">
                <template #default="{ row }">
                    <el-button size="small" @click="emit('view-permissions', row)">查看权限</el-button>
                    <el-button size="small" type="warning" @click="emit('change-role', row)">修改角色</el-button>
                    <el-button size="small" type="primary" @click="emit('batch-auth', row)">批量授权</el-button>
                </template>
            </el-table-column>
        </el-table>
    </div>
</template>

<script setup>
import { defineProps, defineEmits } from 'vue';
import { Refresh } from '@element-plus/icons-vue';

defineProps({
    users: {
        type: Array,
        default: () => [],
    },
});

const emit = defineEmits(['refresh', 'view-permissions', 'change-role', 'batch-auth']);
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
