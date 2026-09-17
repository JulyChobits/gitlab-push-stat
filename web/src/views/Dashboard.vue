<template>
  <div class="dashboard">
    <el-row :gutter="16" class="stat-row">
      <el-col :span="4">
        <el-card shadow="hover" class="stat-card">
          <div class="stat-item">
            <div class="stat-icon" style="background:#409EFF"><el-icon :size="24"><EditPen /></el-icon></div>
            <div class="stat-info">
              <div class="stat-label">今日提交</div>
              <div class="stat-value">{{ summary.today_commits || 0 }}</div>
            </div>
          </div>
        </el-card>
      </el-col>
      <el-col :span="4">
        <el-card shadow="hover" class="stat-card">
          <div class="stat-item">
            <div class="stat-icon" style="background:#909399"><el-icon :size="24"><Clock /></el-icon></div>
            <div class="stat-info">
              <div class="stat-label">昨日提交</div>
              <div class="stat-value">{{ summary.yesterday_commits || 0 }}</div>
            </div>
          </div>
        </el-card>
      </el-col>
      <el-col :span="4">
        <el-card shadow="hover" class="stat-card">
          <div class="stat-item">
            <div class="stat-icon" style="background:#67C23A"><el-icon :size="24"><User /></el-icon></div>
            <div class="stat-info">
              <div class="stat-label">活跃开发者(7日)</div>
              <div class="stat-value">{{ summary.active_developers || 0 }}</div>
            </div>
          </div>
        </el-card>
      </el-col>
      <el-col :span="4">
        <el-card shadow="hover" class="stat-card">
          <div class="stat-item">
            <div class="stat-icon" style="background:#E6A23C"><el-icon :size="24"><TrendCharts /></el-icon></div>
            <div class="stat-info">
              <div class="stat-label">本月净增行数</div>
              <div class="stat-value">{{ formatNumber(summary.month_net_lines || 0) }}</div>
            </div>
          </div>
        </el-card>
      </el-col>
      <el-col :span="4">
        <el-card shadow="hover" class="stat-card">
          <div class="stat-item">
            <div class="stat-icon" style="background:#F56C6C"><el-icon :size="24"><Folder /></el-icon></div>
            <div class="stat-info">
              <div class="stat-label">监控项目数</div>
              <div class="stat-value">{{ summary.project_count || 0 }}</div>
            </div>
          </div>
        </el-card>
      </el-col>
      <el-col :span="4">
        <el-card shadow="hover" class="stat-card">
          <div class="stat-item">
            <div class="stat-icon" style="background:#9B59B6"><el-icon :size="24"><DataLine /></el-icon></div>
            <div class="stat-info">
              <div class="stat-label">本月提交数</div>
              <div class="stat-value">{{ summary.month_commits || 0 }}</div>
            </div>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <el-row :gutter="16" style="margin-top:16px">
      <el-col :span="16">
        <el-card shadow="hover">
          <template #header>
            <div class="card-header">
              <span>提交趋势（按人员）</span>
              <el-radio-group v-model="trendDays" size="small" @change="loadAuthorTrends">
                <el-radio-button :label="7">7天</el-radio-button>
                <el-radio-button :label="30">30天</el-radio-button>
                <el-radio-button :label="90">90天</el-radio-button>
              </el-radio-group>
            </div>
          </template>
          <v-chart :option="commitTrendOption" style="height:360px" autoresize />
        </el-card>
      </el-col>
      <el-col :span="8">
        <el-card shadow="hover">
          <template #header>项目提交分布（本月）</template>
          <v-chart :option="pieOption" style="height:360px" autoresize />
        </el-card>
      </el-col>
    </el-row>

    <el-row :gutter="16" style="margin-top:16px">
      <el-col :span="16">
        <el-card shadow="hover">
          <template #header>代码有效行数趋势（按人员）</template>
          <v-chart :option="netLinesTrendOption" style="height:360px" autoresize />
        </el-card>
      </el-col>
      <el-col :span="8">
        <el-card shadow="hover">
          <template #header>项目有效代码分布（本月）</template>
          <v-chart :option="codePieOption" style="height:360px" autoresize />
        </el-card>
      </el-col>
    </el-row>

    <el-row :gutter="16" style="margin-top:16px">
      <el-col :span="16">
        <el-card shadow="hover">
          <template #header>
            <div class="card-header">
              <span>新增/删除行数对比趋势</span>
              <el-radio-group v-model="adTrendDays" size="small" @change="loadAdTrends">
                <el-radio-button :label="7">7天</el-radio-button>
                <el-radio-button :label="30">30天</el-radio-button>
                <el-radio-button :label="90">90天</el-radio-button>
              </el-radio-group>
            </div>
          </template>
          <v-chart :option="adTrendOption" style="height:360px" autoresize />
        </el-card>
      </el-col>
      <el-col :span="8">
        <el-card shadow="hover">
          <template #header>项目活跃度排名（近30天）</template>
          <el-table :data="projectActivity" stripe size="small" style="height:360px;overflow:auto">
            <el-table-column type="index" label="#" width="40" />
            <el-table-column prop="project_name" label="项目名称" show-overflow-tooltip />
            <el-table-column prop="commit_count" label="提交数" width="70" align="right" />
            <el-table-column prop="developer_count" label="参与人数" width="70" align="right" />
          </el-table>
        </el-card>
      </el-col>
    </el-row>

    <el-row :gutter="16" style="margin-top:16px">
      <el-col :span="12">
        <el-card shadow="hover">
          <template #header>提交次数 TOP10（近30天）</template>
          <el-table :data="topContributors" stripe size="small">
            <el-table-column type="index" label="#" width="50" />
            <el-table-column prop="author_name" label="提交人" />
            <el-table-column prop="commit_count" label="提交次数" width="100" align="right" />
            <el-table-column prop="net_additions" label="净增行数" width="110" align="right">
              <template #default="{ row }">
                <span :class="row.net_additions >= 0 ? 'text-green' : 'text-red'">{{ row.net_additions >= 0 ? '+' : '' }}{{ row.net_additions }}</span>
              </template>
            </el-table-column>
          </el-table>
        </el-card>
      </el-col>
      <el-col :span="12">
        <el-card shadow="hover">
          <template #header>代码贡献 TOP10（近30天）</template>
          <el-table :data="topByLines" stripe size="small">
            <el-table-column type="index" label="#" width="50" />
            <el-table-column prop="author_name" label="提交人" />
            <el-table-column prop="additions" label="新增行数" width="100" align="right" />
            <el-table-column prop="deletions" label="减少行数" width="100" align="right" />
          </el-table>
        </el-card>
      </el-col>
    </el-row>

    <el-row :gutter="16" style="margin-top:16px">
      <el-col :span="12">
        <el-card shadow="hover">
          <template #header>连续提交天数排名</template>
          <el-table :data="consecutiveDays" stripe size="small">
            <el-table-column type="index" label="#" width="50" />
            <el-table-column prop="author_name" label="提交人" />
            <el-table-column prop="consecutive_days" label="连续天数" width="90" align="right">
              <template #default="{ row }">
                <span style="color:#E6A23C;font-weight:600">{{ row.consecutive_days }}天</span>
              </template>
            </el-table-column>
            <el-table-column prop="last_commit_date" label="最后提交日期" width="130" />
          </el-table>
        </el-card>
      </el-col>
      <el-col :span="12">
        <el-card shadow="hover">
          <template #header>人员最近提交顺序</template>
          <el-table :data="recentCommits" stripe size="small">
            <el-table-column type="index" label="#" width="50" />
            <el-table-column prop="author_name" label="提交人" width="120" />
            <el-table-column prop="last_commit_at" label="最近提交时间" width="180">
              <template #default="{ row }">{{ formatTime(row.last_commit_at) }}</template>
            </el-table-column>
            <el-table-column prop="last_project_name" label="最近提交项目" width="200" show-overflow-tooltip />
            <el-table-column prop="last_commit_msg" label="最近提交信息" show-overflow-tooltip />
          </el-table>
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>

