package handler

import (
	"net/http"
	"strconv"

	"gitlab-push-stat/internal/database"
	"gitlab-push-stat/internal/model"
	"gitlab-push-stat/internal/scheduler"

	"github.com/gin-gonic/gin"
)

// AIReviewHandler AI审核处理器
type AIReviewHandler struct {
	sched *scheduler.Scheduler
}

// NewAIReviewHandler 创建AI审核处理器
func NewAIReviewHandler(sched *scheduler.Scheduler) *AIReviewHandler {
	return &AIReviewHandler{sched: sched}
}

// DailyReviews 获取某天的AI审核结果
func (h *AIReviewHandler) DailyReviews(c *gin.Context) {
	date := c.Query("date")
	if date == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "date参数不能为空"})
		return
	}

	projectIDStr := c.Query("project_id")
	authorName := c.Query("author_name")
	reviewType := c.Query("review_type") // project/author/空=全部

	db := database.GetDB()
	query := db.Model(&model.AIReview{}).
		Where("report_date = ?", date).
		Preload("Project")

	if projectIDStr != "" {
		pid, err := strconv.ParseUint(projectIDStr, 10, 64)
		if err == nil {
			query = query.Where("project_id = ?", pid)
		}
	}
	if authorName != "" {
		query = query.Where("author_name LIKE ?", "%"+authorName+"%")
	}
	if reviewType != "" {
		query = query.Where("review_type = ?", reviewType)
	}

	var reviews []model.AIReview
	if err := query.Order("review_type DESC, project_id ASC").Find(&reviews).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  reviews,
		"total": len(reviews),
	})
}

// AuthorReviews 获取某人某天的AI审核汇总
func (h *AIReviewHandler) AuthorReviews(c *gin.Context) {
	date := c.Query("date")
	authorName := c.Query("author_name")

	if date == "" || authorName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "date和author_name参数不能为空"})
		return
	}

	db := database.GetDB()
	var reviews []model.AIReview
	if err := db.Where("report_date = ? AND author_name LIKE ?", date, "%"+authorName+"%").
		Preload("Project").
		Order("review_type DESC, project_id ASC").
		Find(&reviews).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  reviews,
		"total": len(reviews),
	})
}

// TriggerReview 手动触发AI审核
func (h *AIReviewHandler) TriggerReview(c *gin.Context) {
	date := c.Query("date")
	if date == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "date参数不能为空"})
		return
	}

	if h.sched.GetAIReviewer() == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "AI审核功能未启用"})
		return
	}

	// 异步执行
	go func() {
		if err := h.sched.RunAIReviewNow(date); err != nil {
			// 只记录日志，不阻塞响应
		}
	}()

	c.JSON(http.StatusOK, gin.H{
		"message": "AI审核任务已提交，请稍后刷新查看结果",
		"date":    date,
	})
}

// ReviewStatus 获取AI审核状态
func (h *AIReviewHandler) ReviewStatus(c *gin.Context) {
	date := c.Query("date")
	if date == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "date参数不能为空"})
		return
	}

	db := database.GetDB()

	var total int64
	var success int64
	var failed int64
	var pending int64

	db.Model(&model.AIReview{}).Where("report_date = ?", date).Count(&total)
	db.Model(&model.AIReview{}).Where("report_date = ? AND status = ?", date, "success").Count(&success)
	db.Model(&model.AIReview{}).Where("report_date = ? AND status = ?", date, "failed").Count(&failed)
	db.Model(&model.AIReview{}).Where("report_date = ? AND status = ?", date, "pending").Count(&pending)

	enabled := h.sched.GetAIReviewer() != nil

	c.JSON(http.StatusOK, gin.H{
		"enabled": enabled,
		"total":   total,
		"success": success,
		"failed":  failed,
		"pending": pending,
	})
}
