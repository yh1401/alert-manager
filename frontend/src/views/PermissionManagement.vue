<template>
    <div class="permission-container">
        <el-card class="permission-card">
            <template #header>
                <div class="card-header">
                    <span>权限管理</span>
                </div>
            </template>

            <el-tabs v-model="activeTab" class="permission-tabs">
                <!-- 用户列表 -->
                <el-tab-pane label="用户管理" name="users">
                    <div class="tab-content">
                        <el-button
                            type="primary"
                            @click="fetchUsers"
                            icon="Refresh"
                            style="margin-bottom: 16px"
                            >刷新</el-button
                        >
                        <el-table
                            :data="users"
                            stripe
                            border
                            class="permission-table"
                        >
                            <el-table-column prop="id" label="ID" width="80" />
                            <el-table-column prop="username" label="用户名" />
                            <el-table-column prop="role" label="角色">
                                <template #default="{ row }">
                                    <el-tag
                                        :type="
                                            row.role === 'admin'
                                                ? 'danger'
                                                : 'info'
                                        "
                                        >{{ row.role }}</el-tag
                                    >
                                </template>
                            </el-table-column>
                            <el-table-column label="操作" width="300">
                                <template #default="{ row }">
                                    <el-button
                                        size="small"
                                        @click="viewUserPermissions(row)"
                                        >查看权限</el-button
                                    >
                                    <el-button
                                        size="small"
                                        type="warning"
                                        @click="changeRole(row)"
                                        >修改角色</el-button
                                    >
                                    <el-button
                                        size="small"
                                        type="primary"
                                        @click="batchSetPermissions(row)"
                                        >批量授权</el-button
                                    >
                                </template>
                            </el-table-column>
                        </el-table>
                    </div>
                </el-tab-pane>

                <!-- 权限详情 -->
                <el-tab-pane
                    label="权限详情"
                    name="permissions"
                    v-if="selectedUser"
                >
                    <div class="tab-content">
                        <div style="margin-bottom: 16px">
                            <el-tag type="success"
                                >用户: {{ selectedUser.username }}</el-tag
                            >
                            <el-button
                                type="primary"
                                @click="refreshPermissions"
                                icon="Refresh"
                                style="margin-left: 10px"
                                >刷新权限</el-button
                            >
                        </div>
                        <el-table
                            :data="permissions"
                            stripe
                            border
                            class="permission-table"
                        >
                            <el-table-column prop="id" label="ID" width="80" />
                            <el-table-column
                                prop="resource_type"
                                label="资源类型"
                                width="120"
                            >
                                <template #default="{ row }">
                                    <el-tag
                                        :type="
                                            row.resource_type === 'rule'
                                                ? 'primary'
                                                : 'success'
                                        "
                                        >{{ row.resource_type }}</el-tag
                                    >
                                </template>
                            </el-table-column>
                            <el-table-column
                                prop="resource_id"
                                label="资源ID"
                                width="100"
                            />
                            <el-table-column prop="action" label="权限">
                                <template #default="{ row }">
                                    <el-tag
                                        :type="
                                            row.action === 'write'
                                                ? 'warning'
                                                : 'info'
                                        "
                                        >{{ row.action }}</el-tag
                                    >
                                </template>
                            </el-table-column>
                            <el-table-column
                                prop="created_at"
                                label="创建时间"
                                width="180"
                            />
                            <el-table-column label="操作" width="120">
                                <template #default="{ row }">
                                    <el-button
                                        size="small"
                                        type="danger"
                                        @click="removePermission(row)"
                                        >移除</el-button
                                    >
                                </template>
                            </el-table-column>
                        </el-table>
                    </div>
                </el-tab-pane>

                <!-- 批量授权 -->
                <el-tab-pane label="批量授权" name="batch" v-if="selectedUser">
                    <div class="tab-content">
                        <el-form :model="batchForm" label-width="120px">
                            <el-form-item label="用户">
                                <el-tag type="success">{{
                                    selectedUser.username
                                }}</el-tag>
                            </el-form-item>
                            <el-form-item label="资源类型">
                                <el-radio-group
                                    v-model="batchForm.resource_type"
                                >
                                    <el-radio label="rule">规则</el-radio>
                                    <el-radio label="node">节点</el-radio>
                                </el-radio-group>
                            </el-form-item>
                            <el-form-item label="资源ID列表">
                                <el-input
                                    v-model="batchForm.resource_ids_str"
                                    placeholder="输入ID，用逗号分隔，如：1,2,3"
                                />
                            </el-form-item>
                            <el-form-item label="权限类型">
                                <el-radio-group v-model="batchForm.action">
                                    <el-radio label="read">只读</el-radio>
                                    <el-radio label="write">读写</el-radio>
                                </el-radio-group>
                            </el-form-item>
                            <el-form-item>
                                <el-button
                                    type="primary"
                                    @click="submitBatchSet"
                                    >批量授权</el-button
                                >
                                <el-button
                                    type="danger"
                                    @click="submitBatchRemove"
                                    >批量移除</el-button
                                >
                                <el-button @click="resetBatchForm"
                                    >重置</el-button
                                >
                            </el-form-item>
                        </el-form>
                    </div>
                </el-tab-pane>
            </el-tabs>
        </el-card>
    </div>
