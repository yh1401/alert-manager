<template>
    <div class="rule-editor-container">
        <el-card class="editor-card">
            <template #header>
                <div class="card-header">
                    <span class="title">{{
                        isEditMode ? "编辑规则" : "新建规则"
                    }}</span>
                    <el-button @click="goBack" type="info" plain size="small"
                        >返回列表</el-button
                    >
                </div>
            </template>

            <el-form
                :model="ruleForm"
                :rules="rules"
                ref="ruleFormRef"
                label-width="100px"
            >
                <el-form-item label="规则名称" prop="name">
                    <el-input
                        v-model="ruleForm.name"
                        placeholder="输入规则名称"
                    ></el-input>
                </el-form-item>

                <el-form-item label="所属节点" prop="node_id">
                    <el-select
                        v-model="ruleForm.node_id"
                        placeholder="选择节点"
                        filterable
                        clearable
                    >
                        <el-option
                            v-for="n in nodeOptions"
                            :key="n.id"
                            :label="`${n.name} (${n.ip_address || '-'})`"
                            :value="n.id"
                        />
                    </el-select>
                </el-form-item>

                <el-form-item label="文件路径" prop="file_path">
                    <el-input
                        v-model="ruleForm.file_path"
                        placeholder="输入绝对路径，例如 /etc/vmalert/rules.yaml"
                    ></el-input>
                </el-form-item>

                <el-form-item label="标签" prop="tags">
                    <el-select
                        v-model="ruleForm.tags"
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
                </el-form-item>

                <el-form-item label="编辑模式">
                    <el-radio-group v-model="editorMode" @change="onEditorModeChange">
                        <el-radio-button label="form">表单模式</el-radio-button>
                        <el-radio-button label="yaml">YAML 模式</el-radio-button>
                    </el-radio-group>
                </el-form-item>

                <el-form-item label="规则内容" prop="file_content">
                    <!-- 左右分栏布局 -->
                    <div class="editor-preview-layout">
                        <!-- 左侧：表单构建器 / YAML 编辑器 -->
                        <RuleFormBuilder
                            v-if="editorMode === 'form'"
                            ref="ruleBuilderRef"
                            :existing-groups="sourceGroups"
                            :is-edit-mode="isEditMode"
                            @change="onBuilderChange"
                        />
                        <YamlEditor
                            v-else
                            v-model="ruleForm.file_content"
                            :validation-result="validationResult"
                            :is-validating="isValidating"
                            @change="onContentChange"
                        />

                        <!-- 右侧：预览面板 -->
                        <RulePreview :groups="parsedRules" />
                    </div>
                </el-form-item>

                <el-form-item label="版本说明" prop="comment" v-if="isEditMode">
                    <el-input
                        v-model="ruleForm.comment"
                        type="textarea"
                        :rows="2"
                        placeholder="描述本次修改的内容（可选）"
                    ></el-input>
                </el-form-item>
            </el-form>

            <div class="action-buttons">
                <el-button
                    type="primary"
                    @click="handleSave"
                    :loading="isSaving"
                    :disabled="isSaving || isValidating"
                    :title="
                        isValidating ? '规则语法正在校验中，保存已禁用' : ''
                    "
                >
                    {{
                        isEditMode
                            ? isValidating
                                ? "验证中..."
                                : "保存修改"
                            : isValidating
                              ? "验证中..."
                              : "创建规则"
                    }}
                </el-button>
                <el-button @click="goBack" :disabled="isSaving || isValidating"
                    >取消</el-button
                >
            </div>
        </el-card>
    </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted } from "vue";
import { useRoute, useRouter } from "vue-router";
import { ElMessage } from "element-plus";
import axios from "axios";
import jsyaml from "js-yaml";
import {
    loadUserPermissions,
    hasPermission,
    checkIsAdmin,
} from "../utils/permissions";
import YamlEditor from "@/components/rule-editor/YamlEditor.vue";
import RulePreview from "@/components/rule-editor/RulePreview.vue";
import RuleFormBuilder from "@/components/rule-editor/RuleFormBuilder.vue";

const validateFilePath = (_rule, value, callback) => {
    if (!value) return callback();
    if (!value.startsWith("/")) {
        return callback(new Error("文件路径必须是绝对路径"));
    }
    if (value.includes("..")) {
        return callback(new Error("文件路径不能包含 .."));
    }
    return callback();
};

