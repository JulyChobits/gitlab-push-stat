package handler

import (
	"net/http"
	"strconv"
	"time"

	"gitlab-push-stat/internal/collector"
	"gitlab-push-stat/internal/config"
	"gitlab-push-stat/internal/database"
	"gitlab-push-stat/internal/model"
	"gitlab-push-stat/internal/scheduler"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// DashboardHandler 看板处理器
type DashboardHandler struct {
	db        *gorm.DB
	scheduler *scheduler.Scheduler
	collector *collector.Collector
}

// NewDashboardHandler 创建看板处理器
func NewDashboardHandler(sched *scheduler.Scheduler, coll *collector.Collector) *DashboardHandler {
	return &DashboardHandler{
		db:        database.GetDB(),
		scheduler: sched,
		collector: coll,
	}
}

// Summary 看板汇总数据
func (h *DashboardHandler) Summary(c *gin.Context) {
	today := time.Now().Format("2006-01-02")
	yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")

	// 今日提交数
	var todayCommits int64
	h.db.Model(&model.Commit{}).
		Where("DATE(committed_at) = ? AND is_filtered = ?", today, false).
		Count(&todayCommits)

	// 昨日提交数
	var yesterdayCommits int64
	h.db.Model(&model.Commit{}).
		Where("DATE(committed_at) = ? AND is_filtered = ?", yesterday, false).
		Count(&yesterdayCommits)

	// 活跃开发者（最近7天）- 按 gitlab_name 去重
	weekAgo := time.Now().AddDate(0, 0, -7)
	var activeDevelopers int64
	h.db.Table("commits").
		Where("committed_at >= ? AND is_filtered = ?", weekAgo, false).
		Distinct("COALESCE(NULLIF(git_lab_name, ''), author_name)").
		Count(&activeDevelopers)

	// 本月代码净增行数和提交数
	monthStart := time.Now().Format("2006-01")
	var monthStats struct {
		NetAdditions int64
		CommitCount  int64
	}
	h.db.Model(&model.DailyReport{}).
		Select("COALESCE(SUM(net_additions), 0) as net_additions, COALESCE(SUM(commit_count), 0) as commit_count").
		Where("report_date LIKE ?", monthStart+"%").
		Scan(&monthStats)

	// 项目数量
	var projectCount int64
	h.db.Model(&model.Project{}).Where("enabled = ?", true).Count(&projectCount)

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"today_commits":     todayCommits,
			"yesterday_commits": yesterdayCommits,
			"active_developers": activeDevelopers,
			"month_net_lines":   monthStats.NetAdditions,
			"month_commits":     monthStats.CommitCount,
			"project_count":     projectCount,
		},
	})
}

// Trends 趋势数据
func (h *DashboardHandler) Trends(c *gin.Context) {
	days := 30
	// 前端传hours参数，转换为天数
	hoursStr := c.Query("hours")
	if hoursStr != "" {
		if hours, err := strconv.Atoi(hoursStr); err == nil {
			days = hours / 24
		}
	}
	daysStr := c.Query("days")
	if daysStr != "" {
		if d, err := strconv.Atoi(daysStr); err == nil {
			days = d
		}
	}

	startDate := time.Now().AddDate(0, 0, -days).Format("2006-01-02")

	type DailyStat struct {
		ReportDate   string `json:"report_date"`
		CommitCount  int64  `json:"commit_count"`
		Additions    int64  `json:"additions"`
		Deletions    int64  `json:"deletions"`
		NetAdditions int64  `json:"net_additions"`
	}

	var stats []DailyStat
	h.db.Model(&model.DailyReport{}).
		Select(`
			report_date,
			SUM(commit_count) as commit_count,
			SUM(additions) as additions,
			SUM(deletions) as deletions,
			SUM(net_additions) as net_additions
		`).
		Where("report_date >= ?", startDate).
		Group("report_date").
		Order("report_date ASC").
		Scan(&stats)

	c.JSON(http.StatusOK, gin.H{"data": stats})
}

