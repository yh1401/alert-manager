<template>
    <div class="audit-log-container">
        <el-card shadow="hover">
            <template #header>
                <div class="card-header">
                    <h2>
                        <el-icon>
                            <Document />
                        </el-icon>
                        审计日志
                    </h2>
                    <el-text type="info">管理员可查看所有用户对规则和节点的操作记录</el-text>
                </div>
            </template>

            <!-- 筛选区域 -->
            <AuditLogFilter :filter="filterForm" @search="handleSearch" @reset="handleReset" />

            <!-- 统计卡片 -->
            <AuditLogStats :stats="stats" />

            <!-- 审计日志表格 -->
            <el-table :data="auditLogs" v-loading="loading" stripe border style="width: 100%; margin-top: 20px"
                :default-sort="{ prop: 'created_at', order: 'descending' }">
                <el-table-column prop="id" label="ID" width="80" align="center" />

                <el-table-column prop="username" label="操作人" width="120" align="center">
                    <template #default="{ row }">
                        <el-tag v-if="row.user_id === 0" type="info" size="small">系统</el-tag>
                        <el-tag v-else type="success" size="small">{{
                            row.username
                        }}</el-tag>
                    </template>
                </el-table-column>

                <el-table-column prop="resource_type" label="资源类型" width="100" align="center">
                    <template #default="{ row }">
                        <el-tag v-if="row.resource_type === 'rule'" type="primary" size="small">规则</el-tag>
                        <el-tag v-else type="warning" size="small">节点</el-tag>
                    </template>
                </el-table-column>

                <el-table-column prop="resource_name" label="资源名称" width="180" show-overflow-tooltip />

                <el-table-column prop="action" label="操作类型" width="100" align="center">
                    <template #default="{ row }">
                        <el-tag v-if="row.action === 'create'" type="success" size="small">创建</el-tag>
                        <el-tag v-else-if="row.action === 'update'" type="primary" size="small">更新</el-tag>
                        <el-tag v-else-if="row.action === 'delete'" type="danger" size="small">删除</el-tag>
                        <el-tag v-else-if="row.action === 'rollback'" type="warning" size="small">回滚</el-tag>
                        <el-tag v-else-if="row.action === 'manual_sync'" type="primary" size="small">同步</el-tag>
                        <el-tag v-else type="info" size="small">{{
                            row.action
                        }}</el-tag>
                    </template>
                </el-table-column>

                <el-table-column prop="description" label="操作描述" min-width="250" show-overflow-tooltip />

                <el-table-column prop="ip_address" label="IP地址" width="140" align="center" />

                <el-table-column prop="created_at" label="操作时间" width="180" align="center">
                    <template #default="{ row }">
                        {{ formatDateTime(row.created_at) }}
                    </template>
                </el-table-column>

                <el-table-column label="操作" width="220" align="center" fixed="right">
                    <template #default="{ row }">
                        <el-button type="primary" size="small" @click="viewDetail(row)" :icon="View">
                            查看详情
                        </el-button>
                        <el-button v-if="
                            row.resource_type === 'rule' &&
                            row.action === 'delete'
                        " type="warning" size="small" :icon="RefreshLeft" :loading="restoring"
                            @click="restoreRule(row)">
                            恢复
                        </el-button>
                    </template>
                </el-table-column>
            </el-table>

            <!-- 分页 -->
            <el-pagination v-if="pagination.total > 0" style="margin-top: 20px; justify-content: center"
                :current-page="pagination.page" :page-size="pagination.page_size" :page-sizes="[10, 20, 50, 100]"
                :total="pagination.total" layout="total, sizes, prev, pager, next, jumper"
                @size-change="handleSizeChange" @current-change="handleCurrentChange" />
        </el-card>

        <!-- 详情对话框 -->
        <AuditLogDetailDialog :visible="detailDialogVisible" :detail="currentDetail" :restoring="restoring"
            @close="detailDialogVisible = false" @restore="restoreRule" />
    </div>
</template>

<script setup>
import { ref, reactive, onMounted } from "vue";
import { ElMessage, ElMessageBox } from "element-plus";
import {
    Document,
    View,
    RefreshLeft,
} from "@element-plus/icons-vue";
import axios from "axios";
import AuditLogFilter from '../components/audit/AuditLogFilter.vue';
import AuditLogStats from '../components/audit/AuditLogStats.vue';
import AuditLogDetailDialog from '../components/audit/AuditLogDetailDialog.vue';