const route = useRoute();
const router = useRouter();

const ruleFormRef = ref(null);
const ruleBuilderRef = ref(null);
const isEditMode = computed(() => !!route.params.id);
const ruleId = computed(() => route.params.id);
const nodeOptions = ref([]);
const editorMode = ref("form");
const formBuilderBaseContent = ref("");
const latestBuilderPayload = ref(null);

const ruleForm = reactive({
    name: "",
    node_id: "",
    file_path: "",
    file_content: "",
    comment: "",
    base_version: "",
    tags: [],
});

const rules = {
    name: [{ required: true, message: "请输入规则名称", trigger: "blur" }],
    node_id: [{ required: true, message: "请选择节点", trigger: "change" }],
    file_path: [
        { required: true, message: "请输入文件路径", trigger: "blur" },
        { validator: validateFilePath, trigger: "blur" },
    ],
    file_content: [
        { required: true, message: "请输入规则内容", trigger: "blur" },
    ],
};

const validationResult = ref(null);
const isValidating = ref(false);
const isSaving = ref(false);
const parsedRules = ref([]);

const parseGroupsFromYaml = (content) => {
    if (!content || !content.trim()) {
        return [];
    }

    try {
        const parsed = jsyaml.load(content);
        if (parsed && Array.isArray(parsed.groups)) {
            return parsed.groups;
        }
        if (Array.isArray(parsed)) {
            return parsed;
        }
        return [];
    } catch {
        return [];
    }
};

const sourceGroups = computed(() => parseGroupsFromYaml(formBuilderBaseContent.value));

const allTags = ref([]);
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

let validateTimer = null;

// 获取认证头
const getAuthHeaders = () => {
    return { Authorization: localStorage.getItem("token") };
};

// 加载用户权限、节点列表并在编辑模式下加载规则数据
onMounted(async () => {
    // 先加载权限（用于筛选节点下拉）
    await loadUserPermissions();
    await fetchTags();

    // 拉取节点列表，填充 nodeOptions（仅展示可写节点或管理员可见）
    try {
        const res = await axios.get("/api/agent/nodes", {
            headers: getAuthHeaders(),
        });
        const nodes = res.data.data || [];
        // 仅展示当前用户可写的节点（或管理员展示所有）
        const admin = checkIsAdmin();
        nodeOptions.value = nodes.filter((n) => {
            if (admin) return true;
            try {
                return hasPermission("node", n.id, "write");
            } catch {
                return false;
            }
        });
    } catch (err) {
        console.error("加载节点列表失败", err);
    }

    // 如果是编辑模式，加载规则详情到表单
    if (isEditMode.value) {
        try {
            const response = await axios.get("/api/rule/list", {
                headers: getAuthHeaders(),
            });
            const rule = response.data.data.find(
                (r) => r.id === parseInt(ruleId.value),
            );
            if (rule) {
                ruleForm.name = rule.name;
                // 兼容旧字段：如果后端返回 node_id/file_path 则使用，否则保持空
                // 优先使用后端返回的 node_id/file_path（可能为数字或字符串）
                ruleForm.node_id = rule.node_id || "";
                ruleForm.file_path = rule.file_path || "";
                ruleForm.file_content = rule.file_content;
                formBuilderBaseContent.value = rule.file_content || "";
                ruleForm.base_version = rule.version;
                ruleForm.tags = rule.tags || [];
                parseYamlContent(rule.file_content);

                // 确保当前规则所属节点出现在 nodeOptions 下拉中（如果用户无权限查看该节点也可编辑，但下拉需显示）
                try {
                    const nid = rule.node_id || rule.nodeId || null;
                    if (nid) {
                        const exist = nodeOptions.value.find(
                            (n) => String(n.id) === String(nid),
                        );
                        if (!exist) {
                            // 如果后端没返回节点的名字/IP，构造一个可识别的占位项
                            nodeOptions.value.push({
                                id: nid,
                                name: rule.node_name || `node-${nid}`,
                                ip_address: rule.ip_address || "",
                            });
                        }
                    }
                } catch (e) {
                    // 不影响主流程，静默处理
                    console.warn("确保节点选项存在时发生错误", e);
                }
            } else {
                ElMessage.error("规则不存在");
            }
        } catch (err) {
            ElMessage.error("加载失败");
        }
    }
});