</template>

<script setup>
import { ref, reactive, onMounted } from "vue";
import axios from "axios";
import { ElMessage, ElMessageBox } from "element-plus";

const API_BASE = "http://localhost:8080/api";
const activeTab = ref("users");
const users = ref([]);
const selectedUser = ref(null);
const permissions = ref([]);

const batchForm = reactive({
    resource_type: "rule",
    resource_ids_str: "",
    action: "read",
});

const getToken = () => localStorage.getItem("token");

const fetchUsers = async () => {
    try {
        const res = await axios.get(`${API_BASE}/admin/users`, {
            headers: { Authorization: `Bearer ${getToken()}` },
        });
        users.value = res.data.data || [];
        ElMessage.success("用户列表加载成功");
    } catch (err) {
        ElMessage.error(
            "加载用户列表失败: " + (err.response?.data?.error || err.message),
        );
    }
};

const viewUserPermissions = async (user) => {
    selectedUser.value = user;
    activeTab.value = "permissions";
    await refreshPermissions();
};

const refreshPermissions = async () => {
    if (!selectedUser.value) return;
    try {
        const res = await axios.get(
            `${API_BASE}/admin/users/${selectedUser.value.id}/permissions`,
            {
                headers: { Authorization: `Bearer ${getToken()}` },
            },
        );
        permissions.value = res.data.data || [];
        ElMessage.success("权限列表加载成功");
    } catch (err) {
        ElMessage.error(
            "加载权限失败: " + (err.response?.data?.error || err.message),
        );
    }
};

const removePermission = async (perm) => {
    try {
        await ElMessageBox.confirm("确认移除该权限？", "提示", {
            type: "warning",
        });
        await axios.post(
            `${API_BASE}/admin/permissions/remove`,
            {
                user_id: perm.user_id,
                resource_type: perm.resource_type,
                resource_id: perm.resource_id,
                action: perm.action,
            },
            {
                headers: { Authorization: `Bearer ${getToken()}` },
            },
        );
        ElMessage.success("移除成功");
        refreshPermissions();
    } catch (err) {
        if (err !== "cancel") {
            ElMessage.error(
                "移除失败: " + (err.response?.data?.error || err.message),
            );
        }
    }
};

const changeRole = async (user) => {
    try {
        const newRole = await ElMessageBox.prompt(
            `修改用户 ${user.username} 的角色`,
            "提示",
            {
                confirmButtonText: "确定",
                cancelButtonText: "取消",
                inputValue: user.role,
                inputPattern: /^(admin|user)$/,
                inputErrorMessage: "角色必须是 admin 或 user",
            },
        );
        await axios.post(
            `${API_BASE}/admin/users/role`,
            {
                user_id: user.id,
                role: newRole.value,
            },
            {
                headers: { Authorization: `Bearer ${getToken()}` },
            },
        );
        ElMessage.success("角色修改成功");
        fetchUsers();
    } catch (err) {
        if (err !== "cancel") {
            ElMessage.error(
                "修改角色失败: " + (err.response?.data?.error || err.message),
            );
        }
    }
};