const loading = ref(false);
const auditLogs = ref([]);
const stats = ref(null);
const detailDialogVisible = ref(false);
const currentDetail = ref(null);
const restoring = ref(false);

const filterForm = reactive({
    resource_type: "",
    action: "",
    username: "",
    start_date: "",
    end_date: "",
});

const pagination = reactive({
    page: 1,
    page_size: 20,
    total: 0,
    total_page: 0,
});

// 加载审计日志列表
const loadAuditLogs = async () => {
    loading.value = true;
    try {
        const params = {
            page: pagination.page,
            page_size: pagination.page_size,
            ...filterForm,
        };

        // 清除空参数
        Object.keys(params).forEach((key) => {
            if (
                params[key] === "" ||
                params[key] === null ||
                params[key] === undefined
            ) {
                delete params[key];
            }
        });

        const response = await axios.get("/api/admin/audit/logs", { params });
        auditLogs.value = response.data.data || [];

        if (response.data.pagination) {
            pagination.page = response.data.pagination.page;
            pagination.page_size = response.data.pagination.page_size;
            pagination.total = response.data.pagination.total;
            pagination.total_page = response.data.pagination.total_page;
        }
    } catch (error) {
        console.error("加载审计日志失败:", error);
        ElMessage.error(error.response?.data?.error || "加载审计日志失败");
    } finally {
        loading.value = false;
    }
};

// 加载统计数据
const loadStats = async () => {
    try {
        const response = await axios.get("/api/admin/audit/stats");
        stats.value = response.data.data;
    } catch (error) {
        console.error("加载统计数据失败:", error);
    }
};

// 查看详情
const viewDetail = async (row) => {
    try {
        const response = await axios.get(`/api/admin/audit/logs/${row.id}`);
        currentDetail.value = response.data.data;
        detailDialogVisible.value = true;
    } catch (error) {
        console.error("加载日志详情失败:", error);
        ElMessage.error(error.response?.data?.error || "加载日志详情失败");
    }
};

const restoreRule = async (row) => {
    // If called from dialog, row is passed directly. 
    // If called from table, row is the row object.
    // Ensure we have a valid object.
    const target = row || currentDetail.value;
    if (!target) return;

    try {
        await ElMessageBox.confirm(
            `确认从审计记录恢复规则「${target.resource_name || target.resource_id}」吗？`,
            "确认恢复",
            { type: "warning" },
        );
    } catch {
        return;
    }

    if (restoring.value) return;
    restoring.value = true;
    try {
        await axios.post("/api/admin/audit/rules/restore", {
            audit_id: target.id,
        });
        ElMessage.success("恢复成功");
        loadAuditLogs();
        if (detailDialogVisible.value) {
            detailDialogVisible.value = false;
        }
    } catch (error) {
        console.error("恢复规则失败:", error);
        ElMessage.error(error.response?.data?.error || "恢复失败");
    } finally {
        restoring.value = false;
    }
};

// 搜索
const handleSearch = () => {
    pagination.page = 1;
    loadAuditLogs();
};

// 重置
const handleReset = () => {
    filterForm.resource_type = "";
    filterForm.action = "";
    filterForm.username = "";
    filterForm.start_date = "";
    filterForm.end_date = "";
    pagination.page = 1;
    loadAuditLogs();
};

// 分页
const handleSizeChange = (val) => {
    pagination.page_size = val;
    pagination.page = 1;
    loadAuditLogs();
};

const handleCurrentChange = (val) => {
    pagination.page = val;
    loadAuditLogs();
};

// 格式化日期时间
const formatDateTime = (dateString) => {
    if (!dateString) return "-";
    const date = new Date(dateString);
    return date.toLocaleString("zh-CN", {
        year: "numeric",
        month: "2-digit",
        day: "2-digit",
        hour: "2-digit",
        minute: "2-digit",
        second: "2-digit",
        hour12: false,
    });
};

onMounted(() => {
    loadAuditLogs();
    loadStats();
});
</script>

<style scoped>
.audit-log-container {
    padding: 20px;
}

.card-header {
    display: flex;
    flex-direction: column;
    gap: 8px;
}

.card-header h2 {
    margin: 0;
    display: flex;
    align-items: center;
    gap: 8px;
    font-size: 20px;
}

:deep(.el-table) {
    font-size: 14px;
}

:deep(.el-pagination) {
    display: flex;
    justify-content: center;
}
</style>