// TrendsByAuthor 按人员维度的趋势数据
func (h *DashboardHandler) TrendsByAuthor(c *gin.Context) {
	days := 30
	hoursStr := c.Query("hours")
	if hoursStr != "" {
		if hours, err := strconv.Atoi(hoursStr); err == nil {
			days = hours / 24
		}
	}
	daysStr := c.Query("days")
	if daysStr != "" {
		if d, err := strconv.Atoi(daysStr); err == nil {
			days = d
		}
	}

	startDate := time.Now().AddDate(0, 0, -days).Format("2006-01-02")

	type AuthorDailyStat struct {
		ReportDate     string `json:"report_date"`
		AuthorName     string `json:"author_name"`
		CommitCount    int64  `json:"commit_count"`
		Additions      int64  `json:"additions"`
		Deletions      int64  `json:"deletions"`
		NetAdditions   int64  `json:"net_additions"`
		EffectiveLines int64  `json:"effective_lines"`
	}

	var stats []AuthorDailyStat
	h.db.Model(&model.DailyReport{}).
		Select(`
			report_date,
			author_name,
			SUM(commit_count) as commit_count,
			SUM(additions) as additions,
			SUM(deletions) as deletions,
			SUM(net_additions) as net_additions,
			COALESCE(SUM(effective_lines), 0) as effective_lines
		`).
		Where("report_date >= ?", startDate).
		Group("report_date, author_name").
		Order("report_date ASC").
		Scan(&stats)

	// 转换为按人员分组的结构
	dateSet := make(map[string]bool)
	var dates []string
	type AuthorSeries struct {
		Commits        []int64 `json:"commits"`
		NetAdditions   []int64 `json:"net_additions"`
		EffectiveLines []int64 `json:"effective_lines"`
	}
	authorMap := make(map[string]*AuthorSeries)

	for _, s := range stats {
		if !dateSet[s.ReportDate] {
			dateSet[s.ReportDate] = true
			dates = append(dates, s.ReportDate)
		}
	}

	for _, s := range stats {
		displayName := s.AuthorName
		if displayName == "" {
			displayName = "未知用户"
		}
		if _, ok := authorMap[displayName]; !ok {
			authorMap[displayName] = &AuthorSeries{
				Commits:        make([]int64, len(dates)),
				NetAdditions:   make([]int64, len(dates)),
				EffectiveLines: make([]int64, len(dates)),
			}
		}
	}

	for _, s := range stats {
		idx := -1
		for i, d := range dates {
			if d == s.ReportDate {
				idx = i
				break
			}
		}
		if idx >= 0 {
			displayName := s.AuthorName
			if displayName == "" {
				displayName = "未知用户"
			}
			series := authorMap[displayName]
			series.Commits[idx] = s.CommitCount
			series.NetAdditions[idx] = s.NetAdditions
			series.EffectiveLines[idx] = s.EffectiveLines
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"dates":   dates,
			"authors": authorMap,
		},
	})
}

// TopContributors 贡献排行榜
func (h *DashboardHandler) TopContributors(c *gin.Context) {
	days := 30
	hoursStr := c.Query("hours")
	if hoursStr != "" {
		if hours, err := strconv.Atoi(hoursStr); err == nil {
			days = hours / 24
		}
	}
	daysStr := c.Query("days")
	if daysStr != "" {
		if d, err := strconv.Atoi(daysStr); err == nil {
			days = d
		}
	}

	startDate := time.Now().AddDate(0, 0, -days).Format("2006-01-02")

	type Contributor struct {
		AuthorName   string `json:"author_name"`
		CommitCount  int64  `json:"commit_count"`
		Additions    int64  `json:"additions"`
		Deletions    int64  `json:"deletions"`
		NetAdditions int64  `json:"net_additions"`
	}

	var contributors []Contributor
	h.db.Model(&model.DailyReport{}).
		Select(`
			author_name,
			SUM(commit_count) as commit_count,
			SUM(additions) as additions,
			SUM(deletions) as deletions,
			SUM(net_additions) as net_additions
		`).
		Where("report_date >= ?", startDate).
		Group("author_name").
		Order("commit_count DESC").
		Limit(10).
		Scan(&contributors)

	c.JSON(http.StatusOK, gin.H{"data": contributors})
}

