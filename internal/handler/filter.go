package handler

import (
	"net/http"

	"gitlab-push-stat/internal/database"
	"gitlab-push-stat/internal/filter"
	"gitlab-push-stat/internal/model"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// FilterHandler 过滤规则处理器
type FilterHandler struct {
	db     *gorm.DB
	engine *filter.Engine
}

// NewFilterHandler 创建过滤规则处理器
func NewFilterHandler(engine *filter.Engine) *FilterHandler {
	return &FilterHandler{db: database.GetDB(), engine: engine}
}

// List 获取过滤规则列表
func (h *FilterHandler) List(c *gin.Context) {
	var rules []model.FilterRule
	if err := h.db.Order("rule_type, created_at ASC").Find(&rules).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": rules})
}

// Create 创建过滤规则
func (h *FilterHandler) Create(c *gin.Context) {
	var input struct {
		RuleType string `json:"rule_type" binding:"required"` // path/extension/filename/message
		Pattern  string `json:"pattern" binding:"required"`
		Enabled  bool   `json:"enabled"`
		Remark   string `json:"remark"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	rule := model.FilterRule{
		RuleType: input.RuleType,
		Pattern:  input.Pattern,
		Enabled:  input.Enabled,
		Remark:   input.Remark,
	}

	if err := h.db.Create(&rule).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": rule})
}

// Update 更新过滤规则
func (h *FilterHandler) Update(c *gin.Context) {
	id := c.Param("id")
	var rule model.FilterRule

	if err := h.db.First(&rule, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "规则不存在"})
		return
	}

	var input struct {
		RuleType string `json:"rule_type"`
		Pattern  string `json:"pattern"`
		Enabled  *bool  `json:"enabled"`
		Remark   string `json:"remark"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updates := map[string]interface{}{}
	if input.RuleType != "" {
		updates["rule_type"] = input.RuleType
	}
	if input.Pattern != "" {
		updates["pattern"] = input.Pattern
	}
	if input.Enabled != nil {
		updates["enabled"] = *input.Enabled
	}
	if input.Remark != "" {
		updates["remark"] = input.Remark
	}

	if err := h.db.Model(&rule).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	h.db.First(&rule, id)
	c.JSON(http.StatusOK, gin.H{"data": rule})
}

// Delete 删除过滤规则
func (h *FilterHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	var rule model.FilterRule

	if err := h.db.First(&rule, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "规则不存在"})
		return
	}

	if err := h.db.Delete(&rule).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}

// GetConfigRules 获取配置文件中的过滤规则
func (h *FilterHandler) GetConfigRules(c *gin.Context) {
	if h.engine == nil {
		c.JSON(http.StatusOK, gin.H{"data": gin.H{"paths": []string{}, "extensions": []string{}, "filenames": []string{}, "messages": []string{}}})
		return
	}
	cfg := h.engine.GetConfig()
	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"paths":      cfg.ExcludePaths,
			"extensions": cfg.ExcludeExtensions,
			"filenames":  cfg.ExcludeFilenames,
			"messages":   cfg.ExcludeMessages,
		},
	})
}
