<template>
    <div class="prometheus-monitor">
        <!-- 健康状态与概览卡片 -->
        <el-row :gutter="16" class="overview-row">
            <el-col :span="6">
                <el-card shadow="hover" class="stat-card">
                    <div class="stat-card__header">
                        <el-icon :size="24" :color="healthColor"><Connection /></el-icon>
                        <span>Prometheus 状态</span>
                    </div>
                    <div class="stat-card__value">
                        <el-tag :type="healthTagType">{{ healthStatus }}</el-tag>
                    </div>
                    <div class="stat-card__desc">{{ healthDetail }}</div>
                </el-card>
            </el-col>
            <el-col :span="6">
                <el-card shadow="hover" class="stat-card">
                    <div class="stat-card__header">
                        <el-icon :size="24" color="#F56C6C"><Warning /></el-icon>
                        <span>活跃告警</span>
                    </div>
                    <div class="stat-card__value">{{ summary.firing_alerts || 0 }}</div>
                    <div class="stat-card__desc">待处理: {{ summary.pending_alerts || 0 }} | 总计: {{ summary.total_alerts || 0 }}</div>
                </el-card>
            </el-col>
            <el-col :span="6">
                <el-card shadow="hover" class="stat-card">
                    <div class="stat-card__header">
                        <el-icon :size="24" color="#E6A23C"><Document /></el-icon>
                        <span>规则评估</span>
                    </div>
                    <div class="stat-card__value">{{ summary.healthy_rules || 0 }} / {{ summary.total_rules || 0 }}</div>
                    <div class="stat-card__desc" :class="{ 'text-danger': summary.failed_rules > 0 }">
                        异常: {{ summary.failed_rules || 0 }}
                    </div>
                </el-card>
            </el-col>
            <el-col :span="6">
                <el-card shadow="hover" class="stat-card">
                    <div class="stat-card__header">
                        <el-icon :size="24" color="#67C23A"><Monitor /></el-icon>
                        <span>采集目标</span>
                    </div>
                    <div class="stat-card__value">{{ summary.up_targets || 0 }} / {{ summary.total_targets || 0 }}</div>
                    <div class="stat-card__desc" :class="{ 'text-danger': summary.down_targets > 0 }">
                        离线: {{ summary.down_targets || 0 }}
                    </div>
                </el-card>
            </el-col>
        </el-row>

        <!-- Tab 导航 -->
        <el-card shadow="hover" class="content-card">
            <el-tabs v-model="activeTab" @tab-change="handleTabChange">
                <!-- 告警列表 -->
                <el-tab-pane label="活跃告警" name="alerts">
                    <el-table :data="alerts" v-loading="loading.alerts" size="small" border>
                        <el-table-column prop="name" label="告警名称" min-width="160" />
                        <el-table-column prop="state" label="状态" width="100">
                            <template #default="{ row }">
                                <el-tag :type="row.state === 'firing' ? 'danger' : 'warning'" size="small">
                                    {{ row.state }}
                                </el-tag>
                            </template>
                        </el-table-column>
                        <el-table-column prop="severity" label="级别" width="100">
                            <template #default="{ row }">
                                <el-tag :type="severityTagType(row.severity)" size="small">{{ row.severity }}</el-tag>
                            </template>
                        </el-table-column>
                        <el-table-column label="实例" width="160">
                            <template #default="{ row }">{{ row.labels?.instance || '-' }}</template>
                        </el-table-column>
                        <el-table-column prop="value" label="当前值" width="100" />
                        <el-table-column label="描述" min-width="200" show-overflow-tooltip>
                            <template #default="{ row }">{{ row.annotations?.description || row.annotations?.summary || '-' }}</template>
                        </el-table-column>
                        <el-table-column prop="active_at" label="触发时间" width="180">
                            <template #default="{ row }">{{ formatDate(row.active_at) }}</template>
                        </el-table-column>
                    </el-table>
                </el-tab-pane>

                <!-- 规则评估状态 -->
                <el-tab-pane label="规则评估" name="rules">
                    <el-table :data="rules" v-loading="loading.rules" size="small" border>
                        <el-table-column prop="name" label="规则名称" min-width="160" />
                        <el-table-column prop="group" label="规则组" width="140" />
                        <el-table-column prop="health" label="健康" width="80">
                            <template #default="{ row }">
                                <el-tag :type="row.health === 'ok' ? 'success' : 'danger'" size="small">
                                    {{ row.health }}
                                </el-tag>
                            </template>
                        </el-table-column>
                        <el-table-column prop="query" label="PromQL" min-width="240" show-overflow-tooltip />
                        <el-table-column label="错误信息" min-width="180" show-overflow-tooltip>
                            <template #default="{ row }">{{ row.last_error || '-' }}</template>
                        </el-table-column>
                        <el-table-column prop="evaluation_time" label="评估耗时" width="100">
                            <template #default="{ row }">{{ row.evaluation_time }}s</template>
                        </el-table-column>
                        <el-table-column label="最后评估" width="180">
                            <template #default="{ row }">{{ formatDate(row.last_evaluation) }}</template>
                        </el-table-column>
                    </el-table>
                </el-tab-pane>

                <!-- 采集目标 -->
                <el-tab-pane label="采集目标" name="targets">
                    <el-table :data="targets" v-loading="loading.targets" size="small" border>
                        <el-table-column prop="job" label="任务" width="140" />
                        <el-table-column prop="instance" label="实例" width="160" />
                        <el-table-column prop="health" label="状态" width="80">
                            <template #default="{ row }">
                                <el-tag :type="row.health === 'up' ? 'success' : 'danger'" size="small">
                                    {{ row.health }}
                                </el-tag>
                            </template>
                        </el-table-column>
                        <el-table-column prop="scrape_url" label="采集地址" min-width="240" show-overflow-tooltip />
                        <el-table-column label="错误" min-width="160" show-overflow-tooltip>
                            <template #default="{ row }">{{ row.last_error || '-' }}</template>
                        </el-table-column>
                        <el-table-column prop="last_scrape_duration" label="耗时(s)" width="100" />
                        <el-table-column label="最后采集" width="180">
                            <template #default="{ row }">{{ formatDate(row.last_scrape) }}</template>
                        </el-table-column>
                    </el-table>
                </el-tab-pane>

                <!-- PromQL 查询 -->
                <el-tab-pane label="PromQL 查询" name="query">
                    <div class="query-section">
                        <el-input
                            v-model="promQL"
                            placeholder="输入 PromQL 查询，如 up, node_cpu_seconds_total, ..."
                            @keyup.enter="executeQuery"
                        >
                            <template #append>
                                <el-button @click="executeQuery" :loading="loading.query">查询</el-button>
                            </template>
                        </el-input>

                        <el-card v-if="queryResult" class="query-result-card" shadow="never">
                            <template #header>
                                <span>查询结果 ({{ queryResult.resultType }})</span>
                            </template>
                            <el-table :data="parseQueryResult(queryResult)" size="small" border max-height="400">
                                <el-table-column label="指标" min-width="300">
                                    <template #default="{ row }">
                                        <span class="mono">{{ formatMetric(row.metric) }}</span>
                                    </template>
                                </el-table-column>
                                <el-table-column label="值" width="200">
                                    <template #default="{ row }">{{ row.value }}</template>
                                </el-table-column>
                            </el-table>
                        </el-card>
                    </div>
                </el-tab-pane>
            </el-tabs>

            <div class="toolbar">
                <el-button size="small" @click="refresh" :loading="loading.overview">
                    <el-icon><Refresh /></el-icon> 刷新
                </el-button>
                <el-button size="small" @click="clearCache" type="warning">
                    清除缓存
                </el-button>
                <span class="last-update" v-if="lastUpdate">最后更新: {{ lastUpdate }}</span>
            </div>
        </el-card>
    </div>
