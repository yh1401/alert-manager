<template>
    <div class="nodes-container">
        <div class="toolbar">
            <el-input
                v-model="search"
                placeholder="搜索名称或IP"
                clearable
                style="max-width: 300px"
            />
            <el-select
                v-model="offlineSec"
                style="width: 180px; margin-left: 10px"
            >
                <el-option :value="60" label="离线阈值: 60s" />
                <el-option :value="180" label="离线阈值: 180s" />
                <el-option :value="300" label="离线阈值: 300s" />
            </el-select>
            <el-select
                v-model="selectedTags"
                multiple
                filterable
                placeholder="按标签筛选"
                style="min-width: 200px; margin-left: 10px"
                clearable
            >
                <el-option
                    v-for="tag in allTags"
                    :key="tag.id"
                    :label="tag.name"
                    :value="tag.name"
                />
            </el-select>
            <el-button
                type="primary"
                @click="fetchNodes"
                :loading="loading"
                style="margin-left: 10px"
                >刷新</el-button
            >
        </div>

        <el-table
            :data="filtered"
            border
            v-loading="loading"
            class="node-table"
        >
            <el-table-column prop="id" label="ID" width="80" />
            <el-table-column prop="name" label="名称" width="200" />
            <el-table-column prop="ip_address" label="IP 地址" width="180" />
            <el-table-column label="标签" min-width="180">
                <template #default="scope">
                    <el-tag
                        v-for="tag in scope.row.tags"
                        :key="tag.id"
                        size="small"
                        style="margin-right: 5px"
                    >
                        {{ tag.name }}
                    </el-tag>
                </template>
            </el-table-column>
            <el-table-column label="状态" width="120">
                <template #default="scope">
                    <el-tag
                        :type="
                            scope.row.status === 'online' ? 'success' : 'danger'
                        "
                    >
                        {{ scope.row.status === "online" ? "在线" : "离线" }}
                    </el-tag>
                </template>
            </el-table-column>
            <el-table-column prop="last_heartbeat" label="最后心跳" width="200">
                <template #default="scope">{{
                    formatDate(scope.row.last_heartbeat)
                }}</template>
            </el-table-column>
            <el-table-column prop="updated_at" label="更新时间" width="200">
                <template #default="scope">{{
                    formatDate(scope.row.updated_at)
                }}</template>
            </el-table-column>
            <el-table-column label="操作" fixed="right" width="220">
                <template #default="scope">
                    <el-button
                        link
                        type="primary"
                        size="small"
                        @click="viewRules(scope.row)"
                        v-if="canRead(scope.row)"
                        >查看规则</el-button
                    >
                    <el-divider
                        direction="vertical"
                        v-if="canRead(scope.row)"
                    />
                    <el-button
                        link
                        type="default"
                        size="small"
                        @click="viewDetail(scope.row)"
                        v-if="canRead(scope.row)"
                        >详情</el-button
                    >
                    <el-divider
                        direction="vertical"
                        v-if="canWrite(scope.row)"
                    />
                    <el-button
                        link
                        type="warning"
                        size="small"
                        @click="openTagDialog(scope.row)"
                        v-if="canWrite(scope.row)"
                        >编辑标签</el-button
                    >
                    <el-divider
                        direction="vertical"
                        v-if="canWrite(scope.row)"
                    />
                    <el-button
                        link
                        type="danger"
                        size="small"
                        @click="deleteNode(scope.row)"
                        v-if="canWrite(scope.row)"
                        >删除</el-button
                    >
                </template>
            </el-table-column>
        </el-table>

        <el-dialog v-model="isTagDialogVisible" title="编辑标签" width="400px">
            <el-select
                v-model="currentNodeTags"
                multiple
                filterable
                allow-create
                default-first-option
                placeholder="输入并选择或创建标签"
                style="width: 100%"
            >
                <el-option
                    v-for="tag in allTags"
                    :key="tag.name"
                    :label="tag.name"
                    :value="tag.name"
                />
            </el-select>
            <template #footer>
                <span class="dialog-footer">
                    <el-button @click="isTagDialogVisible = false"
                        >取消</el-button
                    >
                    <el-button type="primary" @click="updateNodeTags">
                        保存
                    </el-button>
                </span>
            </template>
        </el-dialog>
    </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted } from "vue";
import axios from "axios";
import { ElMessage, ElMessageBox } from "element-plus";
import { useRouter } from "vue-router";
import {
    loadUserPermissions,
    hasPermission,
    checkIsAdmin,
} from "../utils/permissions";

const router = useRouter();
const list = ref([]);
const loading = ref(false);
const search = ref("");
const offlineSec = ref(180);
const isAdmin = ref(false);
const allTags = ref([]);
const selectedTags = ref([]);
const isTagDialogVisible = ref(false);
const currentNode = ref(null);
const currentNodeTags = ref([]);
let poller = null;

const formatDate = (d) => {
    if (!d) return "-";
    const dt = new Date(d);
    return dt.toLocaleString("zh-CN");
};

