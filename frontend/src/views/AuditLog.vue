<template>
  <div class="audit-log-container">
    <el-card shadow="hover">
      <template #header>
        <div class="card-header">
          <h2>
            <el-icon><Document /></el-icon>
            审计日志
          </h2>
          <el-text type="info">管理员可查看所有用户对规则和节点的操作记录</el-text>
        </div>
      </template>

      <!-- 筛选区域 -->
      <el-form :inline="true" :model="filterForm" class="filter-form">
        <el-form-item label="资源类型">
          <el-select v-model="filterForm.resource_type" placeholder="全部" clearable style="width: 120px">
            <el-option label="规则" value="rule" />
            <el-option label="节点" value="node" />
          </el-select>
        </el-form-item>

        <el-form-item label="操作类型">
          <el-select v-model="filterForm.action" placeholder="全部" clearable style="width: 120px">
            <el-option label="创建" value="create" />
            <el-option label="更新" value="update" />
            <el-option label="删除" value="delete" />
            <el-option label="回滚" value="rollback" />
          </el-select>
        </el-form-item>

        <el-form-item label="操作人">
          <el-input v-model="filterForm.username" placeholder="用户名" clearable style="width: 150px" />
        </el-form-item>

        <el-form-item label="开始日期">
          <el-date-picker
            v-model="filterForm.start_date"
            type="date"
            placeholder="选择日期"
            format="YYYY-MM-DD"
            value-format="YYYY-MM-DD"
            style="width: 150px"
          />
        </el-form-item>

        <el-form-item label="结束日期">
          <el-date-picker
            v-model="filterForm.end_date"
            type="date"
            placeholder="选择日期"
            format="YYYY-MM-DD"
            value-format="YYYY-MM-DD"
            style="width: 150px"
          />
        </el-form-item>

        <el-form-item>
          <el-button type="primary" @click="handleSearch" :icon="Search">查询</el-button>
          <el-button @click="handleReset" :icon="Refresh">重置</el-button>
        </el-form-item>
      </el-form>

      <!-- 统计卡片 -->
      <div class="stats-cards" v-if="stats">
        <el-row :gutter="16">
          <el-col :span="6" v-for="(stat, index) in statsDisplay" :key="index">
            <el-card shadow="hover" class="stat-card">
              <div class="stat-content">
                <el-icon :size="32" :color="stat.color">
                  <component :is="stat.icon" />
                </el-icon>
                <div class="stat-info">
                  <div class="stat-value">{{ stat.value }}</div>
                  <div class="stat-label">{{ stat.label }}</div>
                </div>
              </div>
            </el-card>
          </el-col>
        </el-row>
      </div>

      <!-- 审计日志表格 -->
      <el-table
        :data="auditLogs"
        v-loading="loading"
        stripe
        border
        style="width: 100%; margin-top: 20px"
        :default-sort="{ prop: 'created_at', order: 'descending' }"
      >
        <el-table-column prop="id" label="ID" width="80" align="center" />

        <el-table-column prop="username" label="操作人" width="120" align="center">
          <template #default="{ row }">
            <el-tag v-if="row.user_id === 0" type="info" size="small">系统</el-tag>
            <el-tag v-else type="success" size="small">{{ row.username }}</el-tag>
          </template>
        </el-table-column>

        <el-table-column prop="resource_type" label="资源类型" width="100" align="center">
          <template #default="{ row }">
            <el-tag v-if="row.resource_type === 'rule'" type="primary" size="small">规则</el-tag>
            <el-tag v-else type="warning" size="small">节点</el-tag>
          </template>
        </el-table-column>

        <el-table-column prop="resource_name" label="资源名称" width="180" show-overflow-tooltip />

        <el-table-column prop="action" label="操作类型" width="100" align="center">
          <template #default="{ row }">
            <el-tag v-if="row.action === 'create'" type="success" size="small">创建</el-tag>
            <el-tag v-else-if="row.action === 'update'" type="primary" size="small">更新</el-tag>
            <el-tag v-else-if="row.action === 'delete'" type="danger" size="small">删除</el-tag>
            <el-tag v-else-if="row.action === 'rollback'" type="warning" size="small">回滚</el-tag>
            <el-tag v-else type="info" size="small">{{ row.action }}</el-tag>
          </template>
        </el-table-column>

        <el-table-column prop="description" label="操作描述" min-width="250" show-overflow-tooltip />

        <el-table-column prop="ip_address" label="IP地址" width="140" align="center" />

        <el-table-column prop="created_at" label="操作时间" width="180" align="center">
          <template #default="{ row }">
            {{ formatDateTime(row.created_at) }}
          </template>
        </el-table-column>

        <el-table-column label="操作" width="220" align="center" fixed="right">
          <template #default="{ row }">
            <el-button type="primary" size="small" @click="viewDetail(row)" :icon="View">
              查看详情
            </el-button>
            <el-button
              v-if="row.resource_type === 'rule' && row.action === 'delete'"
              type="warning"
              size="small"
              :icon="RefreshLeft"
              :loading="restoring"
              @click="restoreRule(row)"
            >
              恢复
            </el-button>
          </template>
        </el-table-column>
      </el-table>

      <!-- 分页 -->
      <el-pagination
        v-if="pagination.total > 0"
        style="margin-top: 20px; justify-content: center"
        :current-page="pagination.page"
        :page-size="pagination.page_size"
        :page-sizes="[10, 20, 50, 100]"
        :total="pagination.total"
        layout="total, sizes, prev, pager, next, jumper"
        @size-change="handleSizeChange"
        @current-change="handleCurrentChange"
      />
    </el-card>

    <!-- 详情对话框 -->
    <el-dialog
      v-model="detailDialogVisible"
      title="审计日志详情"
      width="80%"
      :close-on-click-modal="false"
    >
      <div v-if="currentDetail" class="detail-content">
        <el-descriptions :column="2" border>
          <el-descriptions-item label="日志ID">{{ currentDetail.id }}</el-descriptions-item>
          <el-descriptions-item label="操作人">
            <el-tag v-if="currentDetail.user_id === 0" type="info" size="small">系统</el-tag>
            <span v-else>{{ currentDetail.username }} (ID: {{ currentDetail.user_id }})</span>
          </el-descriptions-item>
          <el-descriptions-item label="资源类型">
            <el-tag v-if="currentDetail.resource_type === 'rule'" type="primary" size="small">规则</el-tag>
            <el-tag v-else type="warning" size="small">节点</el-tag>
          </el-descriptions-item>
          <el-descriptions-item label="资源名称">{{ currentDetail.resource_name }}</el-descriptions-item>
          <el-descriptions-item label="操作类型">
            <el-tag v-if="currentDetail.action === 'create'" type="success" size="small">创建</el-tag>
            <el-tag v-else-if="currentDetail.action === 'update'" type="primary" size="small">更新</el-tag>
            <el-tag v-else-if="currentDetail.action === 'delete'" type="danger" size="small">删除</el-tag>
            <el-tag v-else-if="currentDetail.action === 'rollback'" type="warning" size="small">回滚</el-tag>
            <el-tag v-else type="info" size="small">{{ currentDetail.action }}</el-tag>
          </el-descriptions-item>
          <el-descriptions-item label="IP地址">{{ currentDetail.ip_address }}</el-descriptions-item>
          <el-descriptions-item label="操作时间" :span="2">
            {{ formatDateTime(currentDetail.created_at) }}
          </el-descriptions-item>
          <el-descriptions-item label="操作描述" :span="2">
            {{ currentDetail.description }}
          </el-descriptions-item>
        </el-descriptions>

        <!-- 规则内容差异（行级彩色对比） -->
        <div v-if="isRuleWithFileDiff" class="change-comparison" style="margin-top: 20px;">
          <h3>规则内容差异</h3>
          <div class="diff-wrapper">
            <div class="diff-side">
              <div class="diff-title diff-title--from">变更前</div>
              <div class="code-lines">
                <div
                  v-for="(line, idx) in diffLeftLines"
                  :key="'dl-' + idx"
                  :class="['code-line', lineClass(line.type)]"
                >
                  <span class="line-no">{{ idx + 1 }}</span>
                  <span class="line-text">{{ line.text }}</span>
                </div>
              </div>
            </div>
            <div class="diff-side">
              <div class="diff-title diff-title--to">变更后</div>
              <div class="code-lines">
                <div
                  v-for="(line, idx) in diffRightLines"
                  :key="'dr-' + idx"
                  :class="['code-line', lineClass(line.type)]"
                >
                  <span class="line-no">{{ idx + 1 }}</span>
                  <span class="line-text">{{ line.text }}</span>
                </div>
              </div>
            </div>
          </div>
        </div>

        <!-- JSON 级别对比（非规则或无文件内容时） -->
        <div v-else-if="currentDetail.action !== 'create'" class="change-comparison" style="margin-top: 20px;">
          <h3>变更前后对比</h3>
          <el-row :gutter="20">
            <el-col :span="12">
              <el-card shadow="hover">
                <template #header>
                  <div style="display: flex; align-items: center; gap: 8px;">
                    <el-icon color="#909399"><Back /></el-icon>
                    <span>变更前</span>
                  </div>
                </template>
                <pre class="json-content">{{ formatJSON(currentDetail.old_value) }}</pre>
              </el-card>
            </el-col>
            <el-col :span="12">
              <el-card shadow="hover">
                <template #header>
                  <div style="display: flex; align-items: center; gap: 8px;">
                    <el-icon color="#67C23A"><Right /></el-icon>
                    <span>变更后</span>
                  </div>
                </template>
                <pre class="json-content">{{ formatJSON(currentDetail.new_value) }}</pre>
              </el-card>
            </el-col>
          </el-row>
        </div>

        <!-- 仅创建操作显示新值 -->
        <div v-else class="change-comparison" style="margin-top: 20px;">
          <h3>创建内容</h3>
          <el-card shadow="hover">
            <pre class="json-content">{{ formatJSON(currentDetail.new_value) }}</pre>
          </el-card>
        </div>
      </div>

      <template #footer>
        <el-button
          v-if="currentDetail && currentDetail.resource_type === 'rule' && currentDetail.action === 'delete'"
          type="warning"
          :loading="restoring"
          :icon="RefreshLeft"
          @click="restoreRule(currentDetail)"
        >
          从此记录恢复
        </el-button>
        <el-button @click="detailDialogVisible = false">关闭</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted, computed } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  Document,
  Search,
  Refresh,
  View,
  Back,
  Right,
  DataAnalysis,
  User,
  Files,
  RefreshLeft,
  Operation
} from '@element-plus/icons-vue'
import axios from 'axios'
import * as Diff from 'diff'

