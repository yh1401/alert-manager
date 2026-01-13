<template>
  <div class="dashboard">
    <el-row :gutter="16" class="cards-row">
      <el-col :span="6">
        <el-card shadow="hover" class="stat-card">
          <div class="stat-card__header">
            <el-icon :size="24" color="#409EFF"><DataBoard /></el-icon>
            <span>总操作数</span>
          </div>
          <div class="stat-card__value">{{ totalActions }}</div>
          <div class="stat-card__desc">审计日志中的所有操作次数</div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover" class="stat-card">
          <div class="stat-card__header">
            <el-icon :size="24" color="#67C23A"><Plus /></el-icon>
            <span>规则数</span>
          </div>
          <div class="stat-card__value">{{ ruleCount }}</div>
          <div class="stat-card__desc">当前可用的规则条目</div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover" class="stat-card">
          <div class="stat-card__header">
            <el-icon :size="24" color="#E6A23C"><Monitor /></el-icon>
            <span>节点数</span>
          </div>
          <div class="stat-card__value">{{ nodeCount }}</div>
          <div class="stat-card__desc">已注册的监控节点</div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover" class="stat-card">
          <div class="stat-card__header">
            <el-icon :size="24" color="#F56C6C"><Histogram /></el-icon>
            <span>最近7天操作</span>
          </div>
          <div class="stat-card__value">{{ last7Days }}</div>
          <div class="stat-card__desc">最近一周的操作总数</div>
        </el-card>
      </el-col>
    </el-row>

    <el-row :gutter="16" class="chart-row">
      <el-col :span="12">
        <el-card shadow="hover" class="chart-card">
          <template #header>
            <div class="chart-title">
              <el-icon><PieChart /></el-icon>
              <span>操作类型分布</span>
            </div>
          </template>
          <div ref="actionPieRef" class="chart"></div>
        </el-card>
      </el-col>
      <el-col :span="12">
        <el-card shadow="hover" class="chart-card">
          <template #header>
            <div class="chart-title">
              <el-icon><TrendCharts /></el-icon>
              <span>最近7天趋势</span>
            </div>
          </template>
          <div ref="dailyLineRef" class="chart"></div>
        </el-card>
      </el-col>
    </el-row>

    <el-row :gutter="16" class="chart-row">
      <el-col :span="12">
        <el-card shadow="hover" class="chart-card">
          <template #header>
            <div class="chart-title">
              <el-icon><User /></el-icon>
              <span>Top 操作用户</span>
            </div>
          </template>
          <div ref="topUserBarRef" class="chart"></div>
        </el-card>
      </el-col>
      <el-col :span="12">
        <el-card shadow="hover" class="chart-card">
          <template #header>
            <div class="chart-title">
              <el-icon><Files /></el-icon>
              <span>资源类型分布</span>
            </div>
          </template>
          <div ref="resourcePieRef" class="chart"></div>
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted, onBeforeUnmount } from 'vue'
import axios from 'axios'
import * as echarts from 'echarts'
import {
  DataBoard,
  Plus,
  Monitor,
  Histogram,
  PieChart,
  TrendCharts,
  User,
  Files
} from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'

const stats = reactive({
  action_stats: [],
  resource_type_stats: [],
  daily_stats: [],
  top_users: []
})

const totalActions = ref(0)
const ruleCount = ref(0)
const nodeCount = ref(0)
const last7Days = ref(0)

const actionPieRef = ref(null)
const dailyLineRef = ref(null)
const topUserBarRef = ref(null)
const resourcePieRef = ref(null)

let charts = []

const initChart = (domRef) => {
  if (!domRef.value) return null
  const chart = echarts.init(domRef.value)
  charts.push(chart)
  return chart
}

