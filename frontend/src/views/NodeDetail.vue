<template>
    <div class="detail-container">
        <el-card class="box-card">
            <template #header>
                <div class="card-header">
                    <span>节点详情</span>
                    <div>
                        <el-button
                            type="primary"
                            @click="manualSync"
                            :loading="manualLoading"
                            size="small"
                            >手动拉取 & 重载</el-button
                        >
                        <el-button
                            type="primary"
                            @click="fetchDetail"
                            :loading="loading"
                            size="small"
                            style="margin-left: 8px"
                            >刷新</el-button
                        >
                        <el-button
                            type="danger"
                            @click="deleteNode"
                            size="small"
                            style="margin-left: 8px"
                            v-if="canWrite"
                            >删除节点</el-button
                        >
                        <el-button
                            @click="goBack"
                            size="small"
                            style="margin-left: 8px"
                            >返回</el-button
                        >
                    </div>
                </div>
            </template>
            <el-descriptions :column="2" border>
                <el-descriptions-item label="ID">{{
                    detail.id
                }}</el-descriptions-item>
                <el-descriptions-item label="名称">{{
                    detail.name
                }}</el-descriptions-item>
                <el-descriptions-item label="IP 地址">{{
                    detail.ip_address
                }}</el-descriptions-item>
                <el-descriptions-item label="节点 ID">
                    {{ detail.id }}
                </el-descriptions-item>
                <el-descriptions-item label="状态">
                    <el-tag
                        :type="
                            detail.status === 'online' ? 'success' : 'danger'
                        "
                    >
                        {{ detail.status === "online" ? "在线" : "离线" }}
                    </el-tag>
                </el-descriptions-item>
                <el-descriptions-item label="最后心跳">{{
                    formatDate(detail.last_heartbeat)
                }}</el-descriptions-item>
                <el-descriptions-item label="创建时间">{{
                    formatDate(detail.created_at)
                }}</el-descriptions-item>
                <el-descriptions-item label="更新时间">{{
                    formatDate(detail.updated_at)
                }}</el-descriptions-item>
            </el-descriptions>
        </el-card>

        <el-card class="box-card status-card">
            <template #header>
                <div class="card-header">
                    <span>同步与重载状态</span>
                </div>
            </template>
            <div v-if="syncStatus" class="status-body">
                <el-descriptions :column="2" border>
                    <el-descriptions-item label="配置哈希">
                        <span class="mono">{{
                            syncStatus.config_hash || "-"
                        }}</span>
                    </el-descriptions-item>
                    <el-descriptions-item label="拉取状态">
                        <el-tag :type="tagInfo(syncStatus.fetch_status).type">
                            {{ tagInfo(syncStatus.fetch_status).text }}
                        </el-tag>
                    </el-descriptions-item>
                    <el-descriptions-item label="重载状态">
                        <el-tag :type="tagInfo(syncStatus.reload_status).type">
                            {{ tagInfo(syncStatus.reload_status).text }}
                        </el-tag>
                    </el-descriptions-item>
                    <el-descriptions-item label="更新时间">{{
                        formatDate(syncStatus.updated_at)
                    }}</el-descriptions-item>
                    <el-descriptions-item label="错误信息" :span="2">
                        <span class="mono">{{
                            syncStatus.error_msg || "无"
                        }}</span>
                    </el-descriptions-item>
                </el-descriptions>
            </div>
            <el-empty v-else description="暂无同步信息" />
        </el-card>

        <el-card class="box-card status-card">
            <template #header>
                <div class="card-header">
                    <span>同步/重载历史</span>
                </div>
            </template>
            <el-table
                :data="historyList"
                v-loading="historyLoading"
                size="small"
                border
            >
                <el-table-column prop="created_at" label="时间" width="180">
                    <template #default="scope">
                        {{ formatDate(scope.row.created_at) }}
                    </template>
                </el-table-column>
                <el-table-column
                    prop="fetch_status"
                    label="拉取状态"
                    width="120"
                >
                    <template #default="scope">
                        <el-tag :type="tagInfo(scope.row.fetch_status).type">
                            {{ tagInfo(scope.row.fetch_status).text }}
                        </el-tag>
                    </template>
                </el-table-column>
                <el-table-column
                    prop="reload_status"
                    label="重载状态"
                    width="120"
                >
                    <template #default="scope">
                        <el-tag :type="tagInfo(scope.row.reload_status).type">
                            {{ tagInfo(scope.row.reload_status).text }}
                        </el-tag>
                    </template>
                </el-table-column>
                <el-table-column
                    prop="config_hash"
                    label="配置哈希"
                    min-width="240"
                >
                    <template #default="scope">
                        <span class="mono">{{
                            scope.row.config_hash || "-"
                        }}</span>
                    </template>
                </el-table-column>
                <el-table-column
                    prop="error_msg"
                    label="错误信息"
                    min-width="200"
                    show-overflow-tooltip
                >
                    <template #default="scope">
                        <span
                            class="mono clickable"
                            @click="copyError(scope.row.error_msg)"
                        >
                            {{ scope.row.error_msg || "-" }}
                        </span>
                    </template>
                </el-table-column>
            </el-table>
        </el-card>
    </div>
</template>

<script setup>
import { ref, onMounted, watch, computed } from "vue";
import { useRoute, useRouter } from "vue-router";
import axios from "axios";
import { ElMessage, ElMessageBox } from "element-plus";
import {
    loadUserPermissions,
    hasPermission,
    checkIsAdmin,
} from "../utils/permissions";

const route = useRoute();
const router = useRouter();

