<template>
    <div class="tab-content" v-if="user">
        <el-form :model="form" label-width="120px">
            <el-form-item label="用户">
                <el-tag type="success">{{ user.username }}</el-tag>
            </el-form-item>
            <el-form-item label="资源类型">
                <el-radio-group v-model="form.resource_type">
                    <el-radio label="rule">规则</el-radio>
                    <el-radio label="node">节点</el-radio>
                </el-radio-group>
            </el-form-item>
            <el-form-item label="资源列表">
                <el-select
                    v-model="form.resource_ids"
                    multiple
                    filterable
                    clearable
                    placeholder="请选择资源"
                    style="width: 100%"
                >
                    <el-option
                        v-for="item in resourceOptions"
                        :key="item.id"
                        :label="getOptionLabel(item)"
                        :value="item.id"
                    >
                        <div
                            v-if="form.resource_type === 'rule'"
                            class="option-layout"
                        >
                            <span
                                >{{ item.name }} ({{ item.file_path }})
                                <span class="option-extra-info"
                                    >[节点:
                                    {{
                                        nodeIdToNameMap[item.node_id] || "未知"
                                    }}]</span
                                ></span
                            >
                            <span class="option-id">ID: {{ item.id }}</span>
                        </div>
                        <div
                            v-if="form.resource_type === 'node'"
                            class="option-layout"
                        >
                            <span>
                                <span
                                    :class="[
                                        'status-dot',
                                        item.status === 'online'
                                            ? 'online'
                                            : 'offline',
                                    ]"
                                ></span>
                                {{ item.name }} ({{ item.ip_address }})
                                <span class="option-extra-info"
                                    >[规则数:
                                    {{
                                        nodeIdToRuleCountMap[item.id] || 0
                                    }}]</span
                                >
                            </span>
                            <span class="option-id">ID: {{ item.id }}</span>
                        </div>
                    </el-option>
                </el-select>
            </el-form-item>
            <el-form-item label="权限类型">
                <el-radio-group v-model="form.action">
                    <el-radio label="read">只读</el-radio>
                    <el-radio label="write">读写</el-radio>
                </el-radio-group>
            </el-form-item>
            <el-form-item>
                <el-button type="primary" @click="handleSubmit('set')"
                    >批量授权</el-button
                >
                <el-button type="danger" @click="handleSubmit('remove')"
                    >批量移除</el-button
                >
                <el-button @click="resetForm">重置</el-button>
            </el-form-item>
        </el-form>
    </div>
</template>

<script setup>
import {
    reactive,
    defineProps,
    defineEmits,
    watch,
    onMounted,
    ref,
    computed,
} from "vue";
import { ElMessage } from "element-plus";
import axios from "axios";

const props = defineProps({
    user: {
        type: Object,
        default: null,
    },
});

const emit = defineEmits(["submit-set", "submit-remove"]);

const form = reactive({
    resource_type: "rule",
    resource_ids: [],
    action: "read",
});

const rules = ref([]);
const nodes = ref([]);

const resourceOptions = computed(() => {
    return form.resource_type === "rule" ? rules.value : nodes.value;
});

const nodeIdToNameMap = computed(() => {
    return nodes.value.reduce((acc, node) => {
        acc[node.id] = node.name;
        return acc;
    }, {});
});

const nodeIdToRuleCountMap = computed(() => {
    return rules.value.reduce((acc, rule) => {
        acc[rule.node_id] = (acc[rule.node_id] || 0) + 1;
        return acc;
    }, {});
});

const getOptionLabel = (item) => {
    if (form.resource_type === "rule") {
        const nodeName = nodeIdToNameMap.value[item.node_id] || "未知";
        return `${item.name} (${item.file_path}) [节点: ${nodeName}]`;
    } else if (form.resource_type === "node") {
        const ruleCount = nodeIdToRuleCountMap.value[item.id] || 0;
        return `[${item.status}] ${item.name} (${item.ip_address}) [规则数: ${ruleCount}]`;
    }
    return item.id;
};

const fetchRules = async () => {
    try {
        const response = await axios.get("/api/rule/list");
        if (response.data && response.data.data) {
            rules.value = response.data.data;
        }
    } catch (error) {
        ElMessage.error("获取规则列表失败");
        console.error("Error fetching rules:", error);
    }
};

const fetchNodes = async () => {
    try {
        const response = await axios.get("/api/agent/nodes");
        if (response.data && response.data.data) {
            nodes.value = response.data.data;
        }
    } catch (error) {
        ElMessage.error("获取节点列表失败");
        console.error("Error fetching nodes:", error);
    }
};

onMounted(() => {
    fetchRules();
    fetchNodes();
});

const resetForm = () => {
    form.resource_type = "rule";
    form.resource_ids = [];
    form.action = "read";
};

// Reset form when user changes
watch(
    () => props.user,
    () => {
        resetForm();
    },
);

// Clear selected resources when resource type changes
watch(
    () => form.resource_type,
    () => {
        form.resource_ids = [];
    },
);

const handleSubmit = (type) => {
    if (!props.user) {
        ElMessage.error("未选择用户");
        return;
    }

    if (!form.resource_ids || form.resource_ids.length === 0) {
        ElMessage.error("请选择资源ID列表");
        return;
    }

    const payload = {
        user_id: props.user.id,
        resource_type: form.resource_type,
        action: form.action,
        resource_ids: form.resource_ids,
    };

    if (type === "set") {
        emit("submit-set", payload, resetForm);
    } else {
        emit("submit-remove", payload, resetForm);
    }
};
</script>

<style scoped>
.tab-content {
    padding: 20px;
    height: 100%;
    width: 100%;
    display: flex;
    flex-direction: column;
    overflow: auto;
}

.option-layout {
    display: flex;
    justify-content: space-between;
    align-items: center;
    width: 100%;
}

.option-id {
    color: #8492a6;
    font-size: 13px;
    margin-left: 10px;
    flex-shrink: 0;
}

.option-extra-info {
    color: #a8abb2;
    margin-left: 8px;
}

.status-dot {
    display: inline-block;
    width: 8px;
    height: 8px;
    border-radius: 50%;
    margin-right: 5px;
}
.status-dot.online {
    background-color: #67c23a; /* success */
}
.status-dot.offline {
    background-color: #909399; /* info */
}
</style>