<script setup>
import { ref, onMounted, computed } from 'vue'
import { use } from 'echarts/core'
import { CanvasRenderer } from 'echarts/renderers'
import { LineChart, PieChart, BarChart } from 'echarts/charts'
import { GridComponent, TooltipComponent, LegendComponent, TitleComponent } from 'echarts/components'
import VChart from 'vue-echarts'
import { getSummary, getTopContributors, getProjectDistribution, getRecentCommits, getTrendsByAuthor, getTrends, getProjectActivity, getConsecutiveDays } from '../api'

use([CanvasRenderer, LineChart, PieChart, BarChart, GridComponent, TooltipComponent, LegendComponent, TitleComponent])

const summary = ref({})
const authorTrendData = ref({ dates: [], authors: {} })
const trendData = ref([])
const topContributors = ref([])
const projectDist = ref([])
const recentCommits = ref([])
const projectActivity = ref([])
const consecutiveDays = ref([])
const trendDays = ref(30)
const adTrendDays = ref(30)

const topByLines = computed(() => [...topContributors.value].sort((a, b) => b.additions - a.additions).slice(0, 10))
const formatNumber = (n) => { if (Math.abs(n)>=10000) return (n/10000).toFixed(1)+'w'; if (Math.abs(n)>=1000) return (n/1000).toFixed(1)+'k'; return n }
const formatTime = (t) => { if (!t) return '-'; return t.replace('T', ' ').slice(0, 19) }

