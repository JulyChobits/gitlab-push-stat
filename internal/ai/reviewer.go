package ai

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"gitlab-push-stat/internal/config"
	"gitlab-push-stat/internal/database"
	"gitlab-push-stat/internal/filter"
	"gitlab-push-stat/internal/gitlab"
	"gitlab-push-stat/internal/model"

	"gorm.io/gorm"
)

// AIReviewResult AI审核结果（对应JSON结构）
type AIReviewResult struct {
	Difficulty       string      `json:"difficulty"`
	DifficultyReason string      `json:"difficulty_reason"`
	WorkHours        float64     `json:"work_hours"`
	WorkHoursDetail  interface{} `json:"work_hours_detail"`
	CodeQuality      string      `json:"code_quality"`
	QualityDetails   interface{} `json:"quality_details"`
	Highlights       string      `json:"highlights"`
	Suggestions      string      `json:"suggestions"`
	Summary          string      `json:"summary"`
}

// Reviewer AI审核服务
type Reviewer struct {
	client       *Client
	db           *gorm.DB
	gitClient    *gitlab.Client
	cfg          *config.Config
	filterEngine *filter.Engine
}

// NewReviewer 创建AI审核服务
func NewReviewer(cfg *config.Config, filterEngine *filter.Engine) *Reviewer {
	client := NewClient(
		cfg.AI.BaseURL,
		cfg.AI.APIKey,
		cfg.AI.Model,
		cfg.AI.MaxTokens,
		cfg.AI.Temperature,
		cfg.AI.Timeout,
	)

	gitClient := gitlab.NewClient(cfg.GitLab.URL, cfg.GitLab.Token)

	return &Reviewer{
		client:       client,
		db:           database.GetDB(),
		gitClient:    gitClient,
		cfg:          cfg,
		filterEngine: filterEngine,
	}
}

// ReviewDailyCommits 审核某天的提交记录
func (r *Reviewer) ReviewDailyCommits(reportDate string) error {
	if !r.cfg.AI.Enabled {
		log.Println("AI审核未启用，跳过")
		return nil
	}

	log.Printf("开始AI审核: %s", reportDate)

	// 解析日期
	localDate, err := time.ParseInLocation("2006-01-02", reportDate, time.Local)
	if err != nil {
		return fmt.Errorf("日期格式错误: %w", err)
	}
	dayStart := localDate.UTC()
	dayEnd := localDate.AddDate(0, 0, 1).UTC()
	dayStartTS := dayStart.Unix()
	dayEndTS := dayEnd.Unix()

	// 查询该天的提交记录（按项目和作者分组）
	var rows []struct {
		ProjectID   uint
		AuthorName  string
		CommitCount int
		Additions   int
		Deletions   int
	}

	err = r.db.Raw(`
		SELECT 
			project_id,
			COALESCE(NULLIF(git_lab_name, ''), author_name) as author_name,
			COUNT(*) as commit_count,
			SUM(filtered_additions) as additions,
			SUM(filtered_deletions) as deletions
		FROM commits 
		WHERE strftime('%s', committed_at) >= ? AND strftime('%s', committed_at) < ? 
			AND (is_filtered = 0 OR is_filtered = 'false')
		GROUP BY project_id, COALESCE(NULLIF(git_lab_name, ''), author_name)
	`, fmt.Sprintf("%d", dayStartTS), fmt.Sprintf("%d", dayEndTS)).Scan(&rows).Error

	if err != nil {
		return fmt.Errorf("查询提交数据失败: %w", err)
	}

	log.Printf("AI审核: 找到 %d 个项目+人员组合", len(rows))

	// 先删除该日期已有的审核结果
	r.db.Where("report_date = ?", reportDate).Delete(&model.AIReview{})

	// 存储每个作者的各项目审核结果（用于人员汇总）
	authorProjectReviews := make(map[string][]string)
	authorProjectCount := make(map[string]int)

	// 按项目+人员维度审核
	for _, row := range rows {
		log.Printf("AI审核: 项目ID=%d, 作者=%s, 提交数=%d", row.ProjectID, row.AuthorName, row.CommitCount)

		review, err := r.reviewByProject(reportDate, row.ProjectID, row.AuthorName, dayStartTS, dayEndTS)
		if err != nil {
			log.Printf("AI审核失败: 项目ID=%d, 作者=%s, 错误=%v", row.ProjectID, row.AuthorName, err)
			// 保存失败记录
			r.saveFailedReview(reportDate, row.ProjectID, row.AuthorName, "project", row.CommitCount, row.Additions, row.Deletions, err)
			continue
		}

		// 存储审核结果
		if err := r.db.Create(review).Error; err != nil {
			log.Printf("保存AI审核结果失败: %v", err)
			continue
		}

		// 收集用于人员汇总的数据
		key := row.AuthorName
		reviewSummary := fmt.Sprintf("**项目: %s**\n- 难度: %s\n- 工时: %.1f小时\n- 质量: %s\n- 总结: %s",
			r.getProjectName(row.ProjectID), review.Difficulty, review.WorkHours, review.CodeQuality, review.Summary)
		authorProjectReviews[key] = append(authorProjectReviews[key], reviewSummary)
		authorProjectCount[key] += row.CommitCount

		// 每个请求之间休眠，避免API限流
		time.Sleep(2 * time.Second)
	}

	// 按人员维度汇总审核
	for author, projectReviews := range authorProjectReviews {
		log.Printf("AI审核: 人员汇总 - %s, 项目数=%d", author, len(projectReviews))

		review, err := r.reviewByAuthor(reportDate, author, authorProjectCount[author], projectReviews, dayStartTS, dayEndTS)
		if err != nil {
			log.Printf("AI人员汇总审核失败: 作者=%s, 错误=%v", author, err)
			continue
		}

		// 存储审核结果（ProjectID=0表示人员汇总）
		if err := r.db.Create(review).Error; err != nil {
			log.Printf("保存AI人员汇总结果失败: %v", err)
		}

		time.Sleep(2 * time.Second)
	}

	log.Printf("AI审核完成: %s", reportDate)
	return nil
}