const filtered = computed(() => {
    const s = (search.value || "").toLowerCase();
    let data = list.value.filter(
        (x) =>
            !s ||
            (x.name || "").toLowerCase().includes(s) ||
            (x.ip_address || "").toLowerCase().includes(s),
    );

    if (selectedTags.value.length > 0) {
        data = data.filter((row) => {
            if (!row.tags || row.tags.length === 0) {
                return false;
            }
            return selectedTags.value.every((selectedTag) =>
                (row.tags || []).some((tag) => tag.name === selectedTag),
            );
        });
    }

    return data;
});

const getAuthHeaders = () => ({ Authorization: localStorage.getItem("token") });

const fetchTags = async () => {
    try {
        const res = await axios.get("/api/tags", { headers: getAuthHeaders() });
        if (res.data.data) {
            allTags.value = res.data.data;
        }
    } catch (err) {
        console.error("Failed to fetch tags:", err);
        ElMessage.error("加载标签列表失败");
    }
};

const fetchNodes = async () => {
    loading.value = true;
    try {
        const res = await axios.get("/api/agent/nodes", {
            params: { offline_sec: offlineSec.value },
            headers: getAuthHeaders(),
        });
        list.value = res.data.data || [];
    } catch (err) {
        ElMessage.error(err.response?.data?.error || "加载节点失败");
    } finally {
        loading.value = false;
    }
};

const openTagDialog = (node) => {
    currentNode.value = node;
    currentNodeTags.value = node.tags ? node.tags.map((t) => t.name) : [];
    isTagDialogVisible.value = true;
};

const updateNodeTags = async () => {
    if (!currentNode.value) return;
    try {
        await axios.post(
            `/api/agent/nodes/${currentNode.value.id}/tags`,
            { tags: currentNodeTags.value },
            { headers: getAuthHeaders() },
        );
        ElMessage.success("标签更新成功");
        isTagDialogVisible.value = false;
        fetchNodes();
    } catch (err) {
        ElMessage.error(err.response?.data?.error || "更新标签失败");
    }
};

const viewRules = (row) => {
    // 跳转到规则列表并带上节点过滤参数（按 node_id）
    router.push({
        path: "/rules",
        query: { searchType: "node_id", searchValue: String(row.id) },
    });
};

const viewDetail = (row) => {
    router.push(`/nodes/${row.id}`);
};

const deleteNode = async (row) => {
    // 权限二次校验（前端）
    if (!canWrite.value && !canWrite(row)) {
        ElMessage.error("无删除权限");
        return;
    }

    try {
        await ElMessageBox.confirm(
            "确定要删除该节点吗？此操作会将该节点下的规则置为失效，且不可恢复。",
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

    loading.value = true;
    try {
        const res = await axios.delete(`/api/agent/nodes/${row.id}`, {
            headers: getAuthHeaders(),
        });
        ElMessage.success(res.data?.message || "节点删除成功");
        // 刷新列表
        fetchNodes();
        // 如果当前正查看该节点详情页，则跳回节点列表
        if (String(id) === String(row.id)) {
            router.push("/nodes");
        }
    } catch (err) {
        ElMessage.error(err.response?.data?.error || "删除节点失败");
    } finally {
        loading.value = false;
    }
};

// 权限检查函数
const canRead = (row) => {
    return isAdmin.value || hasPermission("node", row.id, "read");
};

const canWrite = (row) => {
    return isAdmin.value || hasPermission("node", row.id, "write");
};

// 停止轮询
const stopPoller = () => {
    if (poller) {
        clearInterval(poller);
        poller = null;
    }
};

onMounted(async () => {
    await loadUserPermissions();
    isAdmin.value = checkIsAdmin();
    await fetchTags();
    fetchNodes();
    // 每 30s 自动刷新一次节点列表
    poller = setInterval(fetchNodes, 30000);
});

// 组件卸载时清除定时器
onUnmounted(() => {
    stopPoller();
});
</script>

<style scoped>
.nodes-container {
    display: flex;
    flex-direction: column;
    height: 100%;
    gap: 16px;
}

.toolbar {
    display: flex;
    align-items: center;
    gap: 12px;
    flex-shrink: 0;
    padding: 20px;
    background: white;
    border-radius: 12px;
    box-shadow: 0 2px 8px rgba(0, 0, 0, 0.06);
}

.node-table {
    flex: 1;
    overflow: auto;
    min-height: 300px;
    background: white;
    border-radius: 12px;
    padding: 16px;
    box-shadow: 0 2px 12px rgba(0, 0, 0, 0.08);
}

:deep(.el-table) {
    border-radius: 8px;
    overflow: hidden;
}

:deep(.el-table th) {
    background: linear-gradient(135deg, #f0f5ff 0%, #e6f0ff 100%) !important;
    color: #1890ff;
    font-weight: 600;
}

:deep(.el-table tr:hover) {
    background: #f0f7ff !important;
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

:deep(.el-tag) {
    border-radius: 6px;
    padding: 4px 12px;
    font-weight: 500;
}
</style>
