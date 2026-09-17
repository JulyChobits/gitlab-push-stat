import { createRouter, createWebHistory } from 'vue-router'
import Layout from '../views/Layout.vue'

const routes = [
  {
    path: '/',
    component: Layout,
    redirect: '/dashboard',
    children: [
      { path: 'dashboard', name: 'Dashboard', component: () => import('../views/Dashboard.vue'), meta: { title: '看板' } },
      { path: 'reports/daily', name: 'DailyReport', component: () => import('../views/DailyReport.vue'), meta: { title: '日报表' } },
      { path: 'reports/weekly', name: 'WeeklyReport', component: () => import('../views/WeeklyReport.vue'), meta: { title: '周报表' } },
      { path: 'reports/monthly', name: 'MonthlyReport', component: () => import('../views/MonthlyReport.vue'), meta: { title: '月报表' } },
      { path: 'reports/annual', name: 'AnnualHeatmap', component: () => import('../views/AnnualHeatmap.vue'), meta: { title: '年度热力图' } },
      { path: 'reports/ai-review', name: 'AIReview', component: () => import('../views/AIReview.vue'), meta: { title: 'AI审核' } },
      { path: 'projects', name: 'Projects', component: () => import('../views/Projects.vue'), meta: { title: '项目管理' } },
      { path: 'filters', name: 'Filters', component: () => import('../views/Filters.vue'), meta: { title: '过滤规则' } },
      { path: 'settings', name: 'Settings', component: () => import('../views/Settings.vue'), meta: { title: '设置' } },
    ],
  },
]

const router = createRouter({
  history: createWebHistory(),
  routes,
})

export default router