// reviewByProject 按项目+人员维度审核
func (r *Reviewer) reviewByProject(reportDate string, projectID uint, authorName string, dayStartTS, dayEndTS int64) (*model.AIReview, error) {
	// 获取项目信息
	project := r.getProject(projectID)
	if project == nil {
		return nil, fmt.Errorf("项目不存在: %d", projectID)
	}

	// 查询该天的commits
	var commits []model.Commit
	err := r.db.Where("project_id = ? AND COALESCE(NULLIF(git_lab_name, ''), author_name) = ? AND strftime('%s', committed_at) >= ? AND strftime('%s', committed_at) < ? AND (is_filtered = 0 OR is_filtered = 'false')",
		projectID, authorName, fmt.Sprintf("%d", dayStartTS), fmt.Sprintf("%d", dayEndTS)).
		Order("committed_at ASC").
		Find(&commits).Error

	if err != nil {
		return nil, fmt.Errorf("查询commits失败: %w", err)
	}

	// 获取每个commit的详细diff
	var commitInfos []CommitInfo
	totalAdditions := 0
	totalDeletions := 0

	for _, commit := range commits {
		totalAdditions += commit.FilteredAdditions
		totalDeletions += commit.FilteredDeletions

		// 获取commit详情
		detail, err := r.gitClient.GetCommitDetail(project.GitLabID, commit.ID)
		if err != nil {
			log.Printf("获取commit详情失败 %s: %v", commit.ID[:8], err)
			// 即使获取失败，也继续处理其他commit
			commitInfos = append(commitInfos, CommitInfo{
				SHA:         commit.ID,
				Message:     commit.Message,
				CommittedAt: commit.CommittedAt.Format("2006-01-02 15:04:05"),
				Additions:   commit.FilteredAdditions,
				Deletions:   commit.FilteredDeletions,
				Diffs:       []DiffInfo{},
			})
			continue
		}

		// 应用过滤规则，只保留有效文件的diff
		var diffs []DiffInfo
		filteredAdditions := 0
		filteredDeletions := 0
		for _, d := range detail.Diffs {
			filePath := d.NewPath
			if filePath == "" {
				filePath = d.OldPath
			}
			// 跳过被过滤的文件
			if r.filterEngine != nil && r.filterEngine.ShouldFilterFile(filePath) {
				continue
			}
			diffs = append(diffs, DiffInfo{
				OldPath:     d.OldPath,
				NewPath:     d.NewPath,
				NewFile:     d.NewFile,
				DeletedFile: d.DeletedFile,
				Diff:        d.Diff,
			})
			// 从diff内容统计有效行数
			adds, dels := countDiffStats(d.Diff)
			filteredAdditions += adds
			filteredDeletions += dels
		}

		// 如果没有过滤引擎，使用commit的filtered值
		if r.filterEngine == nil {
			filteredAdditions = commit.FilteredAdditions
			filteredDeletions = commit.FilteredDeletions
		}

		commitInfos = append(commitInfos, CommitInfo{
			SHA:         commit.ID,
			Message:     commit.Message,
			CommittedAt: commit.CommittedAt.Format("2006-01-02 15:04:05"),
			Additions:   filteredAdditions,
			Deletions:   filteredDeletions,
			Diffs:       diffs,
		})

		// 避免API限流
		time.Sleep(500 * time.Millisecond)
	}

	// 构建Prompt
	ctx := &ReviewContext{
		ReportDate:  reportDate,
		ProjectName: project.Name,
		AuthorName:  authorName,
		CommitCount: len(commits),
		Commits:     commitInfos,
		ReviewType:  "project",
	}

	messages := BuildProjectReviewPrompt(ctx)

	// 调用AI API
	resp, err := r.client.ChatCompletion(messages)
	if err != nil {
		return nil, err
	}

	// 解析结果
	result, err := parseAIResponse(resp.Choices[0].Message.Content)
	if err != nil {
		return nil, err
	}

	// 序列化详情
	workHoursDetailJSON, _ := json.Marshal(result.WorkHoursDetail)
	qualityDetailsJSON, _ := json.Marshal(result.QualityDetails)

	return &model.AIReview{
		ReportDate:       reportDate,
		ProjectID:        projectID,
		AuthorName:       authorName,
		ReviewType:       "project",
		CommitCount:      len(commits),
		TotalAdditions:   totalAdditions,
		TotalDeletions:   totalDeletions,
		EffectiveLines:   totalAdditions - totalDeletions,
		Difficulty:       result.Difficulty,
		DifficultyReason: result.DifficultyReason,
		WorkHours:        result.WorkHours,
		WorkHoursDetail:  string(workHoursDetailJSON),
		CodeQuality:      result.CodeQuality,
		QualityDetails:   string(qualityDetailsJSON),
		Highlights:       result.Highlights,
		Suggestions:      result.Suggestions,
		Summary:          result.Summary,
		RawResponse:      resp.Choices[0].Message.Content,
		Status:           "success",
		ReviewedAt:       time.Now(),
	}, nil
}