</template>

<script setup>
import { ref, reactive, onMounted, computed } from "vue";
import axios from "axios";
import { ElMessage } from "element-plus";
import { Connection, Warning, Document, Monitor, Refresh } from "@element-plus/icons-vue";

const activeTab = ref("alerts");
const promQL = ref("up");
const queryResult = ref(null);
const lastUpdate = ref("");

const summary = ref({});
const alerts = ref([]);
const rules = ref([]);
const targets = ref([]);
const healthStatus = ref("检测中...");
const healthDetail = ref("");

const loading = reactive({
    overview: false,
    alerts: false,
    rules: false,
    targets: false,
    query: false,
});

const getAuthHeaders = () => ({ Authorization: localStorage.getItem("token") });

const healthColor = computed(() => healthStatus.value === "running" ? "#67C23A" : "#F56C6C");
const healthTagType = computed(() => healthStatus.value === "running" ? "success" : "danger");

const fetchHealth = async () => {
    try {
        const res = await axios.get("/api/prometheus/health", { headers: getAuthHeaders() });
        healthStatus.value = res.data.status?.includes("running") || res.data.status?.includes("up") ? "running" : res.data.status;
        healthDetail.value = "服务正常运行";
    } catch (err) {
        healthStatus.value = "unreachable";
        healthDetail.value = err.response?.data?.error || "无法连接到 Prometheus";
    }
};

