<template>
  <div>
    <!-- 全局工具栏 -->
    <div class="global-toolbar">
      <div class="toolbar-left">
        <span class="toolbar-label">年份：</span>
        <el-date-picker v-model="year" type="year" placeholder="选择年份" value-format="YYYY"
          size="default" @change="loadData" style="width:140px" />
        <!-- 颜色图例 -->
        <div class="legend">
          <span class="legend-label">提交量：</span>
          <span class="legend-item"><span class="legend-cell" style="background:#ebedf0"></span>0(工作日)</span>
          <span class="legend-item"><span class="legend-cell" style="background:#d9dce0"></span>0(周末)</span>
          <span class="legend-item"><span class="legend-cell" style="background:#9be9a8"></span>1~2</span>
          <span class="legend-item"><span class="legend-cell" style="background:#40c463"></span>3~5</span>
          <span class="legend-item"><span class="legend-cell" style="background:#30a14e"></span>6~10</span>
          <span class="legend-item"><span class="legend-cell" style="background:#216e39"></span>11+</span>
        </div>
      </div>
    </div>

    <!-- 热力图区域 -->
    <el-card shadow="hover" v-loading="loading">
      <div v-if="authors.length === 0 && !loading" class="empty-tip">
        暂无数据，请先生成日报表
      </div>
      <div v-for="author in authors" :key="author.author_name" class="author-row">
        <div class="author-name" :title="author.author_name">{{ author.author_name }}</div>
        <div class="calendar-wrap">
          <div class="month-labels">
            <span v-for="(m, idx) in monthLabels" :key="idx" class="month-label" :style="{ left: m.left + 'px' }">{{ m.label }}</span>
          </div>
          <div class="calendar-grid">
            <div v-for="(week, wi) in yearWeeks" :key="wi" class="week-col">
              <div v-for="di in 7" :key="di" class="day-cell"
                :style="{ background: getCellColor(author, week[di - 1], di) }"
                @mouseenter="showTooltip($event, author, week[di - 1])"
                @mouseleave="hideTooltip"
              ></div>
            </div>
          </div>
        </div>
      </div>
    </el-card>

    <!-- Tooltip -->
    <div v-if="tooltip.visible" class="heatmap-tooltip" :style="{ left: tooltip.x + 'px', top: tooltip.y + 'px' }">
      <div><strong>{{ tooltip.date }}</strong></div>
      <div>{{ tooltip.author }}: {{ tooltip.commits }} 次提交</div>
      <div v-if="tooltip.commits > 0">
        <span class="text-green">+{{ tooltip.additions }}</span> /
        <span class="text-red">-{{ tooltip.deletions }}</span>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import dayjs from 'dayjs'
import isoWeek from 'dayjs/plugin/isoWeek'
import { getAnnualHeatmap } from '../api'

dayjs.extend(isoWeek)

const year = ref(dayjs().format('YYYY'))
const loading = ref(false)
const authors = ref([])

const tooltip = ref({ visible: false, x: 0, y: 0, date: '', author: '', commits: 0, additions: 0, deletions: 0 })

// 计算该年的所有周（每周包含7天的日期字符串）
const yearWeeks = computed(() => {
  const y = parseInt(year.value)
  const start = dayjs(`${y}-01-01`)
  const end = dayjs(`${y}-12-31`)

  // 找到第一周的周一
  let current = start.startOf('isoWeek') // 周一

  const weeks = []
  while (current.isBefore(end) || current.format('YYYY-MM-DD') === end.format('YYYY-MM-DD')) {
    const week = []
    for (let i = 0; i < 7; i++) {
      const d = current.add(i, 'day')
      // 只包含本年度的日期
      if (d.year() === y) {
        week.push(d.format('YYYY-MM-DD'))
      } else {
        week.push(null)
      }
    }
    weeks.push(week)
    current = current.add(7, 'day')
  }
  return weeks
})

// 月份标签位置
const monthLabels = computed(() => {
  const y = parseInt(year.value)
  const labels = []
  const startOfYear = dayjs(`${y}-01-01`)
  const firstMonday = startOfYear.startOf('isoWeek')

  for (let m = 0; m < 12; m++) {
    const firstOfMonth = dayjs(`${y}-${String(m + 1).padStart(2, '0')}-01`)
    const diffDays = firstOfMonth.diff(firstMonday, 'day')
    const weekIndex = Math.floor(diffDays / 7)
    labels.push({
      label: `${m + 1}月`,
      left: weekIndex * 14
    })
  }
  return labels
})

const getCellColor = (author, dateStr, dayIndex) => {
  if (!dateStr) return 'transparent'
  const isWeekend = dayIndex === 6 || dayIndex === 7
  const day = author.days[dateStr]
  const c = day ? day.commit_count : 0
  if (c === 0) return isWeekend ? '#d9dce0' : '#ebedf0'
  if (c <= 2) return '#9be9a8'
  if (c <= 5) return '#40c463'
  if (c <= 10) return '#30a14e'
  return '#216e39'
}

const showTooltip = (e, author, dateStr) => {
  if (!dateStr) return
  const day = author.days[dateStr]
  tooltip.value = {
    visible: true,
    x: e.clientX + 12,
    y: e.clientY + 12,
    date: dateStr,
    author: author.author_name,
    commits: day ? day.commit_count : 0,
    additions: day ? day.additions : 0,
    deletions: day ? day.deletions : 0,
  }
}

const hideTooltip = () => {
  tooltip.value.visible = false
}

const loadData = async () => {
  loading.value = true
  try {
    const { data } = await getAnnualHeatmap(year.value)
    authors.value = data.authors || []
  } catch (e) {
    ElMessage.error('加载年度热力图失败')
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  loadData()
})
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
  gap: 16px;
  flex-wrap: wrap;
}
.toolbar-label {
  font-size: 14px;
  color: #606266;
  white-space: nowrap;
}
.legend {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 12px;
  color: #606266;
}
.legend-label {
  margin-right: 4px;
}
.legend-item {
  display: flex;
  align-items: center;
  gap: 3px;
}
.legend-cell {
  display: inline-block;
  width: 12px;
  height: 12px;
  border-radius: 2px;
}

.empty-tip {
  text-align: center;
  padding: 60px 0;
  color: #909399;
  font-size: 14px;
}

.author-row {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  padding: 8px 0;
  border-bottom: 1px solid #f0f0f0;
}
.author-row:last-child {
  border-bottom: none;
}
.author-name {
  width: 80px;
  min-width: 80px;
  font-size: 13px;
  font-weight: 500;
  color: #303133;
  text-align: right;
  padding-top: 18px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.calendar-wrap {
  flex: 1;
  min-width: 0;
  overflow-x: auto;
}
.month-labels {
  position: relative;
  height: 18px;
  margin-bottom: 4px;
}
.month-label {
  position: absolute;
  font-size: 11px;
  color: #909399;
}
.calendar-grid {
  display: flex;
  gap: 2px;
}
.week-col {
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.day-cell {
  width: 12px;
  height: 12px;
  border-radius: 2px;
  cursor: pointer;
  transition: opacity 0.15s;
}
.day-cell:hover {
  opacity: 0.7;
  outline: 1px solid #333;
}

.heatmap-tooltip {
  position: fixed;
  background: #333;
  color: #fff;
  padding: 8px 12px;
  border-radius: 4px;
  font-size: 12px;
  line-height: 1.6;
  z-index: 9999;
  pointer-events: none;
  white-space: nowrap;
  box-shadow: 0 2px 8px rgba(0,0,0,0.3);
}
.text-green { color: #67C23A; }
.text-red { color: #F56C6C; }
</style>