const chartColors = ['#409EFF','#67C23A','#E6A23C','#F56C6C','#9B59B6','#1ABC9C','#E74C3C','#3498DB','#00C1DE','#909399','#F39C12','#8E44AD','#2ECC71','#D35400','#2980B9']

// 提交趋势（按人员折线图）
const commitTrendOption = computed(() => {
  const dates = authorTrendData.value.dates || []
  const authors = authorTrendData.value.authors || {}
  const authorNames = Object.keys(authors)

  const series = authorNames.map((name, i) => ({
    name,
    type: 'line',
    data: authors[name].commits || [],
    smooth: true,
    symbol: 'circle',
    symbolSize: 4,
    itemStyle: { color: chartColors[i % chartColors.length] },
  }))

  return {
    tooltip: { trigger: 'axis', appendToBody: true },
    legend: { data: authorNames, bottom: 0, type: 'scroll' },
    grid: { left: 50, right: 20, top: 20, bottom: 50 },
    xAxis: { type: 'category', data: dates.map(d => d.slice(5)), axisLabel: { rotate: 45 } },
    yAxis: { type: 'value', name: '提交次数', minInterval: 1 },
    series,
  }
})

// 代码有效行数趋势（按人员折线图）
const netLinesTrendOption = computed(() => {
  const dates = authorTrendData.value.dates || []
  const authors = authorTrendData.value.authors || {}
  const authorNames = Object.keys(authors)

  const series = authorNames.map((name, i) => ({
    name,
    type: 'line',
    data: authors[name].effective_lines || [],
    smooth: true,
    symbol: 'circle',
    symbolSize: 4,
    itemStyle: { color: chartColors[i % chartColors.length] },
  }))

  return {
    tooltip: {
      trigger: 'axis',
      appendToBody: true,
      formatter: (params) => {
        let html = params[0].axisValue
        params.forEach(p => {
          const val = p.value
          const color = val >= 0 ? '#67C23A' : '#F56C6C'
          html += `<br/>${p.marker}${p.seriesName}: <span style="color:${color};font-weight:bold">${val >= 0 ? '+' : ''}${val}</span>`
        })
        return html
      }
    },
    legend: { data: authorNames, bottom: 0, type: 'scroll' },
    grid: { left: 60, right: 20, top: 20, bottom: 50 },
    xAxis: { type: 'category', data: dates.map(d => d.slice(5)), axisLabel: { rotate: 45 } },
    yAxis: { type: 'value', name: '有效行数' },
    series,
  }
})