const loading = ref(false)
const auditLogs = ref([])
const stats = ref(null)
const detailDialogVisible = ref(false)
const currentDetail = ref(null)
const restoring = ref(false)

const filterForm = reactive({
  resource_type: '',
  action: '',
  username: '',
  start_date: '',
  end_date: ''
})

const pagination = reactive({
  page: 1,
  page_size: 20,
  total: 0,
  total_page: 0
})

// 规则文件差异检测
const getFileContent = (value) => {
  if (!value) return null
  if (typeof value === 'string') {
    try {
      const obj = JSON.parse(value)
      return obj?.file_content || obj?.content || null
    } catch (e) {
      return null
    }
  }
  if (typeof value === 'object') {
    return value.file_content || value.content || null
  }
  return null
}

const isRuleWithFileDiff = computed(() => {
  const d = currentDetail.value
  if (!d || d.resource_type !== 'rule') return false
  const oldContent = getFileContent(d.old_value)
  const newContent = getFileContent(d.new_value)
  return !!(oldContent || newContent)
})

const buildSideBySide = (oldText, newText) => {
  const parts = Diff.diffLines(oldText || '', newText || '')
  const left = []
  const right = []

  for (let i = 0; i < parts.length; i++) {
    const part = parts[i]
    const lines = part.value.split('\n')
    if (lines.length && lines[lines.length - 1] === '') lines.pop()

    if (!part.added && !part.removed) {
      lines.forEach((l) => {
        left.push({ text: l, type: 'unchanged' })
        right.push({ text: l, type: 'unchanged' })
      })
      continue
    }

    if (part.removed) {
      const next = parts[i + 1]
      if (next && next.added) {
        const oldLines = lines
        const newLines = next.value.split('\n')
        if (newLines.length && newLines[newLines.length - 1] === '') newLines.pop()
        const maxLen = Math.max(oldLines.length, newLines.length)
        for (let k = 0; k < maxLen; k++) {
          left.push({ text: oldLines[k] || '', type: 'modified' })
          right.push({ text: newLines[k] || '', type: 'modified' })
        }
        i++
      } else {
        lines.forEach((l) => {
          left.push({ text: l, type: 'deleted' })
          right.push({ text: '', type: 'empty' })
        })
      }
      continue
    }

    if (part.added) {
      lines.forEach((l) => {
        left.push({ text: '', type: 'empty' })
        right.push({ text: l, type: 'added' })
      })
    }
  }

  return { left, right }
}

