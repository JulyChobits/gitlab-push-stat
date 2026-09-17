<template>
  <div>
    <!-- 工具栏（含状态指标） -->
    <div class="toolbar">
      <div class="toolbar-left">
        <span class="toolbar-label">审核日期：</span>
        <el-date-picker v-model="selectedDate" type="date" placeholder="选择日期"
          value-format="YYYY-MM-DD" style="width:160px" @change="handleDateChange" />
        <template v-if="status">
          <el-tag :type="status.enabled ? 'success' : 'info'" size="small" effect="plain" class="stat-chip">
            {{ status.enabled ? '已启用' : '未启用' }}
          </el-tag>
          <span class="stat-chip">记录 <b>{{ status.total }}</b></span>
          <span class="stat-chip" style="color:#67C23A">成功 <b>{{ status.success }}</b></span>
          <span class="stat-chip" style="color:#F56C6C">失败 <b>{{ status.failed }}</b></span>
          <span class="stat-chip" style="color:#E6A23C">待处理 <b>{{ status.pending }}</b></span>
          <span class="stat-chip" style="color:#409EFF">总工时 <b>{{ totalWorkHours }}h</b></span>
        </template>
      </div>
      <div class="toolbar-right">
        <el-button @click="refreshStatus" :loading="statusLoading" size="small">刷新</el-button>
        <el-button type="success" @click="doAIReview" :loading="reviewing" size="small">触发AI审核</el-button>
      </div>
    </div>

    <!-- 人员汇总卡片 -->
    <div style="margin-bottom:16px">
      <h3 style="margin-bottom:12px;color:#303133;font-size:16px">人员汇总</h3>
      <el-row :gutter="16">
        <el-col :span="8" v-for="item in authorSummaries" :key="item.id" style="margin-bottom:16px">
          <el-card shadow="hover" class="author-card" @click="showDetail(item)">
            <template #header>
              <div style="display:flex;align-items:center;justify-content:space-between">
                <span style="font-weight:600;font-size:15px">{{ item.author_name }}</span>
                <el-tag :type="item.status === 'success' ? 'success' : 'danger'" size="small">
                  {{ item.status === 'success' ? '已审核' : '失败' }}
                </el-tag>
              </div>
            </template>
            <div v-if="item.status === 'failed'" style="color:#F56C6C;font-size:13px">
              {{ item.error_message || '审核失败' }}
            </div>
            <div v-else>
              <div class="author-metrics">
                <div class="author-metric">
                  <span class="metric-label">难度</span>
                  <el-tag :type="difficultyType(item.difficulty)" size="small">{{ item.difficulty || '-' }}</el-tag>
                </div>
                <div class="author-metric">
                  <span class="metric-label">工时</span>
                  <span class="metric-val" style="color:#E6A23C">{{ item.work_hours ? item.work_hours.toFixed(1) + 'h' : '-' }}</span>
                </div>
                <div class="author-metric">
                  <span class="metric-label">质量</span>
                  <el-tag :type="qualityType(item.code_quality)" size="small">{{ item.code_quality || '-' }}</el-tag>
                </div>
                <div class="author-metric">
                  <span class="metric-label">提交</span>
                  <span class="metric-val">{{ item.commit_count }}次</span>
                </div>
                <div class="author-metric">
                  <span class="metric-label">代码</span>
                  <span class="metric-val"><span style="color:#67C23A">+{{ item.total_additions }}</span> <span style="color:#F56C6C">-{{ item.total_deletions }}</span></span>
                </div>
                <div class="author-metric">
                  <span class="metric-label">有效</span>
                  <span class="metric-val" :style="{ color: (item.effective_lines || 0) >= 0 ? '#67C23A' : '#F56C6C' }">{{ (item.effective_lines || 0) >= 0 ? '+' : '' }}{{ item.effective_lines || 0 }}</span>
                </div>
              </div>
              <div v-if="item.summary" class="author-summary">{{ item.summary }}</div>
            </div>
          </el-card>
        </el-col>
      </el-row>
      <el-empty v-if="authorSummaries.length === 0 && !listLoading" description="暂无人员汇总数据" />
    </div>

    <!-- 项目审核明细表格 -->
    <div>
      <h3 style="margin-bottom:12px;color:#303133;font-size:16px">项目审核明细</h3>
      <el-table :data="projectReviews" stripe v-loading="listLoading" border size="small">
        <el-table-column prop="author_name" label="提交人" width="100" />
        <el-table-column label="项目" width="160">
          <template #default="{ row }">{{ row.project?.name || '-' }}</template>
        </el-table-column>
        <el-table-column prop="commit_count" label="提交数" width="80" align="center" />
        <el-table-column label="代码变更" width="130" align="center">
          <template #default="{ row }">
            <span style="color:#67C23A">+{{ row.total_additions }}</span>
            <span style="color:#F56C6C;margin-left:4px">-{{ row.total_deletions }}</span>
          </template>
        </el-table-column>
        <el-table-column label="有效行数" width="90" align="center">
          <template #default="{ row }">
            <span :style="{ color: (row.effective_lines || 0) >= 0 ? '#67C23A' : '#F56C6C', fontWeight: 600 }">
              {{ (row.effective_lines || 0) >= 0 ? '+' : '' }}{{ row.effective_lines || 0 }}
            </span>
          </template>
        </el-table-column>
        <el-table-column label="难度" width="90" align="center">
          <template #default="{ row }">
            <el-tag v-if="row.difficulty" :type="difficultyType(row.difficulty)" size="small">{{ row.difficulty }}</el-tag>
            <span v-else>-</span>
          </template>
        </el-table-column>
        <el-table-column label="工时" width="80" align="center">
          <template #default="{ row }">
            <span v-if="row.work_hours" style="color:#E6A23C;font-weight:600">{{ row.work_hours.toFixed(1) }}h</span>
            <span v-else>-</span>
          </template>
        </el-table-column>
        <el-table-column label="质量" width="90" align="center">
          <template #default="{ row }">
            <el-tag v-if="row.code_quality" :type="qualityType(row.code_quality)" size="small">{{ row.code_quality }}</el-tag>
            <span v-else>-</span>
          </template>
        </el-table-column>
        <el-table-column label="状态" width="80" align="center">
          <template #default="{ row }">
            <el-tag :type="row.status === 'success' ? 'success' : 'danger'" size="small">
              {{ row.status === 'success' ? '成功' : '失败' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="summary" label="工作总结" min-width="200" show-overflow-tooltip />
        <el-table-column label="操作" width="80" align="center" fixed="right">
          <template #default="{ row }">
            <el-button type="primary" link size="small" @click="showDetail(row)">详情</el-button>
          </template>
        </el-table-column>
      </el-table>
    </div>

    <!-- 详情抽屉 -->
    <el-drawer v-model="showDrawer" :title="drawerTitle" size="55%" direction="rtl">
      <div v-if="currentItem">
        <div v-if="currentItem.status === 'failed'" style="color:#F56C6C;padding:20px">
          审核失败: {{ currentItem.error_message }}
        </div>
        <div v-else>
          <!-- 指标卡片 -->
          <div class="detail-metrics">
            <div class="detail-metric-item">
              <div class="dm-label">代码难度</div>
              <el-tag :type="difficultyType(currentItem.difficulty)" size="large">{{ currentItem.difficulty || '-' }}</el-tag>
            </div>
            <div class="detail-metric-item">
              <div class="dm-label">预估工时</div>
              <div class="dm-value" style="color:#E6A23C;font-size:24px">{{ currentItem.work_hours ? currentItem.work_hours.toFixed(1) + 'h' : '-' }}</div>
            </div>
            <div class="detail-metric-item">
              <div class="dm-label">代码质量</div>
              <el-tag :type="qualityType(currentItem.code_quality)" size="large">{{ currentItem.code_quality || '-' }}</el-tag>
            </div>
            <div class="detail-metric-item">
              <div class="dm-label">提交次数</div>
              <div class="dm-value">{{ currentItem.commit_count }}次</div>
            </div>
            <div class="detail-metric-item">
              <div class="dm-label">新增行数</div>
              <div class="dm-value" style="color:#67C23A">+{{ currentItem.total_additions }}</div>
            </div>
            <div class="detail-metric-item">
              <div class="dm-label">删除行数</div>
              <div class="dm-value" style="color:#F56C6C">-{{ currentItem.total_deletions }}</div>
            </div>
            <div class="detail-metric-item">
              <div class="dm-label">有效行数</div>
              <div class="dm-value" :style="{ color: (currentItem.effective_lines || 0) >= 0 ? '#67C23A' : '#F56C6C' }">
                {{ (currentItem.effective_lines || 0) >= 0 ? '+' : '' }}{{ currentItem.effective_lines || 0 }}
              </div>
            </div>
          </div>

          <div v-if="currentItem.difficulty_reason" class="detail-section">
            <div class="ds-title">难度评估</div>
            <div class="ds-content">{{ currentItem.difficulty_reason }}</div>
          </div>

          <div v-if="currentItem.work_hours_detail" class="detail-section">
            <div class="ds-title">工时明细</div>
            <div class="ds-content">{{ formatDetail(currentItem.work_hours_detail) }}</div>
          </div>

          <div v-if="currentItem.summary" class="detail-section">
            <div class="ds-title">工作总结</div>
            <div class="ds-content">{{ currentItem.summary }}</div>
          </div>

          <div v-if="currentItem.highlights" class="detail-section">
            <div class="ds-title" style="color:#67C23A">代码亮点</div>
            <div class="ds-content">{{ currentItem.highlights }}</div>
          </div>

          <div v-if="currentItem.suggestions" class="detail-section">
            <div class="ds-title" style="color:#E6A23C">改进建议</div>
            <div class="ds-content">{{ currentItem.suggestions }}</div>
          </div>

          <div v-if="currentItem.quality_details" class="detail-section">
            <div class="ds-title">质量详情</div>
            <div class="ds-content">{{ formatDetail(currentItem.quality_details) }}</div>
          </div>

          <el-divider />
          <div style="color:#909399;font-size:12px">
            <div>审核时间: {{ currentItem.reviewed_at || '-' }}</div>
            <div>审核类型: {{ currentItem.review_type === 'author' ? '人员汇总' : '项目审核' }}</div>
          </div>
        </div>
      </div>
    </el-drawer>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import dayjs from 'dayjs'
import { getAIReviewsDaily, triggerAIReview, getAIReviewStatus } from '../api'

const selectedDate = ref(dayjs().subtract(1, 'day').format('YYYY-MM-DD'))
const status = ref(null)
const statusLoading = ref(false)
const reviewing = ref(false)
const listLoading = ref(false)
const reviews = ref([])
const showDrawer = ref(false)
const currentItem = ref(null)

const drawerTitle = computed(() => {
  if (!currentItem.value) return '审核详情'
  const name = currentItem.value.author_name || ''
  const project = currentItem.value.project?.name || ''
  const type = currentItem.value.review_type === 'author' ? '人员汇总' : project
  return `${name} - ${type} - AI审核详情`
})

// 人员汇总（review_type=author）
const authorSummaries = computed(() => {
  return reviews.value.filter(r => r.review_type === 'author')
})

// 项目审核明细（review_type=project）
const projectReviews = computed(() => {
  return reviews.value.filter(r => r.review_type === 'project')
})

// 总工时（人员汇总的工时总和）
const totalWorkHours = computed(() => {
  return authorSummaries.value
    .reduce((sum, item) => sum + (item.work_hours || 0), 0)
    .toFixed(1)
})

const refreshStatus = async () => {
  if (!selectedDate.value) return
  statusLoading.value = true
  try {
    const { data } = await getAIReviewStatus(selectedDate.value)
    status.value = data
  } catch (e) {
    console.error('获取AI状态失败', e)
  } finally {
    statusLoading.value = false
  }
}

const loadReviews = async () => {
  if (!selectedDate.value) return
  listLoading.value = true
  try {
    const { data } = await getAIReviewsDaily({ date: selectedDate.value })
    reviews.value = data.data || []
  } catch (e) {
    console.error('加载AI结果失败', e)
    reviews.value = []
  } finally {
    listLoading.value = false
  }
}

const doAIReview = async () => {
  if (!selectedDate.value) { ElMessage.warning('请选择审核日期'); return }
  
  try {
    await ElMessageBox.confirm(
      `确认要对 ${selectedDate.value} 的提交数据触发重新AI审核吗？`,
      '确认触发AI审核',
      {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning',
      }
    )
  } catch {
    // 用户点击取消
    return
  }
  
  reviewing.value = true
  try {
    await triggerAIReview(selectedDate.value)
    ElMessage.success('AI审核任务已提交，审核过程可能需要几分钟')
    // 轮询状态
    startPolling()
  } catch (e) {
    ElMessage.error('AI审核触发失败: ' + (e.response?.data?.error || e.message))
  } finally {
    reviewing.value = false
  }
}

let pollingTimer = null
const startPolling = () => {
  stopPolling()
  let count = 0
  pollingTimer = setInterval(async () => {
    count++
    await refreshStatus()
    await loadReviews()
    // 最多轮询60次（5分钟），或全部完成时停止
    if (count > 60 || (status.value && status.value.pending === 0 && status.value.total > 0)) {
      stopPolling()
    }
  }, 5000)
}

const stopPolling = () => {
  if (pollingTimer) {
    clearInterval(pollingTimer)
    pollingTimer = null
  }
}

const handleDateChange = () => {
  stopPolling()
  refreshStatus()
  loadReviews()
}

const showDetail = (item) => {
  currentItem.value = item
  showDrawer.value = true
}

const difficultyType = (d) => {
  const map = { '简单': 'info', '中等': 'warning', '复杂': 'danger', '高复杂': 'danger' }
  return map[d] || 'info'
}

const qualityType = (q) => {
  const map = { '优秀': 'success', '良好': 'primary', '一般': 'warning', '待改进': 'danger' }
  return map[q] || 'info'
}

const formatDetail = (detail) => {
  if (!detail) return '-'
  try {
    const obj = typeof detail === 'string' ? JSON.parse(detail) : detail
    if (typeof obj === 'object') {
      return JSON.stringify(obj, null, 0)
    }
    return String(detail)
  } catch {
    return String(detail)
  }
}

onMounted(() => {
  refreshStatus()
  loadReviews()
})

// 清理轮询
import { onUnmounted } from 'vue'
onUnmounted(() => {
  stopPolling()
})
</script>

<style scoped>
.toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px 20px;
  margin-bottom: 16px;
  background: #fff;
  border-radius: 4px;
  box-shadow: 0 1px 4px rgba(0, 0, 0, 0.08);
}
.toolbar-left {
  display: flex;
  align-items: center;
  gap: 8px;
}
.toolbar-label {
  font-size: 14px;
  color: #606266;
  white-space: nowrap;
}
.toolbar-right {
  display: flex;
  align-items: center;
  gap: 8px;
}

.stat-chip {
  font-size: 13px;
  color: #606266;
  white-space: nowrap;
  padding: 0 6px;
  line-height: 28px;
}
.stat-chip b {
  font-weight: 600;
  margin-left: 2px;
}

.author-card {
  cursor: pointer;
  transition: box-shadow 0.2s;
}
.author-card:hover {
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.12);
}
.author-metrics {
  display: flex;
  gap: 12px;
  flex-wrap: wrap;
  margin-bottom: 10px;
}
.author-metric {
  display: flex;
  flex-direction: column;
  align-items: center;
  min-width: 50px;
}
.author-metric .metric-label {
  font-size: 11px;
  color: #909399;
}
.author-metric .metric-val {
  font-size: 14px;
  font-weight: 600;
}
.author-summary {
  font-size: 12px;
  color: #606266;
  line-height: 1.5;
  overflow: hidden;
  text-overflow: ellipsis;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
}

.detail-metrics {
  display: flex;
  gap: 16px;
  flex-wrap: wrap;
  margin-bottom: 20px;
}
.detail-metric-item {
  text-align: center;
  padding: 10px 16px;
  background: #f5f7fa;
  border-radius: 8px;
  min-width: 90px;
}
.detail-metric-item .dm-label {
  font-size: 12px;
  color: #909399;
  margin-bottom: 4px;
}
.detail-metric-item .dm-value {
  font-size: 18px;
  font-weight: 700;
}

.detail-section {
  margin-top: 16px;
  padding: 12px 14px;
  background: #fafafa;
  border-radius: 6px;
  font-size: 13px;
  line-height: 1.7;
}
.detail-section .ds-title {
  font-weight: 600;
  margin-bottom: 6px;
  color: #303133;
}
.detail-section .ds-content {
  color: #606266;
}
</style>