const batchSetPermissions = (user) => {
    selectedUser.value = user;
    activeTab.value = "batch";
    resetBatchForm();
};

const resetBatchForm = () => {
    batchForm.resource_type = "rule";
    batchForm.resource_ids_str = "";
    batchForm.action = "read";
};

const submitBatchSet = async () => {
    if (!selectedUser.value) {
        ElMessage.error("未选择用户");
        return;
    }
    try {
        const payload = {
            user_id: selectedUser.value.id,
            resource_type: batchForm.resource_type,
            action: batchForm.action,
            resource_ids: [],
        };
        if (!batchForm.resource_ids_str.trim()) {
            ElMessage.error("请输入资源ID列表");
            return;
        }
        payload.resource_ids = batchForm.resource_ids_str
            .split(",")
            .map((id) => parseInt(id.trim()))
            .filter((id) => !isNaN(id));
        if (payload.resource_ids.length === 0) {
            ElMessage.error("请输入有效的资源ID");
            return;
        }
        const res = await axios.post(
            `${API_BASE}/admin/permissions/batch-set`,
            payload,
            {
                headers: { Authorization: `Bearer ${getToken()}` },
            },
        );
        ElMessage.success(`批量授权成功: ${res.data.count}/${res.data.total}`);
        resetBatchForm();
    } catch (err) {
        ElMessage.error(
            "批量授权失败: " + (err.response?.data?.error || err.message),
        );
    }
};

const submitBatchRemove = async () => {
    if (!selectedUser.value) {
        ElMessage.error("未选择用户");
        return;
    }
    try {
        await ElMessageBox.confirm("确认批量移除权限？", "提示", {
            type: "warning",
        });
        const payload = {
            user_id: selectedUser.value.id,
            resource_type: batchForm.resource_type,
            action: batchForm.action,
            resource_ids: [],
        };
        if (!batchForm.resource_ids_str.trim()) {
            ElMessage.error("请输入资源ID列表");
            return;
        }
        payload.resource_ids = batchForm.resource_ids_str
            .split(",")
            .map((id) => parseInt(id.trim()))
            .filter((id) => !isNaN(id));
        if (payload.resource_ids.length === 0) {
            ElMessage.error("请输入有效的资源ID");
            return;
        }
        const res = await axios.post(
            `${API_BASE}/admin/permissions/batch-remove`,
            payload,
            {
                headers: { Authorization: `Bearer ${getToken()}` },
            },
        );
        ElMessage.success(`批量移除成功: ${res.data.count} 条`);
        resetBatchForm();
    } catch (err) {
        if (err !== "cancel") {
            ElMessage.error(
                "批量移除失败: " + (err.response?.data?.error || err.message),
            );
        }
    }
};

onMounted(() => {
    fetchUsers();
});
</script>

<style scoped>
.permission-container {
    height: 100%;
    width: 100%;
    display: flex;
    flex-direction: column;
}

.permission-card {
    height: 100%;
    width: 100%;
    display: flex;
    flex-direction: column;
}

.card-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
}

.permission-tabs {
    height: 100%;
    display: flex;
    flex-direction: column;
}

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

:deep(.el-tabs__content) {
    flex: 1;
    overflow: hidden;
    padding: 0;
}

:deep(.el-tab-pane) {
    height: 100%;
    overflow-y: auto;
}

:deep(.el-tabs__header) {
    margin-bottom: 20px;
}

:deep(.el-tabs__item) {
    font-weight: 500;
    transition: all 0.3s ease;
}

:deep(.el-tabs__item.is-active) {
    color: #1890ff;
    font-weight: 600;
}

:deep(.el-tabs__active-bar) {
    background: linear-gradient(90deg, #1890ff 0%, #096dd9 100%);
    height: 3px;
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

:deep(.el-card) {
    border-radius: 12px;
    box-shadow: 0 2px 8px rgba(0, 0, 0, 0.06);
    border: none;
}
</style>
