package handler

import (
	"net/http"
	"strconv"

	"gitlab-push-stat/internal/config"
	"gitlab-push-stat/internal/database"
	"gitlab-push-stat/internal/gitlab"
	"gitlab-push-stat/internal/model"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// AuthorMappingHandler 用户映射处理器
type AuthorMappingHandler struct {
	db *gorm.DB
}

// NewAuthorMappingHandler 创建用户映射处理器
func NewAuthorMappingHandler() *AuthorMappingHandler {
	return &AuthorMappingHandler{
		db: database.GetDB(),
	}
}

// List 获取所有用户映射
func (h *AuthorMappingHandler) List(c *gin.Context) {
	var mappings []model.AuthorMapping
	h.db.Order("display_name ASC").Find(&mappings)

	// 如果没有映射记录，从commits表中获取所有邮箱
	if len(mappings) == 0 {
		var emails []struct {
			AuthorEmail string
			AuthorName  string
		}
		h.db.Table("commits").
			Select("DISTINCT author_email, author_name").
			Where("author_email != '' AND (is_filtered = 0 OR is_filtered = 'false')").
			Find(&emails)

		for _, e := range emails {
			mapping := model.AuthorMapping{
				Email:       e.AuthorEmail,
				DisplayName: e.AuthorName,
				Source:      "commit",
			}
			h.db.Create(&mapping)
			mappings = append(mappings, mapping)
		}
	}

	c.JSON(http.StatusOK, gin.H{"data": mappings})
}

// Update 更新用户映射
func (h *AuthorMappingHandler) Update(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的ID"})
		return
	}

	var input struct {
		DisplayName string `json:"display_name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var mapping model.AuthorMapping
	if err := h.db.First(&mapping, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "映射不存在"})
		return
	}

	mapping.DisplayName = input.DisplayName
	mapping.Source = "manual"
	h.db.Save(&mapping)

	c.JSON(http.StatusOK, gin.H{"data": mapping})
}

// Backfill 回填已有用户映射的GitLab用户信息
func (h *AuthorMappingHandler) Backfill(c *gin.Context) {
	var mappings []model.AuthorMapping
	h.db.Where("source != ? AND username = ''", "manual").Find(&mappings)

	// 如果为空，也尝试查找所有没有username的
	if len(mappings) == 0 {
		h.db.Where("username = '' OR username IS NULL").Find(&mappings)
	}

	client := gitlab.NewClient(
		config.Get().GitLab.URL,
		config.Get().GitLab.Token,
	)

	updated := 0
	failed := 0
	for _, m := range mappings {
		user, err := client.SearchUserByEmail(m.Email)
		if err != nil || user == nil {
			failed++
			continue
		}
		if user.Name != "" {
			m.DisplayName = user.Name
		}
		m.Username = user.Username
		if m.Source != "manual" {
			m.Source = "gitlab"
		}
		h.db.Save(&m)
		updated++
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "回填完成",
		"updated": updated,
		"failed":  failed,
		"total":   len(mappings),
	})
}

// GetDisplayName 根据邮箱获取显示名称
func GetDisplayName(db *gorm.DB, email string) string {
	if email == "" {
		return "未知用户"
	}

	var mapping model.AuthorMapping
	result := db.Where("email = ?", email).First(&mapping)
	if result.Error == nil {
		return mapping.DisplayName
	}

	// 如果没有映射，返回邮箱前缀
	for i, ch := range email {
		if ch == '@' {
			return email[:i]
		}
	}
	return email
}