const renderCharts = () => {
  // action pie
  const actionChart = initChart(actionPieRef)
  actionChart?.setOption({
    tooltip: { trigger: 'item' },
    legend: { bottom: 0 },
    series: [
      {
        name: '操作类型',
        type: 'pie',
        radius: '60%',
        data: stats.action_stats.map((s) => ({
          name: s.Action,
          value: s.Count
        }))
      }
    ]
  })

  // resource pie
  const resChart = initChart(resourcePieRef)
  resChart?.setOption({
    tooltip: { trigger: 'item' },
    legend: { bottom: 0 },
    series: [
      {
        name: '资源类型',
        type: 'pie',
        radius: '60%',
        data: stats.resource_type_stats.map((s) => ({
          name: s.Action || s.action || s.Resource_type || s.ResourceType || s.resource_type,
          value: s.Count
        }))
      }
    ]
  })

  // daily line
  const dailyChart = initChart(dailyLineRef)
  dailyChart?.setOption({
    tooltip: { trigger: 'axis' },
    xAxis: {
      type: 'category',
      data: stats.daily_stats.map((d) => d.Date || d.date)
    },
    yAxis: { type: 'value' },
    series: [
      {
        data: stats.daily_stats.map((d) => d.Count),
        type: 'line',
        areaStyle: {},
        smooth: true,
        lineStyle: { width: 2, color: '#409EFF' }
      }
    ]
  })

  // top user bar
  const userChart = initChart(topUserBarRef)
  userChart?.setOption({
    tooltip: { trigger: 'axis' },
    xAxis: {
      type: 'category',
      data: stats.top_users.map((u) => u.Username || u.username)
    },
    yAxis: { type: 'value' },
    series: [
      {
        data: stats.top_users.map((u) => u.Count),
        type: 'bar',
        itemStyle: { color: '#67C23A' },
        barWidth: '40%'
      }
    ]
  })
}

const fetchAuditStats = async () => {
  const res = await axios.get('/api/admin/audit/stats')
  const data = res.data?.data || {}
  stats.action_stats = data.action_stats || []
  stats.resource_type_stats = data.resource_type_stats || []
  stats.daily_stats = data.daily_stats || []
  stats.top_users = data.top_users || []
  totalActions.value = (stats.action_stats || []).reduce((acc, cur) => acc + (cur.Count || 0), 0)
  last7Days.value = (stats.daily_stats || []).reduce((acc, cur) => acc + (cur.Count || 0), 0)
}

const fetchRuleCount = async () => {
  const res = await axios.get('/api/rule/list')
  const list = res.data?.data || []
  ruleCount.value = list.length
}

const fetchNodeCount = async () => {
  const res = await axios.get('/api/agent/nodes')
  const list = res.data?.data || []
  nodeCount.value = list.length
}

const loadAll = async () => {
  try {
    await Promise.all([fetchAuditStats(), fetchRuleCount(), fetchNodeCount()])
    renderCharts()
  } catch (err) {
    console.error(err)
    ElMessage.error(err.response?.data?.error || '加载数据失败')
  }
}

const resizeHandler = () => {
  charts.forEach((c) => c?.resize())
}

onMounted(() => {
  loadAll()
  window.addEventListener('resize', resizeHandler)
})

onBeforeUnmount(() => {
  window.removeEventListener('resize', resizeHandler)
  charts.forEach((c) => c?.dispose())
  charts = []
})
</script>

<style scoped>
.dashboard {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.cards-row,
.chart-row {
  width: 100%;
}

.stat-card {
  border: none;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.08);
  border-radius: 12px;
  min-height: 140px;
}

.stat-card__header {
  display: flex;
  align-items: center;
  gap: 8px;
  font-weight: 600;
  color: #303133;
}

.stat-card__value {
  font-size: 32px;
  font-weight: 700;
  margin-top: 12px;
  color: #1f2d3d;
}

.stat-card__desc {
  margin-top: 6px;
  color: #909399;
  font-size: 13px;
}

.chart-card {
  border: none;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.08);
  border-radius: 12px;
  min-height: 360px;
}

.chart-title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-weight: 600;
  color: #303133;
}

.chart {
  width: 100%;
  height: 320px;
}
</style>
