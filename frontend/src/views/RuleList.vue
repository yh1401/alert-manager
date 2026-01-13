<template>
    <div class="rule-list-container">
        <div class="toolbar">
            <el-button
                type="primary"
                icon="Plus"
                @click="handleCreate"
                v-if="isAdmin"
            >
                新建规则
            </el-button>
            <el-button
                type="default"
                icon="Refresh"
                @click="fetchData"
                :loading="loading"
            >
                刷新
            </el-button>
        </div>

        <div class="search-bar">
            <el-select
                v-model="searchType"
                placeholder="搜索方式"
                style="width: 150px"
            >
                <el-option label="规则 ID" value="id" />
                <el-option label="规则名称" value="name" />
                <el-option label="节点 ID" value="node_id" />
                <el-option label="文件路径" value="file_path" />
            </el-select>
            <el-input
                v-model="searchValue"
                placeholder="输入搜索值"
                clearable
                style="max-width: 300px"
                @input="handleSearch"
            />
            <el-button type="default" @click="resetSearch">重置</el-button>
        </div>

        <el-table
            :data="filteredTableData"
            border
            class="rule-table"
            v-loading="loading"
        >
            <el-table-column prop="id" label="ID" width="80" />
            <el-table-column prop="node_id" label="节点 ID" width="100" />
            <el-table-column
                prop="file_path"
                label="文件路径"
                min-width="220"
            />
            <el-table-column prop="name" label="规则名称" min-width="160" />
            <el-table-column
                prop="version"
                label="版本"
                width="80"
                align="center"
            />
            <el-table-column prop="updated_at" label="更新时间" width="180">
                <template #default="scope">
                    {{ formatDate(scope.row.updated_at) }}
                </template>
            </el-table-column>
            <el-table-column label="操作" width="260" fixed="right">
                <template #default="scope">
                    <el-button
                        link
                        type="primary"
                        size="small"
                        @click="handleEdit(scope.row)"
                        v-if="canWrite(scope.row)"
                    >
                        编辑
                    </el-button>
                    <el-divider
                        direction="vertical"
                        v-if="canWrite(scope.row)"
                    />
                    <el-button
                        link
                        type="success"
                        size="small"
                        @click="handleViewHistory(scope.row)"
                        v-if="canRead(scope.row)"
                    >
                        版本历史
                    </el-button>
                    <el-divider
                        direction="vertical"
                        v-if="canRead(scope.row)"
                    />
                    <el-button
                        link
                        type="danger"
                        size="small"
                        @click="handleDelete(scope.row)"
                        v-if="canWrite(scope.row)"
                    >
                        删除
                    </el-button>
                </template>
            </el-table-column>
        </el-table>
    </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted, watch } from "vue";
import axios from "axios";
import { ElMessage, ElMessageBox } from "element-plus";
import { useRouter, useRoute } from "vue-router";
import {
    loadUserPermissions,
    hasPermission,
    checkIsAdmin,
} from "../utils/permissions";

const router = useRouter();
const route = useRoute();

const tableData = ref([]);
const loading = ref(false);
const isAdmin = ref(false);

// 搜索相关
const searchType = ref("name"); // id | name | node_id | file_path
const searchValue = ref("");

// 搜索过滤结果
const filteredTableData = computed(() => {
    if (!searchValue.value) return tableData.value;
    const query = searchValue.value.toLowerCase();
    return tableData.value.filter((row) => {
        if (searchType.value === "id") {
            return row.id?.toString().includes(query);
        } else if (searchType.value === "name") {
            return (row.name || "").toLowerCase().includes(query);
        } else if (searchType.value === "node_id") {
            return (row.node_id || "").toString().includes(query);
        } else if (searchType.value === "file_path") {
            return (row.file_path || "").toLowerCase().includes(query);
        }
        return true;
    });
});

// 表单数据占位（仅用于兼容 YAML->form 切换逻辑）
const ruleForm = reactive({
    rules: [],
});