const diffLeftLines = computed(() => {
  if (!isRuleWithFileDiff.value) return []
  const d = currentDetail.value
  return buildSideBySide(getFileContent(d?.old_value), getFileContent(d?.new_value)).left
})

const diffRightLines = computed(() => {
  if (!isRuleWithFileDiff.value) return []
  const d = currentDetail.value
  return buildSideBySide(getFileContent(d?.old_value), getFileContent(d?.new_value)).right
})

const lineClass = (type) => {
  switch (type) {
    case 'added':
      return 'line-added'
    case 'deleted':
      return 'line-deleted'
    case 'modified':
      return 'line-modified'
    default:
      return ''
  }
}

// 加载审计日志列表
const loadAuditLogs = async () => {
  loading.value = true
  try {
    const params = {
      page: pagination.page,
      page_size: pagination.page_size,
      ...filterForm
    }

    // 清除空参数
    Object.keys(params).forEach(key => {
      if (params[key] === '' || params[key] === null || params[key] === undefined) {
        delete params[key]
      }
    })

    const response = await axios.get('/api/admin/audit/logs', { params })
    auditLogs.value = response.data.data || []

    if (response.data.pagination) {
      pagination.page = response.data.pagination.page
      pagination.page_size = response.data.pagination.page_size
      pagination.total = response.data.pagination.total
      pagination.total_page = response.data.pagination.total_page
    }
  } catch (error) {
    console.error('加载审计日志失败:', error)
    ElMessage.error(error.response?.data?.error || '加载审计日志失败')
  } finally {
    loading.value = false
  }
}

