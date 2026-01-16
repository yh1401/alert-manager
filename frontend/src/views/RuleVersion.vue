<template>
    <div class="rule-version-container">
        <!-- 顶部信息 -->
        <div style="
                margin-bottom: 20px;
                padding: 15px;
                background: #f5f7fa;
                border-radius: 4px;
            ">
            <el-row :gutter="20">
                <el-col :span="12">
                    <div><strong>规则名称：</strong>{{ ruleInfo.name }}</div>
                    <div><strong>节点 ID：</strong>{{ ruleInfo.node_id }}</div>
                    <div>
                        <strong>文件路径：</strong>{{ ruleInfo.file_path }}
                    </div>
                </el-col>
                <el-col :span="12">
                    <div><strong>当前版本：</strong>{{ ruleInfo.version }}</div>
                    <div>
                        <strong>最后更新：</strong>{{ formatDate(ruleInfo.updated_at) }}
                    </div>
                </el-col>
            </el-row>
        </div>

        <!-- 版本列表 + 详情（左侧侧栏，右侧内容） -->
        <div class="version-layout">
            <!-- 左侧侧栏 -->
            <VersionHistorySidebar :versions="versionList" :selected-version="selectedVersion"
                @select="handleVersionSelect" />

            <!-- 右侧详情与对比 -->
            <div class="panel main" v-if="selectedVersion">
                <div class="panel-header panel-header-actions">
                    <span>v{{ selectedVersion.version }} 详情</span>
                    <el-button v-if="!selectedVersion.is_current" type="primary" size="small" @click="handleRollback"
                        :loading="rollbackLoading">
                        回滚到此版本
                    </el-button>
                </div>

                <div class="panel-body">
                    <el-descriptions border :column="2">
                        <el-descriptions-item label="版本号">v{{
                            selectedVersion.version
                        }}</el-descriptions-item>
                        <el-descriptions-item label="状态">
                            <el-tag v-if="selectedVersion.is_current" type="success">当前版本</el-tag>
                            <el-tag v-else type="info">历史版本</el-tag>
                        </el-descriptions-item>
                        <el-descriptions-item label="更新时间">{{
                            formatDate(selectedVersion.created_at)
                        }}</el-descriptions-item>
                        <el-descriptions-item label="备注">{{
                            selectedVersion.comment
                        }}</el-descriptions-item>
                    </el-descriptions>

                    <CodeDiffViewer v-if="currentRuleContent !== null"
                        :old-title="`历史版本 v${selectedVersion.version}`"
                        :new-title="`当前版本 v${ruleInfo.version}`"
                        :old-content="selectedVersion.file_content"
                        :new-content="currentRuleContent"
                        style="margin-top: 20px;" />
                </div>
            </div>
            <div v-else class="panel main placeholder">
                请选择一个版本查看详情
            </div>
        </div>

        <!-- 回滚确认对话框 -->
        <el-dialog v-model="rollbackDialogVisible" title="确认回滚" width="400px">
            <p>
                确定要回滚到
                <strong>v{{ selectedVersion?.version }}</strong> 吗？
            </p>
            <p style="color: #909399; font-size: 12px; margin-top: 10px">
                当前规则内容将被保存为历史版本，不可恢复。请谨慎操作。
            </p>
            <el-form :model="rollbackForm" style="margin-top: 15px">
                <el-form-item label="回滚说明">
                    <el-input v-model="rollbackForm.comment" type="textarea" :rows="3" placeholder="请输入回滚原因（可选）" />
                </el-form-item>
            </el-form>
            <template #footer>
                <el-button @click="rollbackDialogVisible = false">取消</el-button>
                <el-button type="danger" @click="confirmRollback" :loading="rollbackLoading">确认回滚</el-button>
            </template>
        </el-dialog>
    </div>
</template>

<script setup>
import { ref, reactive, onMounted, computed } from "vue";
import { useRoute, useRouter } from "vue-router";
import axios from "axios";
import { ElMessage } from "element-plus";
import VersionHistorySidebar from "@/components/rule-version/VersionHistorySidebar.vue";
import CodeDiffViewer from "@/components/shared/CodeDiffViewer.vue";

const props = defineProps({
    ruleId: Number,
});
const route = useRoute();
const router = useRouter();
// 统一的 ruleId 来源：优先 props，其次路由参数
const effectiveRuleId = computed(() => {
    if (props.ruleId) return props.ruleId;
    const paramId = parseInt(route.params.ruleId, 10);
    return isNaN(paramId) ? null : paramId;
});

