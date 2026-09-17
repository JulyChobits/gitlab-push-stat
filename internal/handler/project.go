package handler

import (
	"net/http"

	"gitlab-push-stat/internal/database"
	"gitlab-push-stat/internal/model"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// ProjectHandler 项目管理处理器
type ProjectHandler struct {
	db *gorm.DB
}

// NewProjectHandler 创建项目管理处理器
func NewProjectHandler() *ProjectHandler {
	return &ProjectHandler{db: database.GetDB()}
}

// List 获取项目列表
func (h *ProjectHandler) List(c *gin.Context) {
	var projects []model.Project
	if err := h.db.Order("name ASC").Find(&projects).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": projects})
}

// Create 创建项目
func (h *ProjectHandler) Create(c *gin.Context) {
	var input struct {
		GitLabID int    `json:"gitlab_id" binding:"required"`
		Name     string `json:"name" binding:"required"`
		Path     string `json:"path"`
		WebURL   string `json:"web_url"`
		Enabled  bool   `json:"enabled"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 检查是否存在软删除的记录（Unscoped 查询包括已删除的）
	var existing model.Project
	result := h.db.Unscoped().Where("gitlab_id = ?", input.GitLabID).First(&existing)
	if result.Error == nil {
		// 已存在（可能是软删除的），恢复并更新
		existing.Name = input.Name
		existing.Path = input.Path
		existing.WebURL = input.WebURL
		existing.Enabled = input.Enabled
		existing.DeletedAt = gorm.DeletedAt{Valid: false}
		h.db.Unscoped().Save(&existing)
		c.JSON(http.StatusOK, gin.H{"data": existing})
		return
	}

	project := model.Project{
		GitLabID: input.GitLabID,
		Name:     input.Name,
		Path:     input.Path,
		WebURL:   input.WebURL,
		Enabled:  input.Enabled,
	}

	if err := h.db.Create(&project).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": project})
}

// Update 更新项目
func (h *ProjectHandler) Update(c *gin.Context) {
	id := c.Param("id")
	var project model.Project

	if err := h.db.First(&project, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "项目不存在"})
		return
	}

	var input struct {
		Name    string `json:"name"`
		Path    string `json:"path"`
		WebURL  string `json:"web_url"`
		Enabled *bool  `json:"enabled"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updates := map[string]interface{}{}
	if input.Name != "" {
		updates["name"] = input.Name
	}
	if input.Path != "" {
		updates["path"] = input.Path
	}
	if input.WebURL != "" {
		updates["web_url"] = input.WebURL
	}
	if input.Enabled != nil {
		updates["enabled"] = *input.Enabled
	}

	if err := h.db.Model(&project).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	h.db.First(&project, id)
	c.JSON(http.StatusOK, gin.H{"data": project})
}

// Delete 删除项目
func (h *ProjectHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	var project model.Project

	if err := h.db.First(&project, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "项目不存在"})
		return
	}

	if err := h.db.Delete(&project).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}

// GetSyncLogs 获取同步日志
func (h *ProjectHandler) GetSyncLogs(c *gin.Context) {
	projectID := c.Query("project_id")

	var logs []model.SyncLog
	query := h.db.Order("created_at DESC").Limit(50)

	if projectID != "" {
		query = query.Where("project_id = ?", projectID)
	}

	if err := query.Find(&logs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": logs})
}
