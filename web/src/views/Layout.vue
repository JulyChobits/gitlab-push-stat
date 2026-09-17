<template>
  <el-container class="layout-container">
    <el-aside width="220px" class="aside">
      <div class="logo">
        <el-icon size="28" color="#409EFF"><DataBoard /></el-icon>
        <span class="logo-text">GitLab Stat</span>
      </div>
      <el-menu
        :default-active="currentRoute"
        router
        background-color="#1d1e1f"
        text-color="#bfcbd9"
        active-text-color="#409EFF"
      >
        <el-menu-item index="/">
          <el-icon><DataBoard /></el-icon>
          <span>看板</span>
        </el-menu-item>
        <el-menu-item index="/projects">
          <el-icon><Folder /></el-icon>
          <span>项目管理</span>
        </el-menu-item>
        <el-sub-menu index="reports">
          <template #title>
            <el-icon><Calendar /></el-icon>
            <span>报表中心</span>
          </template>
          <el-menu-item index="/reports/daily">日报表</el-menu-item>
          <el-menu-item index="/reports/weekly">周报表</el-menu-item>
          <el-menu-item index="/reports/monthly">月报表</el-menu-item>
          <el-menu-item index="/reports/annual">年度热力图</el-menu-item>
          <el-menu-item index="/reports/ai-review">AI审核</el-menu-item>
        </el-sub-menu>
        <el-menu-item index="/filters">
          <el-icon><Filter /></el-icon>
          <span>过滤规则</span>
        </el-menu-item>
        <el-menu-item index="/settings">
          <el-icon><Setting /></el-icon>
          <span>设置</span>
        </el-menu-item>
      </el-menu>
    </el-aside>
    <el-container>
      <el-header class="header">
        <span class="page-title">{{ pageTitle }}</span>
        <div class="header-actions">
          <el-button type="primary" :icon="Refresh" :loading="syncing" @click="handleSync">
            同步数据
          </el-button>
        </div>
      </el-header>
      <el-main class="main">
        <router-view />
      </el-main>
    </el-container>
  </el-container>
</template>

<script setup>
import { computed, ref } from 'vue'
import { useRoute } from 'vue-router'
import { Refresh } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import { triggerSync } from '../api'

const route = useRoute()
const syncing = ref(false)

const currentRoute = computed(() => route.path)

const pageTitle = computed(() => {
  const titles = {
    '/': '看板',
    '/projects': '项目管理',
    '/reports/daily': '日报表',
    '/reports/weekly': '周报表',
    '/reports/monthly': '月报表',
    '/reports/annual': '年度热力图',
    '/reports/ai-review': 'AI审核',
    '/filters': '过滤规则',
    '/settings': '设置',
  }
  return titles[route.path] || 'GitLab Push Stat'
})

const handleSync = async () => {
  syncing.value = true
  try {
    await triggerSync()
    ElMessage.success('同步任务已触发，请稍后刷新查看')
  } catch (e) {
    ElMessage.error('触发同步失败: ' + (e.response?.data?.error || e.message))
  } finally {
    syncing.value = false
  }
}
</script>

<style>
* {
  margin: 0;
  padding: 0;
  box-sizing: border-box;
}
html, body, #app {
  height: 100%;
  font-family: 'Helvetica Neue', Helvetica, 'PingFang SC', sans-serif;
}
.layout-container {
  height: 100vh;
}
.aside {
  background-color: #1d1e1f;
  overflow-y: auto;
}
.logo {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 20px 0;
  border-bottom: 1px solid #333;
}
.logo-text {
  color: #fff;
  font-size: 18px;
  font-weight: 600;
}
.header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  background: #fff;
  border-bottom: 1px solid #ebeef5;
  box-shadow: 0 1px 4px rgba(0, 0, 0, 0.08);
}
.page-title {
  font-size: 20px;
  font-weight: 600;
  color: #303133;
}
.main {
  background: #f0f2f5;
  padding: 20px;
}
.el-menu {
  border-right: none !important;
}
</style>