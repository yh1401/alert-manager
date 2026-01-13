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

                <el-form-item label="规则内容" prop="file_content">
                    <!-- 左右分栏布局 -->
                    <div class="editor-preview-layout">
                        <!-- 左侧：编辑器 -->
                        <div class="editor-section">
                            <div class="section-title">
                                <span>规则内容编辑</span>
                                <div
                                    class="validation-indicator"
                                    v-if="validationResult"
                                >
                                    <el-icon
                                        v-if="validationResult.success"
                                        class="success-icon"
                                        ><CircleCheck
                                    /></el-icon>
                                    <el-icon v-else class="error-icon"
                                        ><CircleClose
                                    /></el-icon>
                                </div>
                            </div>

                            <textarea
                                v-model="ruleForm.file_content"
                                class="yaml-editor"
                                placeholder="输入 YAML 格式的 Prometheus 告警规则..."
                                @input="onContentChange"
                            ></textarea>

                            <div
                                class="validation-panel"
                                v-if="validationResult"
                            >
                                <div
                                    class="validation-status"
                                    :class="
                                        validationResult.success
                                            ? 'success'
                                            : 'error'
                                    "
                                >
                                    <el-icon v-if="validationResult.success"
                                        ><CircleCheck
                                    /></el-icon>
                                    <el-icon v-else><CircleClose /></el-icon>
                                    <span>{{
                                        validationResult.success
                                            ? "✅ 语法验证通过"
                                            : "❌ 语法验证失败"
                                    }}</span>
                                </div>
                                <pre
                                    v-if="validationResult.output"
                                    class="validation-output"
                                    >{{ validationResult.output }}</pre
                                >
                            </div>

                            <div class="validation-loading" v-if="isValidating">
                                <el-icon class="is-loading"
                                    ><Loading
                                /></el-icon>
                                <span>验证中...</span>
                            </div>
                        </div>

                        <!-- 右侧：预览面板 -->
                        <div class="preview-section">
                            <div class="section-title">规则预览</div>
                            <div class="preview-panel">
                                <div
                                    v-if="parsedRules.length > 0"
                                    class="preview-content"
                                >
                                    <div
                                        v-for="(group, idx) in parsedRules"
                                        :key="idx"
                                        class="rule-group-preview"
                                    >
                                        <h5>
                                            {{ group.name || "未命名分组" }}
                                        </h5>
                                        <div
                                            v-for="(rule, rIdx) in group.rules"
                                            :key="rIdx"
                                            class="rule-item-preview"
                                        >
                                            <div class="rule-alert-name">
                                                <el-tag
                                                    type="warning"
                                                    size="small"
                                                    >{{ rule.alert }}</el-tag
                                                >
                                            </div>
                                            <div class="rule-expr">
                                                <strong>条件:</strong
                                                ><br /><code>{{
                                                    rule.expr
                                                }}</code>
                                            </div>
                                            <div
                                                class="rule-for"
                                                v-if="rule.for"
                                            >
                                                <strong>持续:</strong>
                                                {{ rule.for }}
                                            </div>
                                            <div
                                                class="rule-labels"
                                                v-if="rule.labels"
                                            >
                                                <strong>标签:</strong><br />
                                                <el-tag
                                                    v-for="(
                                                        val, key
                                                    ) in rule.labels"
                                                    :key="key"
                                                    size="small"
                                                    style="
                                                        margin-top: 4px;
                                                        margin-right: 4px;
                                                    "
                                                    >{{ key }}:
                                                    {{ val }}</el-tag
                                                >
                                            </div>
                                        </div>
                                    </div>
                                </div>
                                <el-empty
                                    v-else
                                    description="暂无规则预览"
                                    :image-size="60"
                                ></el-empty>
                            </div>
                        </div>
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
import { CircleCheck, CircleClose, Loading } from "@element-plus/icons-vue";
import axios from "axios";
import jsyaml from "js-yaml";
import {
    loadUserPermissions,
    hasPermission,
    checkIsAdmin,
} from "../utils/permissions";

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
const isEditMode = computed(() => !!route.params.id);
const ruleId = computed(() => route.params.id);
const nodeOptions = ref([]);

