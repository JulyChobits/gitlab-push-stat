package model

import (
	"time"

	"gorm.io/gorm"
)

// Project 项目配置表
type Project struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	GitLabID  int            `gorm:"not null;uniqueIndex" json:"gitlab_id"`
	Name      string         `gorm:"size:255;not null" json:"name"`
	Path      string         `gorm:"size:255" json:"path"`
	WebURL    string         `gorm:"size:500" json:"web_url"`
	Enabled   bool           `gorm:"default:true" json:"enabled"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// Commit 提交记录表
type Commit struct {
	ID                string    `gorm:"primarykey;size:40" json:"id"` // commit SHA
	ProjectID         uint      `gorm:"index;not null" json:"project_id"`
	Project           Project   `gorm:"foreignKey:ProjectID" json:"-"`
	AuthorName        string    `gorm:"size:255;index" json:"author_name"`
	AuthorEmail       string    `gorm:"size:255" json:"author_email"`
	GitLabName        string    `gorm:"size:255;index" json:"gitlab_name"`     // GitLab真实姓名（从Events API获取）
	GitLabUsername    string    `gorm:"size:255;index" json:"gitlab_username"` // GitLab用户名（从Events API获取）
	Message           string    `gorm:"type:text" json:"message"`
	CommittedAt       time.Time `gorm:"index" json:"committed_at"`
	Additions         int       `gorm:"default:0" json:"additions"`
	Deletions         int       `gorm:"default:0" json:"deletions"`
	FilteredAdditions int       `gorm:"default:0" json:"filtered_additions"`
	FilteredDeletions int       `gorm:"default:0" json:"filtered_deletions"`
	IsFiltered        bool      `gorm:"default:false" json:"is_filtered"` // 整个提交是否被过滤
	CreatedAt         time.Time `json:"created_at"`
}

// AuthorMapping 邮箱到显示名称的映射表
type AuthorMapping struct {
	ID          uint           `gorm:"primarykey" json:"id"`
	Email       string         `gorm:"size:255;uniqueIndex;not null" json:"email"`
	DisplayName string         `gorm:"size:255;not null" json:"display_name"`
	Username    string         `gorm:"size:255" json:"username"`             // GitLab username
	Source      string         `gorm:"size:20;default:gitlab" json:"source"` // gitlab/commit/manual
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

// DailyReport 日度报表（固化数据）
type DailyReport struct {
	ID             uint      `gorm:"primarykey" json:"id"`
	ProjectID      uint      `gorm:"index;not null" json:"project_id"`
	Project        Project   `gorm:"foreignKey:ProjectID" json:"project"`
	AuthorEmail    string    `gorm:"size:255;index" json:"author_email"`
	AuthorName     string    `gorm:"size:255;index;not null" json:"author_name"`
	ReportDate     string    `gorm:"size:10;index;not null" json:"report_date"` // YYYY-MM-DD
	CommitCount    int       `gorm:"default:0" json:"commit_count"`
	Additions      int       `gorm:"default:0" json:"additions"`
	Deletions      int       `gorm:"default:0" json:"deletions"`
	NetAdditions   int       `gorm:"default:0" json:"net_additions"`
	EffectiveLines int       `gorm:"default:0" json:"effective_lines"`
	CreatedAt      time.Time `json:"created_at"`
}

// WeeklyReport 周度报表
type WeeklyReport struct {
	ID             uint      `gorm:"primarykey" json:"id"`
	ProjectID      uint      `gorm:"index;not null" json:"project_id"`
	Project        Project   `gorm:"foreignKey:ProjectID" json:"project"`
	AuthorEmail    string    `gorm:"size:255;index" json:"author_email"`
	AuthorName     string    `gorm:"size:255;index;not null" json:"author_name"`
	WeekStart      string    `gorm:"size:10;index;not null" json:"week_start"` // YYYY-MM-DD
	WeekEnd        string    `gorm:"size:10;not null" json:"week_end"`
	CommitCount    int       `gorm:"default:0" json:"commit_count"`
	Additions      int       `gorm:"default:0" json:"additions"`
	Deletions      int       `gorm:"default:0" json:"deletions"`
	NetAdditions   int       `gorm:"default:0" json:"net_additions"`
	EffectiveLines int       `gorm:"default:0" json:"effective_lines"`
	CreatedAt      time.Time `json:"created_at"`
}

// MonthlyReport 月度报表
type MonthlyReport struct {
	ID             uint      `gorm:"primarykey" json:"id"`
	ProjectID      uint      `gorm:"index;not null" json:"project_id"`
	Project        Project   `gorm:"foreignKey:ProjectID" json:"project"`
	AuthorEmail    string    `gorm:"size:255;index" json:"author_email"`
	AuthorName     string    `gorm:"size:255;index;not null" json:"author_name"`
	YearMonth      string    `gorm:"size:7;index;not null" json:"year_month"` // YYYY-MM
	CommitCount    int       `gorm:"default:0" json:"commit_count"`
	Additions      int       `gorm:"default:0" json:"additions"`
	Deletions      int       `gorm:"default:0" json:"deletions"`
	NetAdditions   int       `gorm:"default:0" json:"net_additions"`
	EffectiveLines int       `gorm:"default:0" json:"effective_lines"`
	CreatedAt      time.Time `json:"created_at"`
}

// FilterRule 过滤规则配置表
type FilterRule struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	RuleType  string         `gorm:"size:50;not null" json:"rule_type"` // path/extension/filename/message
	Pattern   string         `gorm:"size:500;not null" json:"pattern"`
	Enabled   bool           `gorm:"default:true" json:"enabled"`
	Remark    string         `gorm:"size:255" json:"remark"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// SyncLog 同步日志
