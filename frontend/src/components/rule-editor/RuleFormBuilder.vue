<template>
    <div class="form-builder">
        <div class="section-title">表单化规则配置</div>

        <el-alert
            v-if="isEditMode && existingGroups.length > 0"
            type="info"
            show-icon
            :closable="false"
            class="mode-alert"
        >
            当前文件已有 {{ existingGroups.length }} 个规则组，表单模式会在选定规则组中追加新告警；切到 YAML 模式可编辑完整文件。
        </el-alert>

        <el-form
            :model="form"
            :rules="rules"
            ref="builderFormRef"
            label-width="110px"
            class="builder-form"
        >
            <el-form-item label="规则组" prop="groupName">
                <el-select
                    v-if="existingGroups.length > 0"
                    v-model="groupMode"
                    class="group-mode"
                >
                    <el-option label="追加到已有规则组" value="existing" />
                    <el-option label="新建规则组" value="new" />
                </el-select>
                <el-select
                    v-if="groupMode === 'existing' && existingGroups.length > 0"
                    v-model="form.groupName"
                    filterable
                    placeholder="选择规则组"
                >
                    <el-option
                        v-for="group in existingGroups"
                        :key="group.name"
                        :label="group.name"
                        :value="group.name"
                    />
                </el-select>
                <el-input
                    v-else
                    v-model="form.groupName"
                    placeholder="例如 host.rules"
                />
            </el-form-item>

            <el-form-item label="评估间隔">
                <el-input
                    v-model="form.groupInterval"
                    placeholder="可选，例如 30s、1m"
                />
            </el-form-item>

            <el-divider content-position="left">告警条件</el-divider>

            <el-form-item label="告警名称" prop="alert">
                <el-input
                    v-model="form.alert"
                    placeholder="例如 HighCPUUsage"
                />
            </el-form-item>

            <el-form-item label="PromQL" prop="expr">
                <el-input
                    v-model="form.expr"
                    type="textarea"
                    :rows="4"
                    placeholder="例如 avg(rate(node_cpu_seconds_total{mode!='idle'}[5m])) by (instance) > 0.8"
                />
            </el-form-item>

            <el-form-item label="持续时间" prop="duration">
                <el-input
                    v-model="form.duration"
                    placeholder="可选，例如 5m、1h"
                />
            </el-form-item>

            <el-divider content-position="left">标签</el-divider>

            <div class="kv-list">
                <div
                    v-for="(item, index) in form.labels"
                    :key="item.id"
                    class="kv-row"
                >
                    <el-input v-model="item.key" placeholder="key" />
                    <el-input v-model="item.value" placeholder="value" />
                    <el-button
                        :icon="Delete"
                        circle
                        plain
                        @click="removeLabel(index)"
                    />
                </div>
                <el-button :icon="Plus" plain @click="addLabel">添加标签</el-button>
            </div>

            <el-divider content-position="left">注解</el-divider>

            <el-form-item label="摘要">
                <el-input
                    v-model="form.summary"
                    placeholder="例如 {{ $labels.instance }} CPU 使用率过高"
                />
            </el-form-item>

            <el-form-item label="描述">
                <el-input
                    v-model="form.description"
                    type="textarea"
                    :rows="3"
                    placeholder="例如 当前值为 {{ $value }}，请尽快处理"
                />
            </el-form-item>

            <div class="kv-list">
                <div
                    v-for="(item, index) in form.annotations"
                    :key="item.id"
                    class="kv-row"
                >
                    <el-input v-model="item.key" placeholder="annotation key" />
                    <el-input v-model="item.value" placeholder="annotation value" />
                    <el-button
                        :icon="Delete"
                        circle
                        plain
                        @click="removeAnnotation(index)"
                    />
                </div>
                <el-button :icon="Plus" plain @click="addAnnotation">添加注解</el-button>
            </div>
        </el-form>
    </div>
</template>

<script setup>
import { computed, reactive, ref, watch } from "vue";
import { Delete, Plus } from "@element-plus/icons-vue";

const props = defineProps({
    existingGroups: {
        type: Array,
        default: () => [],
    },
    isEditMode: {
        type: Boolean,
        default: false,
    },
});

const emit = defineEmits(["change"]);

const builderFormRef = ref(null);
const groupMode = ref("new");
let nextID = 1;

const form = reactive({
    groupName: "default.rules",
    groupInterval: "",
    alert: "",
    expr: "",
    duration: "5m",
    labels: [{ id: nextID++, key: "severity", value: "warning" }],
    summary: "",
    description: "",
    annotations: [],
});