// 加载统计数据
const loadStats = async () => {
  try {
    const response = await axios.get('/api/admin/audit/stats')
    stats.value = response.data.data
  } catch (error) {
    console.error('加载统计数据失败:', error)
  }
}

// 统计数据展示
const statsDisplay = computed(() => {
  if (!stats.value) return []

  const actionMap = {
    create: { label: '创建操作', icon: 'Plus', color: '#67C23A' },
    update: { label: '更新操作', icon: 'Edit', color: '#409EFF' },
    delete: { label: '删除操作', icon: 'Delete', color: '#F56C6C' },
    rollback: { label: '回滚操作', icon: 'RefreshLeft', color: '#E6A23C' }
  }

  const result = []

  // 操作类型统计
  if (stats.value.action_stats && stats.value.action_stats.length > 0) {
    stats.value.action_stats.forEach(stat => {
      const config = actionMap[stat.Action] || { label: stat.Action, icon: 'Operation', color: '#909399' }
      result.push({
        label: config.label,
        value: stat.Count,
        icon: Operation,
        color: config.color
      })
    })
  }

  // 填充空位（如果少于4个）
  while (result.length < 4) {
    result.push({
      label: '总操作数',
      value: stats.value.action_stats?.reduce((sum, s) => sum + s.Count, 0) || 0,
      icon: DataAnalysis,
      color: '#409EFF'
    })
    break
  }

  return result.slice(0, 4)
})

// 查看详情
const viewDetail = async (row) => {
  try {
    const response = await axios.get(`/api/admin/audit/logs/${row.id}`)
    currentDetail.value = response.data.data
    detailDialogVisible.value = true
  } catch (error) {
    console.error('加载日志详情失败:', error)
    ElMessage.error(error.response?.data?.error || '加载日志详情失败')
  }
}

const restoreRule = async (row) => {
  try {
    await ElMessageBox.confirm(
      `确认从审计记录恢复规则「${row.resource_name || row.resource_id}」吗？`,
      '确认恢复',
      { type: 'warning' }
    )
  } catch {
    return
  }

  if (restoring.value) return
  restoring.value = true
  try {
    await axios.post('/api/admin/audit/rules/restore', { audit_id: row.id })
    ElMessage.success('恢复成功')
    loadAuditLogs()
    if (detailDialogVisible.value) {
      detailDialogVisible.value = false
    }
  } catch (error) {
    console.error('恢复规则失败:', error)
    ElMessage.error(error.response?.data?.error || '恢复失败')
  } finally {
    restoring.value = false
  }
}