const loading = ref(false);
const manualLoading = ref(false);
const historyLoading = ref(false);
const detail = ref({});
const isAdmin = ref(false);
const canWrite = ref(false);
const historyList = ref([]);
const syncStatus = computed(() => detail.value.sync_status || null);

const id = route.params.id;

const formatDate = (d) => {
    if (!d) return "-";
    const dt = new Date(d);
    return dt.toLocaleString("zh-CN");
};

const tagInfo = (status) => {
    const s = (status || "").toLowerCase();
    switch (s) {
        case "updated":
        case "success":
            return { text: "成功", type: "success" };
        case "not_modified":
        case "unchanged":
        case "skipped":
            return { text: "无变更", type: "info" };
        case "failed":
            return { text: "失败", type: "danger" };
        default:
            if (!status) return { text: "-", type: "info" };
            return { text: status, type: "warning" };
    }
};

const getAuthHeaders = () => ({ Authorization: localStorage.getItem("token") });

const fetchDetail = async () => {
    loading.value = true;
    try {
        const res = await axios.get(`/api/agent/nodes/${id}`, {
            headers: getAuthHeaders(),
        });
        detail.value = res.data.data || {};
    } catch (err) {
        ElMessage.error(err.response?.data?.error || "加载详情失败");
    } finally {
        loading.value = false;
    }
};

const fetchHistory = async () => {
    historyLoading.value = true;
    try {
        const res = await axios.get(
            `/api/agent/nodes/${id}/history?limit=100`,
            { headers: getAuthHeaders() },
        );
        // 后端返回 {data: histories}，但也兼容直接数组；并统一字段为下划线风格
        const payload = res.data?.data ?? res.data ?? [];
        const list = Array.isArray(payload) ? payload : [];
        historyList.value = list.map((item) => ({
            id: item.id ?? item.ID,
            node_id: item.node_id ?? item.NodeID,
            config_hash: item.config_hash ?? item.ConfigHash,
            fetch_status: item.fetch_status ?? item.FetchStatus,
            reload_status: item.reload_status ?? item.ReloadStatus,
            error_msg: item.error_msg ?? item.ErrorMsg,
            created_at: item.created_at ?? item.CreatedAt,
        }));
    } catch (err) {
        ElMessage.error(err.response?.data?.error || "加载历史失败");
    } finally {
        historyLoading.value = false;
    }

    // 删除节点（仅在前端触发删除请求）
    const deleteNode = async () => {
        if (!canWrite.value) {
            ElMessage.error("无删除权限");
            return;
        }

        try {
            await ElMessageBox.confirm(
                "确定要删除该节点吗？此操作会删除该节点的所有下发规则关联（请谨慎操作）",
                "确认删除",
                {
                    type: "warning",
                    confirmButtonText: "确定",
                    cancelButtonText: "取消",
                },
            );
        } catch (e) {
            // 用户取消
            return;
        }

        // 执行删除
        loading.value = true;
        try {
            const res = await axios.delete(`/api/agent/nodes/${id}`, {
                headers: getAuthHeaders(),
            });
            ElMessage.success(res.data?.message || "节点删除成功");
            // 跳回节点列表
            router.push("/nodes");
        } catch (err) {
            ElMessage.error(err.response?.data?.error || "删除节点失败");
        } finally {
            loading.value = false;
        }
    };
};

const manualSync = async () => {
    manualLoading.value = true;
    try {
        await axios.post(
            `/api/agent/nodes/${id}/manual_sync`,
            {},
            { headers: getAuthHeaders() },
        );
        ElMessage.success("已触发手动拉取与重载");
        await fetchDetail();
        await fetchHistory();
    } catch (err) {
        ElMessage.error(err.response?.data?.error || "触发失败");
    } finally {
        manualLoading.value = false;
    }
};

const copyError = async (msg) => {
    const text = msg || "-";
    try {
        await navigator.clipboard.writeText(text);
        ElMessage.success("已复制错误信息");
    } catch (err) {
        ElMessage.error("复制失败");
    }
};

const goBack = () => router.back();

onMounted(async () => {
    await loadUserPermissions();
    isAdmin.value = checkIsAdmin();
    await fetchDetail();
    await fetchHistory();
    canWrite.value =
        isAdmin.value || hasPermission("node", parseInt(id), "write");
});

// 如果路由参数变化（从列表进入不同详情），自动刷新
watch(
    () => route.params.id,
    () => {
        fetchDetail();
        fetchHistory();
    },
);
</script>

<style scoped>
.detail-container {
    padding: 24px;
    height: 100%;
}

.status-card {
    margin-top: 16px;
}

:deep(.el-card) {
    border-radius: 12px;
    box-shadow: 0 4px 16px rgba(0, 0, 0, 0.08);
    border: none;
}

.card-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    font-weight: 600;
    color: #1890ff;
}

.group-edit {
    display: flex;
    align-items: center;
    gap: 8px;
}

:deep(.el-descriptions__label) {
    background: linear-gradient(135deg, #f0f5ff 0%, #e6f0ff 100%) !important;
    color: #1890ff;
    font-weight: 600;
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

:deep(.el-input__wrapper) {
    border-radius: 8px;
    transition: all 0.3s ease;
}

:deep(.el-input__wrapper:hover) {
    box-shadow: 0 2px 8px rgba(24, 144, 255, 0.2);
}

:deep(.el-tag) {
    border-radius: 6px;
    padding: 4px 12px;
    font-weight: 500;
}

.mono {
    font-family:
        "SFMono-Regular", Consolas, "Liberation Mono", Menlo, monospace;
    font-size: 13px;
    color: #3d3d3d;
}

.clickable {
    cursor: pointer;
}

.status-body {
    min-height: 120px;
}
</style>
