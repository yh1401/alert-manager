<template>
    <el-form :inline="true" :model="filter" class="filter-form">
        <el-form-item label="资源类型">
            <el-select v-model="filter.resource_type" placeholder="全部" clearable style="width: 120px">
                <el-option label="规则" value="rule" />
                <el-option label="节点" value="node" />
            </el-select>
        </el-form-item>

        <el-form-item label="操作类型">
            <el-select v-model="filter.action" placeholder="全部" clearable style="width: 120px">
                <el-option label="创建" value="create" />
                <el-option label="更新" value="update" />
                <el-option label="删除" value="delete" />
                <el-option label="回滚" value="rollback" />
                <el-option label="同步" value="manual_sync" />
            </el-select>
        </el-form-item>

        <el-form-item label="操作人">
            <el-input v-model="filter.username" placeholder="用户名" clearable style="width: 150px" />
        </el-form-item>

        <el-form-item label="开始日期">
            <el-date-picker v-model="filter.start_date" type="date" placeholder="选择日期" format="YYYY-MM-DD"
                value-format="YYYY-MM-DD" style="width: 150px" />
        </el-form-item>

        <el-form-item label="结束日期">
            <el-date-picker v-model="filter.end_date" type="date" placeholder="选择日期" format="YYYY-MM-DD"
                value-format="YYYY-MM-DD" style="width: 150px" />
        </el-form-item>

        <el-form-item>
            <el-button type="primary" @click="$emit('search')" :icon="Search">查询</el-button>
            <el-button @click="$emit('reset')" :icon="Refresh">重置</el-button>
        </el-form-item>
    </el-form>
</template>

<script setup>
import { defineProps, defineEmits } from 'vue';
import { Search, Refresh } from '@element-plus/icons-vue';

defineProps({
    filter: {
        type: Object,
        required: true,
    },
});

defineEmits(['search', 'reset']);
</script>

<style scoped>
.filter-form {
    margin-top: 20px;
}
</style>
