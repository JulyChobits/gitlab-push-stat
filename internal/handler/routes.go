package handler

import (
	"gitlab-push-stat/internal/collector"
	"gitlab-push-stat/internal/filter"
	"gitlab-push-stat/internal/scheduler"

	"github.com/gin-gonic/gin"
)

// SetupRoutes 设置路由
func SetupRoutes(r *gin.Engine, sched *scheduler.Scheduler, coll *collector.Collector, filterEngine *filter.Engine) {
	// AI审核处理器
	aiReviewHandler := NewAIReviewHandler(sched)
	// API路由组
	api := r.Group("/api")
	{
		// 项目管理
		projectHandler := NewProjectHandler()
		projects := api.Group("/projects")
		{
			projects.GET("", projectHandler.List)
			projects.POST("", projectHandler.Create)
			projects.PUT("/:id", projectHandler.Update)
			projects.DELETE("/:id", projectHandler.Delete)
		}

		// 同步日志
		api.GET("/sync-logs", projectHandler.GetSyncLogs)

		// 报表查询
		reportHandler := NewReportHandler()
		reports := api.Group("/reports")
		{
			reports.GET("/daily", reportHandler.DailyReports)
			reports.GET("/weekly", reportHandler.WeeklyReports)
			reports.GET("/monthly", reportHandler.MonthlyReports)
			reports.GET("/daily/author-summary", reportHandler.DailyAuthorSummary)
			reports.GET("/weekly/author-summary", reportHandler.WeeklyAuthorSummary)
			reports.GET("/monthly/author-summary", reportHandler.MonthlyAuthorSummary)
			reports.POST("/daily/generate", reportHandler.GenerateDailyReport)
			reports.POST("/weekly/generate", reportHandler.GenerateWeeklyReport)
			reports.POST("/monthly/generate", reportHandler.GenerateMonthlyReport)
			reports.GET("/annual-heatmap", reportHandler.AnnualHeatmap)
			reports.DELETE("/clear-all", reportHandler.ClearAllReports)
		}

		// 过滤规则
		filterHandler := NewFilterHandler(filterEngine)
		filters := api.Group("/filters")
		{
			filters.GET("", filterHandler.List)
			filters.POST("", filterHandler.Create)
			filters.PUT("/:id", filterHandler.Update)
			filters.DELETE("/:id", filterHandler.Delete)
			filters.GET("/config", filterHandler.GetConfigRules)
		}

		// 看板
		dashboardHandler := NewDashboardHandler(sched, coll)
		dashboard := api.Group("/dashboard")
		{
			dashboard.GET("/summary", dashboardHandler.Summary)
			dashboard.GET("/trends", dashboardHandler.Trends)
			dashboard.GET("/trends-by-author", dashboardHandler.TrendsByAuthor)
			dashboard.GET("/top-contributors", dashboardHandler.TopContributors)
			dashboard.GET("/project-distribution", dashboardHandler.ProjectDistribution)
			dashboard.GET("/sync-status", dashboardHandler.SyncStatus)
			dashboard.GET("/config", dashboardHandler.GetConfig)
			dashboard.GET("/recent-commits", dashboardHandler.RecentCommitsByAuthor)
			dashboard.GET("/project-activity", dashboardHandler.ProjectActivity)
			dashboard.GET("/consecutive-days", dashboardHandler.ConsecutiveDays)
		}

		// 任务触发
		tasks := api.Group("/tasks")
		{
			tasks.POST("/sync", dashboardHandler.TriggerSync)
			tasks.POST("/sync-full-history", dashboardHandler.TriggerFullHistorySync)
			tasks.POST("/report/daily", dashboardHandler.TriggerDailyReport)
			tasks.GET("/gitlab/search", dashboardHandler.SearchGitLabProjects)
			tasks.POST("/gitlab/add", dashboardHandler.AddProjectFromGitLab)
		}

		// AI审核
		aiReviews := api.Group("/ai-reviews")
		{
			aiReviews.GET("/daily", aiReviewHandler.DailyReviews)
			aiReviews.GET("/author", aiReviewHandler.AuthorReviews)
			aiReviews.POST("/trigger", aiReviewHandler.TriggerReview)
			aiReviews.GET("/status", aiReviewHandler.ReviewStatus)
		}
	}
}
