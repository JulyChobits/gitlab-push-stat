<template>
  <div>
    <!-- 全局工具栏 -->
    <div class="global-toolbar">
      <div class="toolbar-left">
        <span class="toolbar-label">月份：</span>
        <el-date-picker v-model="yearMonth" type="month" placeholder="选择月份" value-format="YYYY-MM"
          size="default" @change="handleFilterChange" style="width:200px" />
      </div>
      <div class="toolbar-right">
        <el-button type="primary" @click="showGenDialog = true">生成报表</el-button>
      </div>
    </div>

    <!-- 表格区域 -->
    <div style="display:flex;flex-direction:column;gap:16px">
      <!-- 人+项目表格 -->
      <el-card shadow="hover">
        <template #header>
          <div class="card-header">
            <span>月报表（人+项目）</span>
          </div>
        </template>
        <el-table :data="reports" stripe v-loading="loading" border size="small" style="width:100%">
          <el-table-column prop="year_month" label="月份" width="90" />
          <el-table-column label="项目" min-width="140"><template #default="{ row }">{{ row.project?.name || '-' }}</template></el-table-column>
          <el-table-column prop="author_name" label="提交人" width="100" />
          <el-table-column prop="commit_count" label="提交" width="65" align="right" />
          <el-table-column prop="additions" width="95" align="right">
            <template #header><span>新增 <el-tooltip content="原始新增代码行数，包含所有文件（含自动生成、配置文件等）" placement="top"><el-icon class="tip-icon"><QuestionFilled /></el-icon></el-tooltip></span></template>
            <template #default="{ row }"><span class="text-green">+{{ row.additions }}</span></template>
          </el-table-column>
          <el-table-column prop="deletions" width="95" align="right">
            <template #header><span>减少 <el-tooltip content="原始删除代码行数，包含所有文件" placement="top"><el-icon class="tip-icon"><QuestionFilled /></el-icon></el-tooltip></span></template>
            <template #default="{ row }"><span class="text-red">-{{ row.deletions }}</span></template>
          </el-table-column>
          <el-table-column prop="net_additions" width="95" align="right">
            <template #header><span>净增 <el-tooltip content="净增行数 = 新增行数 - 删除行数" placement="top"><el-icon class="tip-icon"><QuestionFilled /></el-icon></el-tooltip></span></template>
            <template #default="{ row }"><span :class="row.net_additions >= 0 ? 'text-green' : 'text-red'">{{ row.net_additions >= 0 ? '+' : '' }}{{ row.net_additions }}</span></template>
          </el-table-column>
          <el-table-column prop="effective_lines" width="100" align="right">
            <template #header><span>有效行数 <el-tooltip content="过滤后的有效代码行数，排除了自动生成文件、配置文件、lock文件等噪音代码" placement="top"><el-icon class="tip-icon"><QuestionFilled /></el-icon></el-tooltip></span></template>
            <template #default="{ row }"><span :class="row.effective_lines >= 0 ? 'text-green' : 'text-red'">{{ row.effective_lines >= 0 ? '+' : '' }}{{ row.effective_lines }}</span></template>
          </el-table-column>
        </el-table>
        <div style="margin-top:16px;display:flex;justify-content:flex-end">
          <el-pagination v-model:current-page="page" :page-size="pageSize" :total="total" layout="total, prev, pager, next" @current-change="loadReports" />
        </div>
      </el-card>

      <!-- 人员汇总表格 -->
      <el-card shadow="hover">
        <template #header>
          <div class="card-header">
            <span>人员汇总</span>
          </div>
        </template>
        <el-table :data="authorReports" stripe v-loading="authorLoading" border size="small" style="width:100%">
          <el-table-column prop="year_month" label="月份" width="90" />
          <el-table-column prop="author_name" label="提交人" width="100" />
          <el-table-column prop="commit_count" label="提交" width="65" align="right" />
          <el-table-column prop="additions" min-width="95" align="right">
            <template #header><span>新增 <el-tooltip content="原始新增代码行数，包含所有文件（含自动生成、配置文件等）" placement="top"><el-icon class="tip-icon"><QuestionFilled /></el-icon></el-tooltip></span></template>
            <template #default="{ row }"><span class="text-green">+{{ row.additions }}</span></template>
          </el-table-column>
          <el-table-column prop="deletions" min-width="95" align="right">
            <template #header><span>减少 <el-tooltip content="原始删除代码行数，包含所有文件" placement="top"><el-icon class="tip-icon"><QuestionFilled /></el-icon></el-tooltip></span></template>
            <template #default="{ row }"><span class="text-red">-{{ row.deletions }}</span></template>
          </el-table-column>
          <el-table-column prop="net_additions" min-width="95" align="right">
            <template #header><span>净增 <el-tooltip content="净增行数 = 新增行数 - 删除行数" placement="top"><el-icon class="tip-icon"><QuestionFilled /></el-icon></el-tooltip></span></template>
            <template #default="{ row }"><span :class="row.net_additions >= 0 ? 'text-green' : 'text-red'">{{ row.net_additions >= 0 ? '+' : '' }}{{ row.net_additions }}</span></template>
          </el-table-column>
          <el-table-column prop="effective_lines" min-width="100" align="right">
            <template #header><span>有效行数 <el-tooltip content="过滤后的有效代码行数，排除了自动生成文件、配置文件、lock文件等噪音代码" placement="top"><el-icon class="tip-icon"><QuestionFilled /></el-icon></el-tooltip></span></template>
            <template #default="{ row }"><span :class="row.effective_lines >= 0 ? 'text-green' : 'text-red'">{{ row.effective_lines >= 0 ? '+' : '' }}{{ row.effective_lines }}</span></template>
          </el-table-column>
        </el-table>
        <div style="margin-top:16px;display:flex;justify-content:flex-end">
          <el-pagination v-model:current-page="authorPage" :page-size="authorPageSize" :total="authorTotal" layout="total, prev, pager, next" @current-change="loadAuthorReports" />
        </div>
      </el-card>
    </div>

    <el-dialog v-model="showGenDialog" title="生成月报表" width="400">
      <el-form label-width="80">
        <el-form-item label="月份"><el-date-picker v-model="genMonth" type="month" placeholder="选择月份" value-format="YYYY-MM" style="width:100%" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showGenDialog = false">取消</el-button>
        <el-button type="primary" @click="doGenerate" :loading="generating">生成</el-button>
      </template>
    </el-dialog>
  </div>