// reviewByAuthor 按人员维度汇总审核
func (r *Reviewer) reviewByAuthor(reportDate string, authorName string, totalCommitCount int, projectReviews []string, dayStartTS, dayEndTS int64) (*model.AIReview, error) {
	// 构建Prompt
	ctx := &ReviewContext{
		ReportDate:  reportDate,
		AuthorName:  authorName,
		CommitCount: len(projectReviews), // 项目数
		ReviewType:  "author",
	}

	messages := BuildAuthorReviewPrompt(ctx, projectReviews)

	// 调用AI API
	resp, err := r.client.ChatCompletion(messages)
	if err != nil {
		return nil, err
	}

	// 解析结果
	result, err := parseAIResponse(resp.Choices[0].Message.Content)
	if err != nil {
		return nil, err
	}

	// 序列化详情
	workHoursDetailJSON, _ := json.Marshal(result.WorkHoursDetail)
	qualityDetailsJSON, _ := json.Marshal(result.QualityDetails)

	// 获取该作者当天的有效代码变更
	var stats struct {
		Additions int
		Deletions int
	}
	r.db.Model(&model.Commit{}).
		Select("COALESCE(SUM(filtered_additions), 0) as additions, COALESCE(SUM(filtered_deletions), 0) as deletions").
		Where("COALESCE(NULLIF(git_lab_name, ''), author_name) = ? AND strftime('%s', committed_at) >= ? AND strftime('%s', committed_at) < ? AND (is_filtered = 0 OR is_filtered = 'false')",
			authorName, fmt.Sprintf("%d", dayStartTS), fmt.Sprintf("%d", dayEndTS)).
		Scan(&stats)

	return &model.AIReview{
		ReportDate:       reportDate,
		ProjectID:        0, // 0表示人员汇总
		AuthorName:       authorName,
		ReviewType:       "author",
		CommitCount:      totalCommitCount,
		TotalAdditions:   stats.Additions,
		TotalDeletions:   stats.Deletions,
		EffectiveLines:   stats.Additions - stats.Deletions,
		Difficulty:       result.Difficulty,
		DifficultyReason: result.DifficultyReason,
		WorkHours:        result.WorkHours,
		WorkHoursDetail:  string(workHoursDetailJSON),
		CodeQuality:      result.CodeQuality,
		QualityDetails:   string(qualityDetailsJSON),
		Highlights:       result.Highlights,
		Suggestions:      result.Suggestions,
		Summary:          result.Summary,
		RawResponse:      resp.Choices[0].Message.Content,
		Status:           "success",
		ReviewedAt:       time.Now(),
	}, nil
}