const ruleForm = reactive({
    name: "",
    node_id: "",
    file_path: "",
    file_content: "",
    comment: "",
    base_version: "",
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

let validateTimer = null;

// 获取认证头
const getAuthHeaders = () => {
    return { Authorization: localStorage.getItem("token") };
};

// 加载用户权限、节点列表并在编辑模式下加载规则数据
onMounted(async () => {
    // 先加载权限（用于筛选节点下拉）
    await loadUserPermissions();

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
                ruleForm.base_version = rule.version;
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
    if (!content || !content.trim()) {
        parsedRules.value = [];
        return;
    }

    try {
        const parsed = jsyaml.load(content);
        if (parsed && parsed.groups) {
            parsedRules.value = parsed.groups;
        } else if (Array.isArray(parsed)) {
            parsedRules.value = parsed;
        } else {
            parsedRules.value = [];
        }
    } catch (e) {
        parsedRules.value = [];
    }
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

.editor-section,
.preview-section {
    display: flex;
    flex-direction: column;
    gap: 12px;
    min-height: 550px;
}

.section-title {
    display: flex;
    justify-content: space-between;
    align-items: center;
    font-size: 16px;
    font-weight: 600;
    color: #303133;
    padding: 0 0 8px 0;
    border-bottom: 2px solid #1890ff;
}

.validation-indicator {
    font-size: 20px;
}

.validation-indicator .success-icon {
    color: #67c23a;
}

.validation-indicator .error-icon {
    color: #f56c6c;
}

.editor-wrapper {
    position: relative;
    width: 100%;
}

.yaml-editor {
    width: 100%;
    min-height: 450px;
    font-family: "Monaco", "Menlo", "Consolas", monospace;
    font-size: 14px;
    line-height: 1.6;
    padding: 16px;
    border: 1px solid #dcdfe6;
    border-radius: 8px;
    resize: vertical;
    background: #f8f9fa;
    transition: all 0.3s;
    box-sizing: border-box;
}

.yaml-editor:focus {
    outline: none;
    border-color: #1890ff;
    box-shadow: 0 0 0 2px rgba(24, 144, 255, 0.1);
}

.validation-panel {
    margin-top: 12px;
    padding: 12px;
    border-radius: 8px;
    background: #fff;
    border: 1px solid #e4e7ed;
}

.validation-status {
    display: flex;
    align-items: center;
    gap: 8px;
    font-weight: 600;
    margin-bottom: 8px;
}

.validation-status.success {
    color: #67c23a;
}

.validation-status.error {
    color: #f56c6c;
}

.validation-output {
    font-family: "Monaco", "Menlo", "Consolas", monospace;
    font-size: 12px;
    background: #f8f9fa;
    padding: 12px;
    border-radius: 6px;
    margin: 0;
    white-space: pre-wrap;
    word-wrap: break-word;
    max-height: 200px;
    overflow-y: auto;
}

.validation-loading {
    display: flex;
    align-items: center;
    gap: 8px;
    margin-top: 12px;
    color: #909399;
    font-size: 14px;
}

.preview-section {
    position: relative;
}

.preview-panel {
    padding: 16px;
    background: #f8f9fa;
    border-radius: 8px;
    max-height: 500px;
    overflow-y: auto;
    border: 1px solid #e4e7ed;
    flex: 1;
}

.preview-content {
    display: flex;
    flex-direction: column;
    gap: 12px;
}

.rule-group-preview {
    padding: 12px;
    background: white;
    border-radius: 8px;
    box-shadow: 0 2px 8px rgba(0, 0, 0, 0.05);
}

.rule-group-preview h5 {
    margin: 0 0 10px 0;
    color: #1890ff;
    font-size: 14px;
    font-weight: 600;
}

.rule-item-preview {
    padding: 10px;
    margin-bottom: 8px;
    background: #fafafa;
    border-left: 3px solid #1890ff;
    border-radius: 4px;
    font-size: 13px;
}

.rule-item-preview:last-child {
    margin-bottom: 0;
}

.rule-alert-name {
    margin-bottom: 6px;
}

.rule-expr {
    margin-bottom: 6px;
    font-size: 13px;
    line-height: 1.4;
}

.rule-expr code {
    background: #e6f7ff;
    padding: 2px 4px;
    border-radius: 3px;
    font-family: "Monaco", "Menlo", "Consolas", monospace;
    font-size: 12px;
    word-break: break-all;
}

.rule-for,
.rule-labels {
    margin-top: 6px;
    font-size: 13px;
    color: #606266;
}

.action-buttons {
    margin-top: 24px;
    text-align: right;
    padding-top: 16px;
    border-top: 1px solid #e4e7ed;
}

.full-width {
    grid-column: 1 / -1;
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

    .preview-section {
        position: static;
    }

    .yaml-editor {
        min-height: 400px;
    }

    .preview-panel {
        max-height: 400px;
    }

    .editor-section,
    .preview-section {
        min-height: auto;
    }
}
</style>