// ProjectDistribution 项目分布
func (h *DashboardHandler) ProjectDistribution(c *gin.Context) {
	type ProjectStat struct {
		ProjectID      uint   `json:"project_id"`
		ProjectName    string `json:"project_name"`
		CommitCount    int64  `json:"commit_count"`
		Additions      int64  `json:"additions"`
		Deletions      int64  `json:"deletions"`
		NetAdditions   int64  `json:"net_additions"`
		EffectiveLines int64  `json:"effective_lines"`
	}

	monthStart := time.Now().Format("2006-01")
	var stats []ProjectStat
	h.db.Table("daily_reports").
		Select(`
			daily_reports.project_id,
			projects.name as project_name,
			SUM(daily_reports.commit_count) as commit_count,
			SUM(daily_reports.additions) as additions,
			SUM(daily_reports.deletions) as deletions,
			SUM(daily_reports.net_additions) as net_additions,
			COALESCE(SUM(daily_reports.effective_lines), 0) as effective_lines
		`).
		Joins("LEFT JOIN projects ON projects.id = daily_reports.project_id").
		Where("daily_reports.report_date LIKE ?", monthStart+"%").
		Group("daily_reports.project_id, projects.name").
		Order("commit_count DESC").
		Scan(&stats)

	c.JSON(http.StatusOK, gin.H{"data": stats})
}

// GetConfig 返回当前系统配置
func (h *DashboardHandler) GetConfig(c *gin.Context) {
	cfg := config.Get()
	if cfg == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "配置未加载"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"server": gin.H{
				"port": cfg.Server.Port,
				"mode": cfg.Server.Mode,
			},
			"gitlab": gin.H{
				"url": cfg.GitLab.URL,
			},
			"database": gin.H{
				"path": cfg.Database.Path,
			},
			"schedule": gin.H{
				"sync":    cfg.Schedule.Sync,
				"daily":   cfg.Schedule.Daily,
				"weekly":  cfg.Schedule.Weekly,
				"monthly": cfg.Schedule.Monthly,
			},
		},
	})
}

// SyncStatus 同步状态
func (h *DashboardHandler) SyncStatus(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"scheduler_running": h.scheduler.IsRunning(),
		},
	})
}

// TriggerSync 手动触发同步
func (h *DashboardHandler) TriggerSync(c *gin.Context) {
	go func() {
		if err := h.scheduler.RunSyncNow(); err != nil {
			// 日志记录错误
		}
	}()
	c.JSON(http.StatusOK, gin.H{"message": "同步任务已触发"})
}

// TriggerDailyReport 手动触发日报表生成
func (h *DashboardHandler) TriggerDailyReport(c *gin.Context) {
	go func() {
		if err := h.scheduler.RunDailyReportNow(); err != nil {
			// 日志记录错误
		}
	}()
	c.JSON(http.StatusOK, gin.H{"message": "日报表生成任务已触发"})
}

// RecentCommitsByAuthor 按人员最近提交顺序
func (h *DashboardHandler) RecentCommitsByAuthor(c *gin.Context) {
	type AuthorRecentCommit struct {
		AuthorName      string `json:"author_name"`
		LastCommitAt    string `json:"last_commit_at"`
		LastCommitMsg   string `json:"last_commit_msg"`
		LastProjectName string `json:"last_project_name"`
	}

	var results []AuthorRecentCommit
	h.db.Table("commits").
		Select(`
			COALESCE(NULLIF(git_lab_name, ''), author_name) as author_name,
			MAX(committed_at) as last_commit_at,
			(SELECT c2.message FROM commits c2 
			 WHERE COALESCE(NULLIF(c2.git_lab_name, ''), c2.author_name) = COALESCE(NULLIF(commits.git_lab_name, ''), commits.author_name) 
			 AND c2.is_filtered = 0 ORDER BY c2.committed_at DESC LIMIT 1) as last_commit_msg,
			(SELECT p.name FROM commits c3 JOIN projects p ON p.id = c3.project_id 
			 WHERE COALESCE(NULLIF(c3.git_lab_name, ''), c3.author_name) = COALESCE(NULLIF(commits.git_lab_name, ''), commits.author_name) 
			 AND c3.is_filtered = 0 ORDER BY c3.committed_at DESC LIMIT 1) as last_project_name
		`).
		Where("is_filtered = ?", false).
		Group("COALESCE(NULLIF(git_lab_name, ''), author_name)").
		Order("last_commit_at DESC").
		Scan(&results)

	c.JSON(http.StatusOK, gin.H{"data": results})
}

// SearchGitLabProjects 搜索GitLab项目
func (h *DashboardHandler) SearchGitLabProjects(c *gin.Context) {
	search := c.Query("search")
	if search == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请提供搜索关键词"})
		return
	}

	projects, err := h.collector.SearchGitLabProjects(search)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": projects})
}