// saveFailedReview 保存失败的审核记录
func (r *Reviewer) saveFailedReview(reportDate string, projectID uint, authorName string, reviewType string, commitCount, additions, deletions int, err error) {
	review := &model.AIReview{
		ReportDate:     reportDate,
		ProjectID:      projectID,
		AuthorName:     authorName,
		ReviewType:     reviewType,
		CommitCount:    commitCount,
		TotalAdditions: additions,
		TotalDeletions: deletions,
		Status:         "failed",
		ErrorMessage:   err.Error(),
		ReviewedAt:     time.Now(),
	}
	r.db.Create(review)
}

// getProject 获取项目信息
func (r *Reviewer) getProject(projectID uint) *model.Project {
	var project model.Project
	if err := r.db.First(&project, projectID).Error; err != nil {
		return nil
	}
	return &project
}

// getProjectName 获取项目名称
func (r *Reviewer) getProjectName(projectID uint) string {
	project := r.getProject(projectID)
	if project != nil {
		return project.Name
	}
	return fmt.Sprintf("项目#%d", projectID)
}

// countDiffStats 从diff内容统计新增/删除行数
func countDiffStats(diffContent string) (additions, deletions int) {
	lines := strings.Split(diffContent, "\n")
	for _, line := range lines {
		if len(line) == 0 {
			continue
		}
		// 跳过diff头和@@行
		if strings.HasPrefix(line, "diff ") || strings.HasPrefix(line, "index ") ||
			strings.HasPrefix(line, "---") || strings.HasPrefix(line, "+++") ||
			strings.HasPrefix(line, "@@") {
			continue
		}
		if strings.HasPrefix(line, "+") {
			additions++
		} else if strings.HasPrefix(line, "-") {
			deletions++
		}
	}
	return
}

// parseAIResponse 解析AI响应
func parseAIResponse(content string) (*AIReviewResult, error) {
	jsonStr, err := ExtractJSON(content)
	if err != nil {
		return nil, err
	}

	var result AIReviewResult
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return nil, fmt.Errorf("解析AI响应JSON失败: %w, content: %s", err, content)
	}

	// 设置默认值
	if result.Difficulty == "" {
		result.Difficulty = "中等"
	}
	if result.CodeQuality == "" {
		result.CodeQuality = "良好"
	}
	if result.WorkHours <= 0 {
		result.WorkHours = 1.0
	}

	return &result, nil
}
