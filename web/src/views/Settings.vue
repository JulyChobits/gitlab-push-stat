<template>
  <div>
    <el-card shadow="hover">
      <template #header><span>系统设置</span></template>
      <el-descriptions :column="1" border>
        <el-descriptions-item label="服务器端口">{{ config.server?.port || '-' }}</el-descriptions-item>
        <el-descriptions-item label="运行模式">{{ config.server?.mode || '-' }}</el-descriptions-item>
        <el-descriptions-item label="GitLab 地址">{{ config.gitlab?.url || '-' }}</el-descriptions-item>
        <el-descriptions-item label="数据库路径">{{ config.database?.path || '-' }}</el-descriptions-item>
        <el-descriptions-item label="数据同步">{{ config.schedule?.sync || '-' }}</el-descriptions-item>
        <el-descriptions-item label="日报表生成">{{ config.schedule?.daily || '-' }}</el-descriptions-item>
        <el-descriptions-item label="周报表生成">{{ config.schedule?.weekly || '-' }}</el-descriptions-item>
        <el-descriptions-item label="月报表生成">{{ config.schedule?.monthly || '-' }}</el-descriptions-item>
      </el-descriptions>
      <el-alert type="info" :closable="false" style="margin-top:16px">
        <template #title>提示</template>
        系统设置请通过修改 <code>config/config.yaml</code> 配置文件完成，修改后重启服务生效。
      </el-alert>
    </el-card>

    <el-card shadow="hover" style="margin-top:16px">
      <template #header><span>数据初始化</span></template>
      <el-alert type="warning" :closable="false" style="margin-bottom:16px">
        <template #title>注意</template>
        全量同步将从 GitLab 拉取所有启用项目的全部历史提交数据。此操作耗时较长（取决于提交数量），仅建议在系统首次上线时使用。<strong>不会触发 AI 分析</strong>，仅拉取数据入库。已存在的提交记录会自动跳过，重复执行安全。
      </el-alert>
      <el-button type="danger" :loading="syncingFull" @click="handleFullHistorySync">
        全量同步历史数据
      </el-button>
      <el-button type="warning" :loading="clearingReports" style="margin-left:12px" @click="handleClearReports">
        清除所有报表数据
      </el-button>
    </el-card>

    <el-card shadow="hover" style="margin-top:16px">
      <template #header><span>同步日志</span></template>
      <el-table :data="syncLogs" stripe v-loading="loadingLogs" size="small" max-height="400">
        <el-table-column prop="project_id" label="项目ID" width="80" />
        <el-table-column prop="status" label="状态" width="90" align="center">
          <template #default="{ row }">
            <el-tag :type="row.status === 'success' ? 'success' : row.status === 'failed' ? 'danger' : 'warning'" size="small">
              {{ row.status }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="commit_count" label="提交数" width="80" align="right" />
        <el-table-column label="开始时间" width="170">
          <template #default="{ row }">{{ formatTime(row.start_time) }}</template>
        </el-table-column>
        <el-table-column label="结束时间" width="170">
          <template #default="{ row }">{{ formatTime(row.end_time) }}</template>
        </el-table-column>
        <el-table-column prop="error_msg" label="错误信息" show-overflow-tooltip />
      </el-table>
    </el-card>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import dayjs from 'dayjs'
import { ElMessageBox, ElMessage } from 'element-plus'
import { getSyncLogs, getSystemConfig, triggerFullHistorySync, clearAllReports } from '../api'

const config = ref({})
const syncLogs = ref([])
const loadingLogs = ref(false)
const syncingFull = ref(false)
const clearingReports = ref(false)

const formatTime = (t) => t ? dayjs(t).format('YYYY-MM-DD HH:mm:ss') : '-'

const loadConfig = async () => {
  try {
    const { data } = await getSystemConfig()
    config.value = data.data || {}
  } catch (e) { console.error(e) }
}

const loadSyncLogs = async () => {
  loadingLogs.value = true
  try {
    const { data } = await getSyncLogs()
    syncLogs.value = data.data || []
  } catch (e) { console.error(e) }
  finally { loadingLogs.value = false }
}

const handleFullHistorySync = async () => {
  try {
    await ElMessageBox.confirm(
      '此操作将同步所有启用项目的全部历史提交数据，耗时较长（可能需要数分钟到数小时），且不会触发 AI 分析。确定继续吗？',
      '全量同步历史数据',
      {
        confirmButtonText: '确定执行',
        cancelButtonText: '取消',
        type: 'warning',
      }
    )
    syncingFull.value = true
    await triggerFullHistorySync()
    ElMessage.success('全量同步任务已触发，请稍后刷新同步日志查看进度')
  } catch (e) {
    if (e !== 'cancel') {
      ElMessage.error('触发全量同步失败: ' + (e.message || e))
    }
  } finally {
    syncingFull.value = false
  }
}

const handleClearReports = async () => {
  try {
    await ElMessageBox.confirm(
      '此操作将删除所有日报表、周报表和月报表数据，清除后看板统计将无数据。确定继续吗？',
      '清除所有报表数据',
      {
        confirmButtonText: '确定清除',
        cancelButtonText: '取消',
        type: 'warning',
      }
    )
    clearingReports.value = true
    const { data } = await clearAllReports()
    const deleted = data.deleted || {}
    ElMessage.success(`已清除：日报 ${deleted.daily_reports || 0} 条，周报 ${deleted.weekly_reports || 0} 条，月报 ${deleted.monthly_reports || 0} 条`)
  } catch (e) {
    if (e !== 'cancel') {
      ElMessage.error('清除报表失败: ' + (e.response?.data?.error || e.message))
    }
  } finally {
    clearingReports.value = false
  }
}

onMounted(() => {
  loadConfig()
  loadSyncLogs()
})
</script>