const fetchOverview = async () => {
    loading.overview = true;
    try {
        const res = await axios.get("/api/prometheus/overview", { headers: getAuthHeaders() });
        const data = res.data?.data || {};
        summary.value = data.summary || {};
        alerts.value = data.alerts || [];
        rules.value = data.rules || [];
        targets.value = data.targets || [];
        lastUpdate.value = new Date().toLocaleString("zh-CN");
    } catch (err) {
        ElMessage.error(err.response?.data?.error || "获取概览失败");
    } finally {
        loading.overview = false;
    }
};

const fetchAlerts = async () => {
    loading.alerts = true;
    try {
        const res = await axios.get("/api/prometheus/alerts", { headers: getAuthHeaders() });
        alerts.value = res.data?.data || [];
    } catch (err) {
        ElMessage.error(err.response?.data?.error || "获取告警失败");
    } finally {
        loading.alerts = false;
    }
};

const fetchRules = async () => {
    loading.rules = true;
    try {
        const res = await axios.get("/api/prometheus/rules", { headers: getAuthHeaders() });
        rules.value = res.data?.data || [];
    } catch (err) {
        ElMessage.error(err.response?.data?.error || "获取规则失败");
    } finally {
        loading.rules = false;
    }
};

const fetchTargets = async () => {
    loading.targets = true;
    try {
        const res = await axios.get("/api/prometheus/targets", { headers: getAuthHeaders() });
        targets.value = res.data?.data || [];
    } catch (err) {
        ElMessage.error(err.response?.data?.error || "获取采集目标失败");
    } finally {
        loading.targets = false;
    }
};

const executeQuery = async () => {
    if (!promQL.value) {
        ElMessage.warning("请输入 PromQL 查询");
        return;
    }
    loading.query = true;
    try {
        const res = await axios.get("/api/prometheus/query", {
            params: { query: promQL.value },
            headers: getAuthHeaders(),
        });
        queryResult.value = res.data?.data;
    } catch (err) {
        ElMessage.error(err.response?.data?.error || "查询失败");
    } finally {
        loading.query = false;
    }
};

const clearCache = async () => {
    try {
        await axios.post("/api/prometheus/clear_cache", {}, { headers: getAuthHeaders() });
        ElMessage.success("缓存已清除");
        await fetchOverview();
    } catch (err) {
        ElMessage.error(err.response?.data?.error || "清除缓存失败");
    }
};

const refresh = () => {
    fetchHealth();
    fetchOverview();
};

const handleTabChange = (tab) => {
    if (tab === "alerts" && alerts.value.length === 0) fetchAlerts();
    if (tab === "rules" && rules.value.length === 0) fetchRules();
    if (tab === "targets" && targets.value.length === 0) fetchTargets();
};

// 辅助函数
const formatDate = (d) => d ? new Date(d).toLocaleString("zh-CN") : "-";
const severityTagType = (s) => s === "critical" ? "danger" : s === "warning" ? "warning" : "info";
const formatMetric = (m) => m ? JSON.stringify(m) : "-";
const parseQueryResult = (result) => {
    if (!result || !result.result) return [];
    try {
        const parsed = typeof result.result === "string" ? JSON.parse(result.result) : result.result;
        if (result.resultType === "vector") {
            return parsed.map(item => ({
                metric: item.metric,
                value: Array.isArray(item.value) ? item.value[1] : item.value,
            }));
        }
        if (result.resultType === "matrix") {
            return parsed.map(item => ({
                metric: item.metric,
                value: `共 ${item.values?.length || 0} 个数据点`,
            }));
        }
        if (result.resultType === "scalar") {
            return [{ metric: {}, value: parsed }];
        }
        return parsed;
    } catch {
        return [];
    }
};

onMounted(() => {
    fetchHealth();
    fetchOverview();
});
</script>

<style scoped>
.prometheus-monitor { display: flex; flex-direction: column; gap: 16px; }
.overview-row { width: 100%; }
.stat-card { border: none; box-shadow: 0 4px 12px rgba(0,0,0,0.08); border-radius: 12px; min-height: 140px; }
.stat-card__header { display: flex; align-items: center; gap: 8px; font-weight: 600; color: #303133; }
.stat-card__value { font-size: 28px; font-weight: 700; margin-top: 12px; color: #1f2d3d; }
.stat-card__desc { margin-top: 6px; color: #909399; font-size: 13px; }
.content-card { border: none; box-shadow: 0 4px 12px rgba(0,0,0,0.08); border-radius: 12px; }
.toolbar { display: flex; align-items: center; gap: 12px; margin-top: 12px; }
.last-update { color: #909399; font-size: 13px; }
.text-danger { color: #F56C6C; }
.mono { font-family: "SFMono-Regular", Consolas, "Liberation Mono", Menlo, monospace; font-size: 13px; }
.query-section { display: flex; flex-direction: column; gap: 16px; }
.query-result-card { margin-top: 16px; }
</style>