</template>
<script setup>
import { ref, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { QuestionFilled } from '@element-plus/icons-vue'
import dayjs from 'dayjs'
import { getMonthlyReports, generateMonthlyReport, getMonthlyAuthorSummary } from '../api'

const reports = ref([])
const loading = ref(false)
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)
const yearMonth = ref(null)
const showGenDialog = ref(false)
const generating = ref(false)
const genMonth = ref(dayjs().subtract(1, 'month').format('YYYY-MM'))

const authorReports = ref([])
const authorLoading = ref(false)
const authorPage = ref(1)
const authorPageSize = ref(20)
const authorTotal = ref(0)

const handleFilterChange = () => {
  page.value = 1
  authorPage.value = 1
  loadReports()
  loadAuthorReports()
}

const loadReports = async () => {
  loading.value = true
  try {
    const params = { page: page.value, page_size: pageSize.value }
    if (yearMonth.value) params.year_month = yearMonth.value
    const { data } = await getMonthlyReports(params)
    reports.value = data.data || []
    total.value = data.total || 0
  } catch (e) {
    ElMessage.error('加载月报表失败')
  } finally {
    loading.value = false
  }
}

const loadAuthorReports = async () => {
  authorLoading.value = true
  try {
    const params = { page: authorPage.value, page_size: authorPageSize.value }
    if (yearMonth.value) params.year_month = yearMonth.value
    const { data } = await getMonthlyAuthorSummary(params)
    authorReports.value = data.data || []
    authorTotal.value = data.total || 0
  } catch (e) {
    ElMessage.error('加载人员汇总失败')
  } finally {
    authorLoading.value = false
  }
}

const doGenerate = async () => {
  if (!genMonth.value) { ElMessage.warning('请选择月份'); return }
  generating.value = true
  try {
    await generateMonthlyReport(genMonth.value)
    ElMessage.success('月报表生成成功')
    showGenDialog.value = false
    loadReports()
    loadAuthorReports()
  } catch (e) {
    ElMessage.error('生成失败')
  } finally {
    generating.value = false
  }
}

onMounted(() => { loadReports(); loadAuthorReports() })
</script>
<style scoped>
.global-toolbar {
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
.card-header { display: flex; align-items: center; justify-content: space-between; flex-wrap: wrap; gap: 8px; }
.text-green { color: #67C23A; font-weight: 600; }
.text-red { color: #F56C6C; font-weight: 600; }
.tip-icon { cursor: help; color: #909399; font-size: 14px; margin-left: 2px; vertical-align: middle; }
.tip-icon:hover { color: #409EFF; }
</style>