// 获取请求头辅助函数
const getAuthHeaders = () => {
    return { Authorization: localStorage.getItem("token") };
};

const formatDate = (dateStr) => {
    if (!dateStr) return "-";
    const date = new Date(dateStr);
    return date.toLocaleString("zh-CN");
};

const fetchData = async () => {
    loading.value = true;
    try {
        const res = await axios.get("/api/rule/list", {
            headers: getAuthHeaders(),
        });
        if (res.data.data) {
            tableData.value = res.data.data;
        }
    } catch (err) {
        ElMessage.error(err.response?.data?.error || "加载规则失败");
    } finally {
        loading.value = false;
    }
};

const handleSearch = () => {
    // computed 已处理
};

const resetSearch = () => {
    searchValue.value = "";
    searchType.value = "name";
};

const handleCreate = () => {
    router.push("/rules/new");
};

const handleEdit = (row) => {
    router.push(`/rules/${row.id}/edit`);
};

// 查看版本历史
const handleViewHistory = (row) => {
    router.push({
        name: "RuleVersion",
        params: { ruleId: row.id },
    });
};

// 删除规则
const handleDelete = (row) => {
    ElMessageBox.confirm(`确定要删除规则 "${row.name}" 吗？`, "警告", {
        confirmButtonText: "确定",
        cancelButtonText: "取消",
        type: "warning",
    }).then(async () => {
        try {
            await axios.post(
                "/api/rule/delete_rule",
                { id: row.id },
                { headers: getAuthHeaders() },
            );
            ElMessage.success("删除成功");
            fetchData();
        } catch (err) {
            ElMessage.error(err.response?.data?.error || "删除失败");
        }
    });
};

// 权限检查函数
const canRead = (row) => {
    return isAdmin.value || hasPermission("rule", row.id, "read");
};

const canWrite = (row) => {
    return isAdmin.value || hasPermission("rule", row.id, "write");
};

onMounted(async () => {
    await loadUserPermissions();
    isAdmin.value = checkIsAdmin();
    fetchData().then(() => {
        // 初始化搜索参数（支持从节点列表跳转带 query）
        if (route.query.searchType && route.query.searchValue) {
            searchType.value = route.query.searchType;
            searchValue.value = route.query.searchValue;
        }
    });
});

// 监听路由查询参数变化
watch(
    () => route.query,
    (newQuery) => {
        if (newQuery.searchType && newQuery.searchValue) {
            searchType.value = newQuery.searchType;
            searchValue.value = newQuery.searchValue;
        }
    },
    { deep: true },
);
</script>

<style scoped>
.rule-list-container {
    display: flex;
    flex-direction: column;
    height: 100%;
    gap: 16px;
}

.toolbar {
    display: flex;
    gap: 12px;
    align-items: center;
    flex-shrink: 0;
    padding: 20px;
    background: white;
    border-radius: 12px;
    box-shadow: 0 2px 8px rgba(0, 0, 0, 0.06);
}

.search-bar {
    display: flex;
    gap: 12px;
    align-items: center;
    flex-shrink: 0;
    padding: 20px;
    background: white;
    border-radius: 12px;
    box-shadow: 0 2px 8px rgba(0, 0, 0, 0.06);
}

.rule-table {
    flex: 1;
    overflow: auto;
    width: 100%;
    min-height: 300px;
    background: white;
    border-radius: 12px;
    padding: 16px;
    box-shadow: 0 2px 12px rgba(0, 0, 0, 0.08);
}

:deep(.el-button--primary) {
    background: linear-gradient(135deg, #1890ff 0%, #096dd9 100%);
    border: none;
    border-radius: 8px;
    padding: 10px 20px;
    transition: all 0.3s ease;
    box-shadow: 0 4px 12px rgba(24, 144, 255, 0.3);
}

:deep(.el-button--primary:hover) {
    transform: translateY(-2px);
    box-shadow: 0 6px 16px rgba(24, 144, 255, 0.4);
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
</style>