type SyncLog struct {
	ID          uint      `gorm:"primarykey" json:"id"`
	ProjectID   uint      `gorm:"index" json:"project_id"`
	StartTime   time.Time `json:"start_time"`
	EndTime     time.Time `json:"end_time"`
	Status      string    `gorm:"size:20" json:"status"` // success/failed/running
	CommitCount int       `json:"commit_count"`
	ErrorMsg    string    `gorm:"type:text" json:"error_msg"`
	CreatedAt   time.Time `json:"created_at"`
}

// AIReview AI审核结果表
type AIReview struct {
	ID             uint    `gorm:"primarykey" json:"id"`
	ReportDate     string  `gorm:"size:10;index;not null" json:"report_date"` // YYYY-MM-DD
	ProjectID      uint    `gorm:"index" json:"project_id"`                   // 项目ID（0表示人员汇总）
	Project        Project `gorm:"foreignKey:ProjectID" json:"project"`
	AuthorName     string  `gorm:"size:255;index;not null" json:"author_name"` // 作者名称
	ReviewType     string  `gorm:"size:20;not null" json:"review_type"`        // project（项目维度）/ author（人员汇总）
	CommitCount    int     `gorm:"default:0" json:"commit_count"`              // 审查的commit数量
	TotalAdditions int     `gorm:"default:0" json:"total_additions"`           // 总新增行数
	TotalDeletions int     `gorm:"default:0" json:"total_deletions"`           // 总删除行数
	EffectiveLines int     `gorm:"default:0" json:"effective_lines"`           // 有效行数（新增-删除）

	// AI评估结果
	Difficulty       string  `gorm:"size:20" json:"difficulty"`           // 难度：简单/中等/复杂/高复杂
	DifficultyReason string  `gorm:"type:text" json:"difficulty_reason"`  // 难度评估理由
	WorkHours        float64 `gorm:"type:decimal(5,2)" json:"work_hours"` // 预估工时（小时）
	WorkHoursDetail  string  `gorm:"type:text" json:"work_hours_detail"`  // 工时明细JSON
	CodeQuality      string  `gorm:"size:20" json:"code_quality"`         // 代码质量：优秀/良好/一般/待改进
	QualityDetails   string  `gorm:"type:text" json:"quality_details"`    // 质量详情JSON

	// 代码审查内容
	Highlights  string `gorm:"type:text" json:"highlights"`   // 代码亮点
	Suggestions string `gorm:"type:text" json:"suggestions"`  // 改进建议
	Summary     string `gorm:"type:text" json:"summary"`      // 工作总结
	RawResponse string `gorm:"type:text" json:"raw_response"` // AI原始响应

	Status       string    `gorm:"size:20;default:pending" json:"status"` // pending/success/failed
	ErrorMessage string    `gorm:"type:text" json:"error_message"`        // 错误信息
	ReviewedAt   time.Time `json:"reviewed_at"`                           // 审核时间
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// TableName 设置表名
func (AuthorMapping) TableName() string { return "author_mappings" }
func (Project) TableName() string       { return "projects" }
func (Commit) TableName() string        { return "commits" }
func (DailyReport) TableName() string   { return "daily_reports" }
func (WeeklyReport) TableName() string  { return "weekly_reports" }
func (MonthlyReport) TableName() string { return "monthly_reports" }
func (FilterRule) TableName() string    { return "filter_rules" }
func (SyncLog) TableName() string       { return "sync_logs" }
func (AIReview) TableName() string      { return "ai_reviews" }