// 搜索
const handleSearch = () => {
  pagination.page = 1
  loadAuditLogs()
}

// 重置
const handleReset = () => {
  filterForm.resource_type = ''
  filterForm.action = ''
  filterForm.username = ''
  filterForm.start_date = ''
  filterForm.end_date = ''
  pagination.page = 1
  loadAuditLogs()
}

// 分页
const handleSizeChange = (val) => {
  pagination.page_size = val
  pagination.page = 1
  loadAuditLogs()
}

const handleCurrentChange = (val) => {
  pagination.page = val
  loadAuditLogs()
}

// 格式化日期时间
const formatDateTime = (dateString) => {
  if (!dateString) return '-'
  const date = new Date(dateString)
  return date.toLocaleString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
    hour12: false
  })
}

// 格式化 JSON
const formatJSON = (jsonString) => {
  if (!jsonString) return '无数据'
  try {
    const obj = typeof jsonString === 'string' ? JSON.parse(jsonString) : jsonString
    return JSON.stringify(obj, null, 2)
  } catch (e) {
    return jsonString
  }
}

onMounted(() => {
  loadAuditLogs()
  loadStats()
})
</script>

<style scoped>
.audit-log-container {
  padding: 20px;
}

.card-header {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.card-header h2 {
  margin: 0;
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 20px;
}

.filter-form {
  margin-top: 20px;
}

.stats-cards {
  margin-top: 20px;
}

.stat-card {
  cursor: pointer;
  transition: all 0.3s;
}

.stat-card:hover {
  transform: translateY(-4px);
}

.stat-content {
  display: flex;
  align-items: center;
  gap: 16px;
}

.stat-info {
  flex: 1;
}

.stat-value {
  font-size: 28px;
  font-weight: bold;
  color: #303133;
}

.stat-label {
  font-size: 14px;
  color: #909399;
  margin-top: 4px;
}

.detail-content {
  max-height: 70vh;
  overflow-y: auto;
}

.change-comparison h3 {
  margin: 0 0 16px 0;
  font-size: 16px;
  color: #303133;
}

.json-content {
  max-height: 400px;
  overflow-y: auto;
  background-color: #f5f7fa;
  padding: 12px;
  border-radius: 4px;
  font-size: 13px;
  line-height: 1.6;
  margin: 0;
  white-space: pre-wrap;
  word-wrap: break-word;
}

.diff-wrapper {
  display: flex;
  gap: 16px;
  flex-wrap: wrap;
}

.diff-side {
  flex: 1;
  min-width: 320px;
}

.code-lines {
  border: 1px solid #e8e8e8;
  border-radius: 8px;
  background: #fff;
  max-height: 420px;
  overflow: auto;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.04);
}

.code-line {
  display: flex;
  gap: 12px;
  padding: 4px 12px;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", "Courier New", monospace;
  transition: background 0.2s ease;
}

.code-line:hover {
  background: #fafafa;
}

.line-no {
  width: 40px;
  color: #999;
  text-align: right;
  font-weight: 500;
}

.line-text {
  flex: 1;
  white-space: pre-wrap;
  word-break: break-word;
}

.line-added {
  background: linear-gradient(90deg, rgba(183, 235, 143, 0.15) 0%, transparent 100%);
}

.line-deleted {
  background: linear-gradient(90deg, rgba(255, 189, 189, 0.15) 0%, transparent 100%);
}

.line-modified {
  background: linear-gradient(90deg, rgba(145, 213, 255, 0.15) 0%, transparent 100%);
}

.diff-title {
  padding: 10px 14px;
  border-radius: 8px;
  margin-bottom: 10px;
  font-weight: 600;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.06);
}

.diff-title--from {
  background: linear-gradient(135deg, #fff1f0 0%, #ffccc7 100%);
  color: #cf1322;
}

.diff-title--to {
  background: linear-gradient(135deg, #f6ffed 0%, #d9f7be 100%);
  color: #389e0d;
}

:deep(.el-table) {
  font-size: 14px;
}

:deep(.el-pagination) {
  display: flex;
  justify-content: center;
}
</style>
