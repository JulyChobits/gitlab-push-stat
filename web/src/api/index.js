import axios from 'axios'

const http = axios.create({ baseURL: '/api', timeout: 30000 })

http.interceptors.response.use(
  res => res,
  err => { return Promise.reject(err) }
)

// Projects
export const getProjects = () => http.get('/projects')
export const createProject = (data) => http.post('/projects', data)
export const updateProject = (id, data) => http.put('/projects/' + id, data)
export const deleteProject = (id) => http.delete('/projects/' + id)

// Sync Logs
export const getSyncLogs = (params) => http.get('/sync-logs', { params })

// Reports
export const getDailyReports = (params) => http.get('/reports/daily', { params })
export const getWeeklyReports = (params) => http.get('/reports/weekly', { params })
export const getMonthlyReports = (params) => http.get('/reports/monthly', { params })
export const generateDailyReport = (date) => http.post('/reports/daily/generate', { date })
export const generateWeeklyReport = (start, end) => http.post('/reports/weekly/generate', { week_start: start, week_end: end })
export const generateMonthlyReport = (month) => http.post('/reports/monthly/generate', { year_month: month })
export const getDailyAuthorSummary = (params) => http.get('/reports/daily/author-summary', { params })
export const getWeeklyAuthorSummary = (params) => http.get('/reports/weekly/author-summary', { params })
export const getMonthlyAuthorSummary = (params) => http.get('/reports/monthly/author-summary', { params })
export const getAnnualHeatmap = (year) => http.get('/reports/annual-heatmap', { params: { year } })
export const clearAllReports = () => http.delete('/reports/clear-all')

// Filters
export const getFilters = () => http.get('/filters')
export const createFilter = (data) => http.post('/filters', data)
export const updateFilter = (id, data) => http.put('/filters/' + id, data)
export const deleteFilter = (id) => http.delete('/filters/' + id)
export const getFilterConfigRules = () => http.get('/filters/config')

// Author Mappings
export const getAuthorMappings = () => http.get('/author-mappings')
export const updateAuthorMapping = (id, data) => http.put('/author-mappings/' + id, data)
export const backfillAuthorMappings = () => http.post('/author-mappings/backfill')

// Dashboard
export const getSummary = () => http.get('/dashboard/summary')
export const getTrends = (hours) => http.get('/dashboard/trends', { params: { hours } })
export const getTrendsByAuthor = (hours) => http.get('/dashboard/trends-by-author', { params: { hours } })
export const getTopContributors = (hours) => http.get('/dashboard/top-contributors', { params: { hours } })
export const getProjectDistribution = () => http.get('/dashboard/project-distribution')
export const getRecentCommits = () => http.get('/dashboard/recent-commits')
export const getProjectActivity = (hours) => http.get('/dashboard/project-activity', { params: { hours } })
export const getConsecutiveDays = () => http.get('/dashboard/consecutive-days')
export const getSyncStatus = () => http.get('/dashboard/sync-status')
export const getSystemConfig = () => http.get('/dashboard/config')

// Tasks
export const triggerSync = () => http.post('/tasks/sync')
export const triggerFullHistorySync = () => http.post('/tasks/sync-full-history')
export const triggerDailyReport = () => http.post('/tasks/report/daily')
export const searchGitLabProjects = (keyword) => http.get('/tasks/gitlab/search', { params: { search: keyword } })
export const addProjectFromGitLab = (data) => http.post('/tasks/gitlab/add', data)

// AI Reviews
export const getAIReviewsDaily = (params) => http.get('/ai-reviews/daily', { params })
export const getAIReviewsAuthor = (params) => http.get('/ai-reviews/author', { params })
export const triggerAIReview = (date) => http.post('/ai-reviews/trigger', null, { params: { date } })
export const getAIReviewStatus = (date) => http.get('/ai-reviews/status', { params: { date } })

export default http