const cloneGroups = (groups) => {
    return JSON.parse(JSON.stringify(groups || []));
};

const buildYamlFromBuilder = (payload) => {
    const groups = cloneGroups(sourceGroups.value);
    let group = groups.find((item) => item.name === payload.groupName);

    if (!group) {
        group = {
            name: payload.groupName,
            rules: [],
        };
        groups.push(group);
    }

    if (payload.groupInterval) {
        group.interval = payload.groupInterval;
    }

    if (!Array.isArray(group.rules)) {
        group.rules = [];
    }

    group.rules.push(payload.rule);

    return jsyaml.dump(
        { groups },
        {
            lineWidth: -1,
            noRefs: true,
            sortKeys: false,
        },
    );
};

const hasDuplicateAlertInSource = (payload) => {
    const group = sourceGroups.value.find((item) => item.name === payload.groupName);
    if (!group || !Array.isArray(group.rules)) {
        return false;
    }
    return group.rules.some((rule) => rule.alert === payload.rule.alert);
};

const syncBuilderContent = (payload, shouldValidate = true) => {
    latestBuilderPayload.value = payload;

    if (!payload.groupName || !payload.rule.alert || !payload.rule.expr) {
        ruleForm.file_content = isEditMode.value ? formBuilderBaseContent.value : "";
        parseYamlContent(ruleForm.file_content);
        validationResult.value = null;
        return;
    }

    ruleForm.file_content = buildYamlFromBuilder(payload);
    parseYamlContent(ruleForm.file_content);
    validationResult.value = null;

    if (shouldValidate) {
        onContentChange();
    }
};

const onBuilderChange = (payload) => {
    syncBuilderContent(payload);
};

const onEditorModeChange = (mode) => {
    validationResult.value = null;
    if (mode === "form" && latestBuilderPayload.value) {
        syncBuilderContent(latestBuilderPayload.value);
    } else {
        parseYamlContent(ruleForm.file_content);
    }
};

// 实时验证（带防抖）
const onContentChange = () => {
    clearTimeout(validateTimer);
    parseYamlContent(ruleForm.file_content);

    validateTimer = setTimeout(() => {
        if (ruleForm.file_content.trim()) {
            validateRuleContent();
        } else {
            validationResult.value = null;
        }
    }, 800);
};

// 调用后端验证接口
const validateRuleContent = async () => {
    isValidating.value = true;
    validationResult.value = null;

    try {
        const response = await axios.post(
            "/api/rule/validate_rule",
            {
                file_content: ruleForm.file_content,
            },
            {
                headers: getAuthHeaders(),
            },
        );

        // 后端返回200表示验证通过
        validationResult.value = {
            success: true,
            output:
                response.data.output ||
                response.data.message ||
                "✅ 规则语法正确",
        };
    } catch (error) {
        // 验证失败
        validationResult.value = {
            success: false,
            output:
                error.response?.data?.output ||
                error.response?.data?.error ||
                error.message,
        };
    } finally {
        isValidating.value = false;
    }
};

// 解析 YAML 用于预览
const parseYamlContent = (content) => {
    parsedRules.value = parseGroupsFromYaml(content);
};