const durationPattern = /^$|^\d+(ms|s|m|h|d|w|y)$/;
const labelKeyPattern = /^[a-zA-Z_][a-zA-Z0-9_]*$/;

const validateDuration = (_rule, value, callback) => {
    if (!durationPattern.test(value || "")) {
        callback(new Error("格式示例：30s、5m、1h"));
        return;
    }
    callback();
};

const rules = {
    groupName: [{ required: true, message: "请输入规则组名称", trigger: "blur" }],
    alert: [{ required: true, message: "请输入告警名称", trigger: "blur" }],
    expr: [{ required: true, message: "请输入 PromQL 表达式", trigger: "blur" }],
    duration: [{ validator: validateDuration, trigger: "blur" }],
};

const cleanObject = (rows) => {
    return rows.reduce((acc, row) => {
        const key = (row.key || "").trim();
        const value = (row.value || "").trim();
        if (key && value) {
            acc[key] = value;
        }
        return acc;
    }, {});
};

const builtRule = computed(() => {
    const labels = cleanObject(form.labels);
    const annotations = cleanObject(form.annotations);

    if (form.summary.trim()) {
        annotations.summary = form.summary.trim();
    }
    if (form.description.trim()) {
        annotations.description = form.description.trim();
    }

    const rule = {
        alert: form.alert.trim(),
        expr: form.expr.trim(),
    };

    if (form.duration.trim()) {
        rule.for = form.duration.trim();
    }
    if (Object.keys(labels).length > 0) {
        rule.labels = labels;
    }
    if (Object.keys(annotations).length > 0) {
        rule.annotations = annotations;
    }

    return rule;
});

const invalidKeys = computed(() => {
    return [...form.labels, ...form.annotations]
        .map((item) => (item.key || "").trim())
        .filter((key) => key && !labelKeyPattern.test(key));
});

const addLabel = () => {
    form.labels.push({ id: nextID++, key: "", value: "" });
};

const removeLabel = (index) => {
    form.labels.splice(index, 1);
};

const addAnnotation = () => {
    form.annotations.push({ id: nextID++, key: "", value: "" });
};

const removeAnnotation = (index) => {
    form.annotations.splice(index, 1);
};

const buildPayload = () => {
    return {
        groupName: form.groupName.trim(),
        groupInterval: form.groupInterval.trim(),
        rule: builtRule.value,
        invalidKeys: invalidKeys.value,
    };
};

watch(
    () => props.existingGroups,
    (groups) => {
        if (groups.length > 0 && props.isEditMode) {
            groupMode.value = "existing";
            form.groupName = groups[0].name || "default.rules";
        }
    },
    { immediate: true },
);

watch(
    groupMode,
    (mode) => {
        if (mode === "existing" && props.existingGroups.length > 0) {
            form.groupName = props.existingGroups[0].name || "";
        } else if (!form.groupName) {
            form.groupName = "default.rules";
        }
    },
);

watch(
    form,
    () => {
        emit("change", buildPayload());
    },
    { deep: true, immediate: true },
);

defineExpose({
    validate: async () => {
        await builderFormRef.value.validate();
        const payload = buildPayload();
        if (payload.invalidKeys.length > 0) {
            throw new Error(`标签或注解 key 不合法: ${payload.invalidKeys.join(", ")}`);
        }
        return payload;
    },
});
</script>

<style scoped>
.form-builder {
    display: flex;
    flex-direction: column;
    gap: 12px;
    min-height: 550px;
}

.section-title {
    font-size: 16px;
    font-weight: 600;
    color: #303133;
    padding: 0 0 8px 0;
    border-bottom: 2px solid #1890ff;
}

.mode-alert {
    margin-bottom: 4px;
}

.builder-form {
    padding: 16px;
    border: 1px solid #e4e7ed;
    border-radius: 8px;
    background: #f8f9fa;
}

.group-mode {
    margin-bottom: 8px;
    width: 180px;
}

.kv-list {
    display: flex;
    flex-direction: column;
    gap: 8px;
    margin: 0 0 18px 110px;
}

.kv-row {
    display: grid;
    grid-template-columns: minmax(120px, 0.8fr) minmax(160px, 1.2fr) 36px;
    gap: 8px;
    align-items: center;
}

@media (max-width: 900px) {
    .kv-list {
        margin-left: 0;
    }

    .kv-row {
        grid-template-columns: 1fr;
    }
}
</style>