// TriggerFullHistorySync 全量同步所有历史提交数据（不触发报表生成和AI分析）
func (h *DashboardHandler) TriggerFullHistorySync(c *gin.Context) {
	go func() {
		if err := h.scheduler.RunFullHistorySync(); err != nil {
			// 仅记录日志，HTTP响应已立即返回
		}
	}()
	c.JSON(http.StatusOK, gin.H{"message": "全量历史同步任务已触发，请查看同步日志了解进度"})
}

// AddProjectFromGitLab 从GitLab添加项目
func (h *DashboardHandler) AddProjectFromGitLab(c *gin.Context) {
	var input struct {
		GitLabID int `json:"gitlab_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	project, err := h.collector.RefreshProject(input.GitLabID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": project})
}

// ProjectActivity 项目活跃度排名（近30天）
func (h *DashboardHandler) ProjectActivity(c *gin.Context) {
	days := 30
	hoursStr := c.Query("hours")
	if hoursStr != "" {
		if hours, err := strconv.Atoi(hoursStr); err == nil {
			days = hours / 24
		}
	}
	daysStr := c.Query("days")
	if daysStr != "" {
		if d, err := strconv.Atoi(daysStr); err == nil {
			days = d
		}
	}

	startDate := time.Now().AddDate(0, 0, -days).Format("2006-01-02")

	type ProjectActivityItem struct {
		ProjectID      uint   `json:"project_id"`
		ProjectName    string `json:"project_name"`
		CommitCount    int64  `json:"commit_count"`
		DeveloperCount int64  `json:"developer_count"`
	}

	var results []ProjectActivityItem
	h.db.Model(&model.DailyReport{}).
		Select(`
			daily_reports.project_id,
			projects.name as project_name,
			SUM(daily_reports.commit_count) as commit_count,
			COUNT(DISTINCT daily_reports.author_name) as developer_count
		`).
		Joins("LEFT JOIN projects ON projects.id = daily_reports.project_id").
		Where("daily_reports.report_date >= ?", startDate).
		Group("daily_reports.project_id, projects.name").
		Order("commit_count DESC").
		Limit(10).
		Scan(&results)

	c.JSON(http.StatusOK, gin.H{"data": results})
}

// ConsecutiveDays 连续提交天数排名
func (h *DashboardHandler) ConsecutiveDays(c *gin.Context) {
	type AuthorConsecutive struct {
		AuthorName      string `json:"author_name"`
		LastCommitDate  string `json:"last_commit_date"`
		ConsecutiveDays int64  `json:"consecutive_days"`
	}

	// 获取所有有提交记录的作者及其最后提交日期
	type AuthorLastCommit struct {
		AuthorName string
		LastDate   string
	}

	var authors []AuthorLastCommit
	h.db.Table("commits").
		Select("COALESCE(NULLIF(git_lab_name, ''), author_name) as author_name, MAX(DATE(committed_at)) as last_date").
		Where("is_filtered = ?", false).
		Group("COALESCE(NULLIF(git_lab_name, ''), author_name)").
		Order("last_date DESC").
		Scan(&authors)

	var results []AuthorConsecutive

	for _, a := range authors {
		var dates []string
		h.db.Table("commits").
			Select("DISTINCT DATE(committed_at) as commit_date").
			Where("COALESCE(NULLIF(git_lab_name, ''), author_name) = ? AND is_filtered = ?", a.AuthorName, false).
			Order("DATE(committed_at) DESC").
			Limit(365).
			Pluck("DATE(committed_at)", &dates)

		consecutive := int64(0)
		if len(dates) > 0 {
			consecutive = 1
			lastDate, _ := time.Parse("2006-01-02", a.LastDate)
			for i := 1; i < len(dates); i++ {
				expected := lastDate.AddDate(0, 0, -int(i)).Format("2006-01-02")
				if dates[i] == expected {
					consecutive++
				} else {
					break
				}
			}
		}

		results = append(results, AuthorConsecutive{
			AuthorName:      a.AuthorName,
			LastCommitDate:  a.LastDate,
			ConsecutiveDays: consecutive,
		})
	}

	// 按连续天数降序排列
	for i := 0; i < len(results); i++ {
		for j := i + 1; j < len(results); j++ {
			if results[j].ConsecutiveDays > results[i].ConsecutiveDays {
				results[i], results[j] = results[j], results[i]
			}
		}
	}

	if len(results) > 10 {
		results = results[:10]
	}

	c.JSON(http.StatusOK, gin.H{"data": results})
}