// 保存规则
const handleSave = async () => {
    // 先校验表单字段
    try {
        await ruleFormRef.value.validate();
    } catch {
        ElMessage.warning("请填写完整的表单信息后再保存");
        return;
    }

    if (editorMode.value === "form") {
        try {
            const payload = await ruleBuilderRef.value.validate();
            if (hasDuplicateAlertInSource(payload)) {
                ElMessage.error("同一规则组下已存在相同告警名称，请修改告警名称或切到 YAML 模式手动处理");
                return;
            }
            syncBuilderContent(payload, false);
        } catch (err) {
            ElMessage.warning(err?.message || "请完善表单化规则配置");
            return;
        }
    }

    // 阻止在语法校验进行中提交
    if (isValidating.value) {
        ElMessage.warning("规则语法正在校验中，请稍候再保存");
        return;
    }

    // 确保规则语法已通过校验；若尚未校验或未通过，则触发一次校验并等待结果
    if (!validationResult.value || !validationResult.value.success) {
        try {
            await validateRuleContent();
        } catch (err) {
            // validateRuleContent 已会设置 validationResult / isValidating
        }
        if (!validationResult.value || !validationResult.value.success) {
            ElMessage.error("规则语法校验未通过，请根据验证面板修正后再保存");
            return;
        }
    }

    // 检查同一 node 下 file_path 是否重复（创建时禁止重复，编辑时允许更新自身）
    try {
        const res = await axios.get("/api/rule/list", {
            headers: getAuthHeaders(),
        });
        const existing = res.data.data || [];
        const normalize = (s) => (s || "").trim();
        const targetNode = parseInt(ruleForm.node_id);
        const targetPath = normalize(ruleForm.file_path);
        if (!targetNode || !targetPath) {
            ElMessage.warning("请填写并选择合法的节点与文件路径");
            return;
        }
        const dup = existing.find((r) => {
            const sameNode = parseInt(r.node_id) === targetNode;
            const samePath = normalize(r.file_path) === targetPath;
            const notSelf =
                !isEditMode.value || r.id !== parseInt(ruleId.value);
            return sameNode && samePath && notSelf;
        });
        if (dup) {
            ElMessage.error(
                `同一节点下已存在相同文件路径的规则 (rule id: ${dup.id})。请修改文件路径或编辑现有规则。`,
            );
            return;
        }
    } catch (err) {
        // 如果检查失败，不阻塞保存，但给出提示
        console.warn("检查重复规则失败，继续保存前请注意可能存在重复:", err);
    }

    // 标记保存中，防止重复提交
    isSaving.value = true;

    try {
        if (isEditMode.value) {
            // 更新规则（保留版本控制参数）
            await axios.post(
                "/api/rule/update_rule",
                {
                    id: parseInt(ruleId.value),
                    node_id: parseInt(ruleForm.node_id),
                    file_path: ruleForm.file_path,
                    name: ruleForm.name,
                    file_content: ruleForm.file_content,
                    comment: ruleForm.comment || "更新规则",
                    base_version: ruleForm.base_version,
                    tags: ruleForm.tags,
                },
                {
                    headers: getAuthHeaders(),
                },
            );
            ElMessage.success("规则更新成功");
        } else {
            // 创建新规则
            await axios.post(
                "/api/rule/create_rule",
                {
                    node_id: parseInt(ruleForm.node_id),
                    file_path: ruleForm.file_path,
                    name: ruleForm.name,
                    file_content: ruleForm.file_content,
                    tags: ruleForm.tags,
                },
                {
                    headers: getAuthHeaders(),
                },
            );
            ElMessage.success("规则创建成功");
        }

        router.push("/rules");
    } catch (error) {
        const msg =
            error?.response?.data?.error ||
            (error?.message ? `保存失败: ${error.message}` : "保存失败");
        ElMessage.error(msg);
    } finally {
        isSaving.value = false;
    }
};

const goBack = () => {
    router.push("/rules");
};
</script>

<style scoped>
.rule-editor-container {
    padding: 0;
    background: linear-gradient(135deg, #f5f7fa 0%, #c3cfe2 100%);
    min-height: 100%;
    width: 100%;
}

.editor-card {
    width: 100%;
    border-radius: 12px;
    box-shadow: 0 4px 16px rgba(0, 0, 0, 0.1);
    margin: 20px;
    box-sizing: border-box;
}

.card-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
}

.title {
    font-size: 20px;
    font-weight: 600;
    background: linear-gradient(135deg, #1890ff 0%, #096dd9 100%);
    -webkit-background-clip: text;
    -webkit-text-fill-color: transparent;
}

/* 左右分栏布局 */
.editor-preview-layout {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 20px;
    margin: 0;
    align-items: start;
    width: 100%;
}

.action-buttons {
    margin-top: 24px;
    text-align: right;
    padding-top: 16px;
    border-top: 1px solid #e4e7ed;
}

:deep(.el-form-item) {
    margin-bottom: 20px;
}

:deep(.el-form-item__content) {
    line-height: unset;
}

/* 响应式设计 */
@media (max-width: 1600px) {
    .editor-preview-layout {
        grid-template-columns: 1fr;
    }
}
</style>