// 数据
const ruleInfo = reactive({
    name: "",
    node_id: "",
    file_path: "",
    version: 0,
    updated_at: "",
});

const versionList = ref([]);
const selectedVersion = ref(null);
const rollbackLoading = ref(false);
const rollbackDialogVisible = ref(false);
const currentRuleContent = ref("");

const rollbackForm = reactive({
    comment: "",
});

// 获取授权头
const getAuthHeaders = () => {
    return { Authorization: localStorage.getItem("token") };
};

// 格式化日期
const formatDate = (dateStr) => {
    if (!dateStr) return "-";
    const date = new Date(dateStr);
    return date.toLocaleString("zh-CN");
};

// 获取版本历史
const fetchVersions = async () => {
    try {
        const res = await axios.get("/api/rule/versions", {
            params: { id: effectiveRuleId.value },
            headers: getAuthHeaders(),
        });
        if (res.data.data) {
            versionList.value = res.data.data;

            // 设置规则信息
            const current = versionList.value.find((v) => v.is_current);
            if (current) {
                ruleInfo.name = current.name;
                ruleInfo.node_id = current.node_id;
                ruleInfo.file_path = current.file_path;
                ruleInfo.version = current.version;
                ruleInfo.updated_at = current.created_at;
                currentRuleContent.value = current.file_content || "";
            }

            // 默认选中当前版本
            if (current) {
                selectedVersion.value = current;
            }
            // 如果有历史版本，默认选择最新的历史版本进行对比
            const firstHistory = versionList.value.find((v) => !v.is_current);
            if (firstHistory) {
                selectedVersion.value = firstHistory;
            }
        }
    } catch (err) {
        ElMessage.error("获取版本历史失败");
        console.error(err);
    }
};

// 选择版本
const handleVersionSelect = (data) => {
    if (data.version !== undefined) {
        selectedVersion.value = data;
    }
};

// 回滚
const handleRollback = () => {
    rollbackForm.comment = "";
    rollbackDialogVisible.value = true;
};

// 确认回滚
const confirmRollback = async () => {
    rollbackLoading.value = true;
    try {
        // 强制转换为数字，确保JSON中是数字类型而非字符串
        const ruleId = parseInt(effectiveRuleId.value, 10);
        const version = parseInt(selectedVersion.value.version, 10);

        if (isNaN(ruleId) || isNaN(version)) {
            ElMessage.error("无效的规则ID或版本号");
            rollbackLoading.value = false;
            return;
        }

        await axios.post(
            "/api/rule/rollback",
            {
                id: ruleId,
                version: version,
                comment: rollbackForm.comment || `Rollback to v${version}`,
            },
            {
                headers: getAuthHeaders(),
            },
        );
        ElMessage.success("回滚成功");
        rollbackDialogVisible.value = false;
        // 刷新版本列表
        fetchVersions();
    } catch (err) {
        ElMessage.error(err.response?.data?.error || "回滚失败");
    } finally {
        rollbackLoading.value = false;
    }
};

onMounted(() => {
    if (effectiveRuleId.value) {
        fetchVersions();
    } else {
        // 未拿到 ruleId，提示或静默处理
        ElMessage.error("无法识别规则ID，请从列表页进入");
    }
});
</script>

<style scoped>
.rule-version-container {
    padding: 20px;
}

.panel {
    border: none;
    border-radius: 12px;
    background: #fff;
    display: flex;
    flex-direction: column;
    height: 65vh;
    box-shadow: 0 2px 12px rgba(0, 0, 0, 0.08);
}

.panel-header {
    padding: 16px 20px;
    border-bottom: 1px solid #f0f0f0;
    font-weight: 600;
    background: linear-gradient(135deg, #f0f5ff 0%, #e6f0ff 100%);
    color: #1890ff;
    border-radius: 12px 12px 0 0;
}

.panel-header-actions {
    display: flex;
    justify-content: space-between;
    align-items: center;
}

.panel-body {
    padding: 16px;
    overflow: auto;
    flex: 1;
}

.placeholder {
    display: flex;
    align-items: center;
    justify-content: center;
    color: #909399;
}

.version-layout {
    display: flex;
    gap: 16px;
}

.sidebar {
    width: 300px;
    flex: 0 0 300px;
}

.main {
    flex: 1;
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

:deep(.el-button--danger) {
    background: linear-gradient(135deg, #ff4d4f 0%, #cf1322 100%);
    border: none;
    border-radius: 8px;
    transition: all 0.3s ease;
}

:deep(.el-tag) {
    border-radius: 6px;
    padding: 4px 12px;
    font-weight: 500;
}
</style>