// 新增/删除行数对比趋势（堆叠柱状图）
const adTrendOption = computed(() => {
  const data = trendData.value || []
  const dates = data.map(d => (d.report_date || '').slice(5))
  const additions = data.map(d => d.additions || 0)
  const deletions = data.map(d => d.deletions || 0)

  return {
    tooltip: {
      trigger: 'axis',
      appendToBody: true,
      formatter: (params) => {
        let html = params[0].axisValue
        params.forEach(p => {
          html += `<br/>${p.marker}${p.seriesName}: <span style="font-weight:bold">${p.value}</span>行`
        })
        return html
      }
    },
    legend: { data: ['新增行数', '删除行数'], bottom: 0 },
    grid: { left: 60, right: 20, top: 20, bottom: 50 },
    xAxis: { type: 'category', data: dates, axisLabel: { rotate: 45 } },
    yAxis: { type: 'value', name: '行数' },
    series: [
      { name: '新增行数', type: 'bar', stack: 'code', data: additions, itemStyle: { color: '#67C23A' } },
      { name: '删除行数', type: 'bar', stack: 'code', data: deletions, itemStyle: { color: '#F56C6C' } },
    ],
  }
})

const pieOption = computed(() => ({
  tooltip: { trigger: 'item', formatter: '{b}: {c}次 ({d}%)', appendToBody: true },
  series: [{ type: 'pie', radius: ['40%','70%'], data: projectDist.value.map(p => ({ name: p.project_name || '未知', value: p.commit_count })), label: { formatter: '{b}\n{d}%' } }],
}))

const codeColors = ['#409EFF','#67C23A','#E6A23C','#F56C6C','#909399','#00C1DE','#9B59B6','#1ABC9C','#E74C3C','#3498DB']
const codePieOption = computed(() => ({
  tooltip: { trigger: 'item', formatter: (p) => `${p.name}: ${p.data.rawValue >= 0 ? '+' : ''}${p.data.rawValue}行 (${p.percent}%)`, appendToBody: true },
  color: codeColors,
  series: [{
    type: 'pie', radius: ['40%','70%'],
    data: projectDist.value
      .filter(p => p.effective_lines !== 0)
      .map((p, i) => ({
        name: p.project_name || '未知',
        value: Math.abs(p.effective_lines),
        rawValue: p.effective_lines,
        itemStyle: { color: codeColors[i % codeColors.length] }
      })),
    label: { formatter: '{b}\n{d}%' }
  }],
}))

const loadSummary = async () => { try { const { data } = await getSummary(); summary.value = data.data || {} } catch(e) { console.error(e) } }
const loadAuthorTrends = async () => { try { const { data } = await getTrendsByAuthor(trendDays.value*24); authorTrendData.value = data.data || { dates: [], authors: {} } } catch(e) { console.error(e) } }
const loadAdTrends = async () => { try { const { data } = await getTrends(adTrendDays.value*24); trendData.value = data.data || [] } catch(e) { console.error(e) } }
const loadTopContributors = async () => { try { const { data } = await getTopContributors(30*24); topContributors.value = data.data || [] } catch(e) { console.error(e) } }
const loadProjectDist = async () => { try { const { data } = await getProjectDistribution(); projectDist.value = data.data || [] } catch(e) { console.error(e) } }
const loadRecentCommits = async () => { try { const { data } = await getRecentCommits(); recentCommits.value = data.data || [] } catch(e) { console.error(e) } }
const loadProjectActivity = async () => { try { const { data } = await getProjectActivity(30*24); projectActivity.value = data.data || [] } catch(e) { console.error(e) } }
const loadConsecutiveDays = async () => { try { const { data } = await getConsecutiveDays(); consecutiveDays.value = data.data || [] } catch(e) { console.error(e) } }

onMounted(() => {
  loadSummary()
  loadAuthorTrends()
  loadAdTrends()
  loadTopContributors()
  loadProjectDist()
  loadRecentCommits()
  loadProjectActivity()
  loadConsecutiveDays()
})
</script>

<style scoped>
.stat-card { cursor: default; }
.stat-item { display: flex; align-items: center; gap: 12px; }
.stat-icon { width: 48px; height: 48px; border-radius: 10px; display: flex; align-items: center; justify-content: center; color: #fff; flex-shrink: 0; }
.stat-label { font-size: 12px; color: #909399; margin-bottom: 2px; }
.stat-value { font-size: 24px; font-weight: 700; color: #303133; }
.card-header { display: flex; align-items: center; justify-content: space-between; }
.text-green { color: #67C23A; font-weight: 600; }
.text-red { color: #F56C6C; font-weight: 600; }
</style>