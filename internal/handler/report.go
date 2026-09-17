package handler

import (
	"fmt"
	"net/http"
	"time"

	"gitlab-push-stat/internal/database"
	"gitlab-push-stat/internal/model"
	"gitlab-push-stat/internal/report"

	"github.com/gin-gonic/gin"
)

// ReportHandler 报表处理器
type ReportHandler struct {
	service *report.Service
}

// NewReportHandler 创建报表处理器
func NewReportHandler() *ReportHandler {
	return &ReportHandler{service: report.NewService()}
}

// DailyReports 获取日报表
func (h *ReportHandler) DailyReports(c *gin.Context) {
	var q report.DailyReportQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if q.Page == 0 {
		q.Page = 1
	}
	if q.PageSize == 0 {
		q.PageSize = 20
	}

	reports, total, err := h.service.QueryDailyReports(q)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  reports,
		"total": total,
		"page":  q.Page,
	})
}

// WeeklyReports 获取周报表
func (h *ReportHandler) WeeklyReports(c *gin.Context) {
	var q report.WeeklyReportQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if q.Page == 0 {
		q.Page = 1
	}
	if q.PageSize == 0 {
		q.PageSize = 20
	}

	reports, total, err := h.service.QueryWeeklyReports(q)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  reports,
		"total": total,
		"page":  q.Page,
	})
}

// MonthlyReports 获取月报表
func (h *ReportHandler) MonthlyReports(c *gin.Context) {
	var q report.MonthlyReportQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if q.Page == 0 {
		q.Page = 1
	}
	if q.PageSize == 0 {
		q.PageSize = 20
	}

	reports, total, err := h.service.QueryMonthlyReports(q)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  reports,
		"total": total,
		"page":  q.Page,
	})
}

// GenerateDailyReport 手动生成日报表
func (h *ReportHandler) GenerateDailyReport(c *gin.Context) {
	var input struct {
		Date string `json:"date" binding:"required"` // YYYY-MM-DD
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请提供日期参数 date (YYYY-MM-DD)"})
		return
	}

	if err := h.service.GenerateDailyReport(input.Date); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "日报表生成成功"})
}

// GenerateWeeklyReport 手动生成周报表
func (h *ReportHandler) GenerateWeeklyReport(c *gin.Context) {
	var input struct {
		WeekStart string `json:"week_start" binding:"required"`
		WeekEnd   string `json:"week_end" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请提供 week_start 和 week_end 参数"})
		return
	}

	if err := h.service.GenerateWeeklyReport(input.WeekStart, input.WeekEnd); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "周报表生成成功"})
}

// DailyAuthorSummary 日报按人汇总
func (h *ReportHandler) DailyAuthorSummary(c *gin.Context) {
	var q report.DailyReportQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if q.Page == 0 {
		q.Page = 1
	}
	if q.PageSize == 0 {
		q.PageSize = 20
	}
	reports, total, err := h.service.QueryDailyAuthorSummary(q)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"data":  reports,
		"total": total,
		"page":  q.Page,
	})
}

// WeeklyAuthorSummary 周报按人汇总
func (h *ReportHandler) WeeklyAuthorSummary(c *gin.Context) {
	var q report.WeeklyReportQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if q.Page == 0 {
		q.Page = 1
	}
	if q.PageSize == 0 {
		q.PageSize = 20
	}
	reports, total, err := h.service.QueryWeeklyAuthorSummary(q)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"data":  reports,
		"total": total,
		"page":  q.Page,
	})
}

// MonthlyAuthorSummary 月报按人汇总
func (h *ReportHandler) MonthlyAuthorSummary(c *gin.Context) {
	var q report.MonthlyReportQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if q.Page == 0 {
		q.Page = 1
	}
	if q.PageSize == 0 {
		q.PageSize = 20
	}
	reports, total, err := h.service.QueryMonthlyAuthorSummary(q)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"data":  reports,
		"total": total,
		"page":  q.Page,
	})
}

// AnnualHeatmap 获取年度热力图数据
func (h *ReportHandler) AnnualHeatmap(c *gin.Context) {
	yearStr := c.DefaultQuery("year", "")
	if yearStr == "" {
		yearStr = fmt.Sprintf("%d", time.Now().Year())
	}

	var year int
	if _, err := fmt.Sscanf(yearStr, "%d", &year); err != nil || year < 2000 || year > 2100 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请提供有效的年份参数 year"})
		return
	}

	result, err := h.service.QueryAnnualHeatmap(year)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// ClearAllReports 清除所有日报、周报、月报数据
func (h *ReportHandler) ClearAllReports(c *gin.Context) {
	db := database.GetDB()

	var dailyCount, weeklyCount, monthlyCount int64
	db.Model(&model.DailyReport{}).Count(&dailyCount)
	db.Model(&model.WeeklyReport{}).Count(&weeklyCount)
	db.Model(&model.MonthlyReport{}).Count(&monthlyCount)

	db.Where("1 = 1").Delete(&model.DailyReport{})
	db.Where("1 = 1").Delete(&model.WeeklyReport{})
	db.Where("1 = 1").Delete(&model.MonthlyReport{})

	c.JSON(http.StatusOK, gin.H{
		"message": "已清除所有报表数据",
		"deleted": gin.H{
			"daily_reports":   dailyCount,
			"weekly_reports":  weeklyCount,
			"monthly_reports": monthlyCount,
		},
	})
}

// GenerateMonthlyReport 手动生成月报表
func (h *ReportHandler) GenerateMonthlyReport(c *gin.Context) {
	var input struct {
		YearMonth string `json:"year_month" binding:"required"` // YYYY-MM
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请提供 year_month 参数 (YYYY-MM)"})
		return
	}

	if err := h.service.GenerateMonthlyReport(input.YearMonth); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "月报表生成成功"})
}
