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
            <el-form-item label="资源ID列表">
                <el-input v-model="form.resource_ids_str" placeholder="输入ID，用逗号分隔，如：1,2,3" />
            </el-form-item>
            <el-form-item label="权限类型">
                <el-radio-group v-model="form.action">
                    <el-radio label="read">只读</el-radio>
                    <el-radio label="write">读写</el-radio>
                </el-radio-group>
            </el-form-item>
            <el-form-item>
                <el-button type="primary" @click="handleSubmit('set')">批量授权</el-button>
                <el-button type="danger" @click="handleSubmit('remove')">批量移除</el-button>
                <el-button @click="resetForm">重置</el-button>
            </el-form-item>
        </el-form>
    </div>
</template>

<script setup>
import { reactive, defineProps, defineEmits, watch } from 'vue';
import { ElMessage } from 'element-plus';

const props = defineProps({
    user: {
        type: Object,
        default: null,
    },
});

const emit = defineEmits(['submit-set', 'submit-remove']);

const form = reactive({
    resource_type: "rule",
    resource_ids_str: "",
    action: "read",
});

const resetForm = () => {
    form.resource_type = "rule";
    form.resource_ids_str = "";
    form.action = "read";
};

// Reset form when user changes
watch(() => props.user, () => {
    resetForm();
});

const handleSubmit = (type) => {
    if (!props.user) {
        ElMessage.error("未选择用户");
        return;
    }

    if (!form.resource_ids_str.trim()) {
        ElMessage.error("请输入资源ID列表");
        return;
    }

    const resource_ids = form.resource_ids_str
        .split(",")
        .map((id) => parseInt(id.trim()))
        .filter((id) => !isNaN(id));

    if (resource_ids.length === 0) {
        ElMessage.error("请输入有效的资源ID");
        return;
    }

    const payload = {
        user_id: props.user.id,
        resource_type: form.resource_type,
        action: form.action,
        resource_ids: resource_ids,
    };

    if (type === 'set') {
        emit('submit-set', payload, resetForm);
    } else {
        emit('submit-remove', payload, resetForm);
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
</style>
