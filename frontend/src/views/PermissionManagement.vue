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
                    <UserManagementTab
                        :users="users"
                        @refresh="fetchUsers"
                        @view-permissions="viewUserPermissions"
                        @change-role="openRoleDialog"
                        @batch-auth="batchSetPermissions"
                    />
                </el-tab-pane>

                <!-- 权限详情 -->
                <el-tab-pane
                    label="权限详情"
                    name="permissions"
                    v-if="selectedUser"
                >
                    <PermissionDetailTab
                        :user="selectedUser"
                        :permissions="permissions"
                        @refresh="refreshPermissions"
                        @remove="removePermission"
                    />
                </el-tab-pane>

                <!-- 批量授权 -->
                <el-tab-pane label="批量授权" name="batch" v-if="selectedUser">
                    <BatchPermissionTab
                        :user="selectedUser"
                        @submit-set="submitBatchSet"
                        @submit-remove="submitBatchRemove"
                    />
                </el-tab-pane>
            </el-tabs>
        </el-card>

        <!-- 修改角色弹窗 -->
        <el-dialog v-model="roleDialogVisible" title="修改角色" width="400px">
            <el-form v-if="editingUser" label-width="80px">
                <el-form-item label="用户">
                    <el-tag type="info">{{ editingUser.username }}</el-tag>
                </el-form-item>
                <el-form-item label="新角色">
                    <el-select v-model="newRoleValue" placeholder="请选择角色">
                        <el-option label="Admin" value="admin"></el-option>
                        <el-option label="User" value="user"></el-option>
                    </el-select>
                </el-form-item>
            </el-form>
            <template #footer>
                <el-button @click="roleDialogVisible = false">取消</el-button>
                <el-button type="primary" @click="handleRoleUpdate"
                    >确定</el-button
                >
            </template>
        </el-dialog>
    </div>
</template>

<script setup>
import { ref, onMounted } from "vue";
import axios from "axios";
import { ElMessage, ElMessageBox } from "element-plus";
import UserManagementTab from "@/components/permissions/UserManagementTab.vue";
import PermissionDetailTab from "@/components/permissions/PermissionDetailTab.vue";
import BatchPermissionTab from "@/components/permissions/BatchPermissionTab.vue";

const API_BASE = "/api";
const activeTab = ref("users");
const users = ref([]);
const selectedUser = ref(null);
const permissions = ref([]);

// State for role editing dialog
const roleDialogVisible = ref(false);
const editingUser = ref(null);
const newRoleValue = ref("");

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

const openRoleDialog = (user) => {
    editingUser.value = user;
    newRoleValue.value = user.role;
    roleDialogVisible.value = true;
};

const handleRoleUpdate = async () => {
    if (!editingUser.value) return;

    try {
        await axios.post(
            `${API_BASE}/admin/users/role`,
            {
                user_id: editingUser.value.id,
                role: newRoleValue.value,
            },
            {
                headers: { Authorization: `Bearer ${getToken()}` },
            },
        );
        ElMessage.success("角色修改成功");
        roleDialogVisible.value = false;
        fetchUsers();
    } catch (err) {
        ElMessage.error(
            "修改角色失败: " + (err.response?.data?.error || err.message),
        );
    }
};

const batchSetPermissions = (user) => {
    selectedUser.value = user;
    activeTab.value = "batch";
};

const submitBatchSet = async (payload, resetCallback) => {
    try {
        const res = await axios.post(
            `${API_BASE}/admin/permissions/batch-set`,
            payload,
            {
                headers: { Authorization: `Bearer ${getToken()}` },
            },
        );
        ElMessage.success(`批量授权成功: ${res.data.count}/${res.data.total}`);
        if (resetCallback) resetCallback();
    } catch (err) {
        ElMessage.error(
            "批量授权失败: " + (err.response?.data?.error || err.message),
        );
    }
};

const submitBatchRemove = async (payload, resetCallback) => {
    try {
        await ElMessageBox.confirm("确认批量移除权限？", "提示", {
            type: "warning",
        });
        const res = await axios.post(
            `${API_BASE}/admin/permissions/batch-remove`,
            payload,
            {
                headers: { Authorization: `Bearer ${getToken()}` },
            },
        );
        ElMessage.success(`批量移除成功: ${res.data.count} 条`);
        if (resetCallback) resetCallback();
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

:deep(.el-card) {
    border-radius: 12px;
    box-shadow: 0 2px 8px rgba(0, 0, 0, 0.06);
    border: none;
}
</style>
