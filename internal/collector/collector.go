package collector

import (
	"fmt"
	"log"
	"sync"
	"time"

	"gitlab-push-stat/internal/config"
	"gitlab-push-stat/internal/database"
	"gitlab-push-stat/internal/filter"
	"gitlab-push-stat/internal/gitlab"
	"gitlab-push-stat/internal/model"

	"gorm.io/gorm"
)

// Collector 数据采集器
type Collector struct {
	client *gitlab.Client
	filter *filter.Engine
	db     *gorm.DB
	cfg    *config.Config
	mu     sync.Mutex
}

// NewCollector 创建采集器
func NewCollector(cfg *config.Config, filterEngine *filter.Engine) *Collector {
	client := gitlab.NewClient(cfg.GitLab.URL, cfg.GitLab.Token)
	return &Collector{
		client: client,
		filter: filterEngine,
		db:     database.GetDB(),
		cfg:    cfg,
	}
}

// SyncProject 同步单个项目的提交数据
func (c *Collector) SyncProject(projectID uint, since, until time.Time) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// 获取项目信息
	var project model.Project
	if err := c.db.First(&project, projectID).Error; err != nil {
		return fmt.Errorf("项目不存在: %w", err)
	}

	log.Printf("开始同步项目: %s (GitLabID: %d), 时间范围: %s ~ %s",
		project.Name, project.GitLabID, since.Format("2006-01-02"), until.Format("2006-01-02"))

	// 创建同步日志
	syncLog := model.SyncLog{
		ProjectID: projectID,
		StartTime: time.Now(),
		Status:    "running",
	}
	c.db.Create(&syncLog)

	// 拉取提交，每拉取一页立即匹配 Events
	totalCommits := 0
	page := 1
	perPage := 100

	for {
		commits, err := c.client.ListCommits(project.GitLabID, since, until, page, perPage)
		if err != nil {
			syncLog.Status = "failed"
			syncLog.ErrorMsg = err.Error()
			syncLog.EndTime = time.Now()
			c.db.Save(&syncLog)
			return fmt.Errorf("拉取提交失败: %w", err)
		}

		log.Printf("从GitLab获取到 %d 个提交 (page=%d)", len(commits), page)
		for _, commit := range commits {
			log.Printf("提交: %s, 作者: %s, committed_at: %s", commit.ID[:8], commit.AuthorName, commit.CommittedDate.Format(time.RFC3339))
			if err := c.processCommit(projectID, project.GitLabID, &commit); err != nil {
				log.Printf("处理提交 %s 失败: %v", commit.ID[:8], err)
			} else {
				totalCommits++
			}
		}

		// 每拉取一页 commits 后，立即拉取这一页时间范围内的 Events 并匹配
		if len(commits) > 0 {
			c.enrichCommitsByEvents(project.GitLabID, commits)
		}

		if len(commits) < perPage {
			break
		}
		page++

		// 避免 API 限流
		time.Sleep(500 * time.Millisecond)
	}

	// 更新同步日志
	syncLog.Status = "success"
	syncLog.CommitCount = totalCommits
	syncLog.EndTime = time.Now()
	c.db.Save(&syncLog)

	log.Printf("项目 %s 同步完成，共处理 %d 个提交", project.Name, totalCommits)

	return nil
}

// processCommit 处理单个提交
func (c *Collector) processCommit(projectID uint, gitlabID int, commit *gitlab.Commit) error {
	// 检查是否已存在
	var count int64
	c.db.Model(&model.Commit{}).Where("id = ?", commit.ID).Count(&count)
	if count > 0 {
		return nil // 已存在，跳过
	}

	// 检查提交信息是否需要过滤
	isFiltered := c.filter.ShouldFilterCommit(commit.Message)

	// 获取提交详情（diff）
	var filteredAdds, filteredDels int
	var adds, dels int

	if commit.Stats != nil {
		adds = commit.Stats.Additions
		dels = commit.Stats.Deletions
	}

	if !isFiltered {
		detail, err := c.client.GetCommitDetail(gitlabID, commit.ID)
		if err != nil {
			log.Printf("获取提交详情失败 %s: %v", commit.ID[:8], err)
			// 如果获取详情失败，使用commit的stats
			filteredAdds = adds
			filteredDels = dels
		} else {
			filteredAdds, filteredDels = c.filter.FilterDiffs(detail.Diffs)
			// 如果stats为空，用diff统计
			if adds == 0 && dels == 0 {
				adds = detail.Stats.Additions
				dels = detail.Stats.Deletions
			}
		}
	}

	// 保存提交记录
	dbCommit := model.Commit{
		ID:                commit.ID,
		ProjectID:         projectID,
		AuthorName:        commit.AuthorName,
		AuthorEmail:       commit.AuthorEmail,
		Message:           commit.Message,
		CommittedAt:       commit.CommittedDate,
		Additions:         adds,
		Deletions:         dels,
		FilteredAdditions: filteredAdds,
		FilteredDeletions: filteredDels,
		IsFiltered:        isFiltered,
	}

	return c.db.Create(&dbCommit).Error
}

// ensureAuthorMapping 确保用户在映射表中存在（优先使用GitLab账号邮箱）
func (c *Collector) ensureAuthorMapping(commitEmail, authorName string) {
	if authorName == "" && commitEmail == "" {
		return
	}

	// 第一步：通过 authorName 在GitLab中查找用户（获取真实的GitLab账号邮箱）
	var gitlabUser *gitlab.User
	var err error

	if authorName != "" {
		// 先尝试按用户名搜索（authorName 可能是 username，如 "fucc1"）
		gitlabUser, err = c.client.SearchUserByUsername(authorName)
		if err != nil {
			log.Printf("从GitLab按用户名搜索失败 %s: %v", authorName, err)
		}
	}

	// 如果按用户名没找到，尝试按邮箱搜索（commit邮箱可能恰好是GitLab邮箱）
	if gitlabUser == nil && commitEmail != "" {
		gitlabUser, err = c.client.SearchUserByEmail(commitEmail)
		if err != nil {
			log.Printf("从GitLab按邮箱搜索失败 %s: %v", commitEmail, err)
		}
	}

	// 确定最终使用的邮箱和显示名
	var finalEmail, displayName, username, source string

	if gitlabUser != nil && gitlabUser.Email != "" {
		// GitLab用户找到了，使用GitLab账号邮箱（可信来源）
		finalEmail = gitlabUser.Email
		displayName = gitlabUser.Name
		username = gitlabUser.Username
		source = "gitlab"
		log.Printf("GitLab匹配: %s -> %s (%s)", authorName, finalEmail, username)
	} else {
		// GitLab中找不到该用户
		// 检查 commit 邮箱是否是公司邮箱（包含 @kfb.cn 等可信域名）
		if commitEmail != "" && c.isTrustedEmail(commitEmail) {
			finalEmail = commitEmail
			displayName = authorName
			source = "commit"
			log.Printf("使用commit邮箱（公司域名）: %s -> %s", authorName, finalEmail)
		} else {
			// 非公司邮箱，不创建映射记录
			log.Printf("跳过非公司邮箱: %s (%s)，该邮箱来自本地git配置，不可信", commitEmail, authorName)
			return
		}
	}

	// 检查是否已存在该邮箱的映射
	var existing model.AuthorMapping
	result := c.db.Where("email = ?", finalEmail).First(&existing)
	if result.Error == nil {
		// 已存在，如果GitLab来源优先级更高则更新
		if source == "gitlab" && existing.Source != "manual" {
			existing.DisplayName = displayName
			existing.Username = username
			existing.Source = source
			c.db.Save(&existing)
		}
		return
	}

	// 创建新的映射记录
	mapping := model.AuthorMapping{
		Email:       finalEmail,
		DisplayName: displayName,
		Username:    username,
		Source:      source,
	}
	c.db.Create(&mapping)
}

// isTrustedEmail 判断邮箱是否是可信的公司邮箱
func (c *Collector) isTrustedEmail(email string) bool {
	trustedDomains := []string{"@kfb.cn"}
	for _, domain := range trustedDomains {
		if len(email) > len(domain) && email[len(email)-len(domain):] == domain {
			return true
		}
	}
	return false
}

// SyncAllProjects 同步所有启用的项目
func (c *Collector) SyncAllProjects(since, until time.Time) error {
	var projects []model.Project
	if err := c.db.Where("enabled = ?", true).Find(&projects).Error; err != nil {
		return err
	}

	for _, project := range projects {
		if err := c.SyncProject(project.ID, since, until); err != nil {
			log.Printf("同步项目 %s 失败: %v", project.Name, err)
		}
		// 每个项目之间休眠2秒，避免API限流
		time.Sleep(2 * time.Second)
	}

	return nil
}

// GetLastSyncTime 获取项目最后同步时间
func (c *Collector) GetLastSyncTime(projectID uint) (time.Time, error) {
	var commit model.Commit
	err := c.db.Where("project_id = ?", projectID).
		Order("committed_at desc").
		First(&commit).Error

	if err == gorm.ErrRecordNotFound {
		// 没有历史记录，返回30天前
		return time.Now().AddDate(0, 0, -30), nil
	}
	if err != nil {
		return time.Time{}, err
	}

	return commit.CommittedAt, nil
}

// SyncIncremental 增量同步
func (c *Collector) SyncIncremental() error {
	var projects []model.Project
	if err := c.db.Where("enabled = ?", true).Find(&projects).Error; err != nil {
		return err
	}

	until := time.Now()

	for _, project := range projects {
		since, err := c.GetLastSyncTime(project.ID)
		if err != nil {
			log.Printf("获取项目 %s 最后同步时间失败: %v", project.Name, err)
			continue
		}

		if err := c.SyncProject(project.ID, since, until); err != nil {
			log.Printf("增量同步项目 %s 失败: %v", project.Name, err)
		}

		time.Sleep(2 * time.Second)
	}

	return nil
}

// RefreshProject 刷新项目信息（从GitLab同步）
func (c *Collector) RefreshProject(gitlabID int) (*model.Project, error) {
	glProject, err := c.client.GetProject(gitlabID)
	if err != nil {
		return nil, err
	}

	var project model.Project
	// Unscoped 查询包括软删除的记录
	result := c.db.Unscoped().Where("gitlab_id = ?", gitlabID).First(&project)

	if result.Error == gorm.ErrRecordNotFound {
		project = model.Project{
			GitLabID: glProject.ID,
			Name:     glProject.Name,
			Path:     glProject.PathWithNamespace,
			WebURL:   glProject.WebURL,
			Enabled:  true,
		}
		c.db.Create(&project)
	} else if result.Error == nil {
		// 已存在（可能是软删除的），恢复并更新
		project.Name = glProject.Name
		project.Path = glProject.PathWithNamespace
		project.WebURL = glProject.WebURL
		project.Enabled = true
		project.DeletedAt = gorm.DeletedAt{Valid: false}
		c.db.Unscoped().Save(&project)
	} else {
		return nil, result.Error
	}

	return &project, nil
}

// SyncFullHistory 全量同步所有项目的所有历史提交数据（仅拉取数据，不触发报表和AI分析）
// 最多同步当前时间往前一年的数据
func (c *Collector) SyncFullHistory() error {
	var projects []model.Project
	if err := c.db.Where("enabled = ?", true).Find(&projects).Error; err != nil {
		return err
	}

	// 从一年前开始同步，最多同步一年内的数据
	since := time.Now().AddDate(-1, 0, 0)
	until := time.Now()

	log.Printf("开始全量历史同步，共 %d 个项目，时间范围: %s ~ %s", len(projects), since.Format("2006-01-02"), until.Format("2006-01-02"))

	for i, project := range projects {
		log.Printf("[%d/%d] 全量同步项目: %s", i+1, len(projects), project.Name)
		if err := c.SyncProject(project.ID, since, until); err != nil {
			log.Printf("全量同步项目 %s 失败: %v", project.Name, err)
		}
		// 每个项目之间休眠2秒，避免API限流
		if i < len(projects)-1 {
			time.Sleep(2 * time.Second)
		}
	}

	log.Printf("全量历史同步完成，共处理 %d 个项目", len(projects))
	return nil
}

// enrichGitLabUsers 通过Events API匹配GitLab用户信息到commits表
// 策略：
// 1. 通过 PushData.CommitTo 匹配 push 的最后一个 commit
// 2. 通过 author_email 搜索 GitLab 用户作为 fallback
func (c *Collector) enrichGitLabUsers(gitlabID int, since, until time.Time) error {
	// 扩展时间范围前后各1天作为冗余
	after := since.AddDate(0, 0, -1).Format("2006-01-02")
	before := until.AddDate(0, 0, 1).Format("2006-01-02")

	log.Printf("开始填充GitLab用户信息: project=%d, events时间范围: %s ~ %s", gitlabID, after, before)

	// 拉取所有push events（分页）
	var allEvents []gitlab.Event
	page := 1
	perPage := 100

	for {
		events, err := c.client.ListProjectEvents(gitlabID, after, before, page, perPage)
		if err != nil {
			return fmt.Errorf("拉取Events失败: %w", err)
		}

		allEvents = append(allEvents, events...)

		if len(events) < perPage {
			break
		}
		page++
	}

	log.Printf("获取到 %d 个push events", len(allEvents))

	// 构建 commit_to → Event 的映射（一个 commit_to 对应一个 event）
	commitToEvent := make(map[string]*gitlab.Event)
	for i := range allEvents {
		e := &allEvents[i]
		if e.PushData != nil && e.PushData.CommitTo != "" {
			commitToEvent[e.PushData.CommitTo] = e
		}
	}

	// 查询本项目中 git_lab_name 为空的 commits
	var commits []model.Commit
	c.db.Where("project_id = ? AND git_lab_name = '' AND author_email != ''", gitlabID).
		Order("committed_at DESC").
		Find(&commits)

	if len(commits) == 0 {
		log.Printf("没有需要填充GitLab用户信息的commits")
		return nil
	}

	log.Printf("需要填充GitLab用户信息的commits: %d 个", len(commits))

	// 第一步：通过 commit_to 直接匹配
	updatedByCommitTo := 0
	var unmatchedCommits []model.Commit
	for _, commit := range commits {
		event, ok := commitToEvent[commit.ID]
		if !ok {
			unmatchedCommits = append(unmatchedCommits, commit)
			continue
		}

		c.db.Model(&model.Commit{}).Where("id = ?", commit.ID).
			Updates(map[string]interface{}{
				"git_lab_name":     event.Author.Name,
				"git_lab_username": event.Author.Username,
			})
		updatedByCommitTo++
	}

	log.Printf("第一步(commit_to匹配): 更新 %d 个commits", updatedByCommitTo)

	// 第二步：对未匹配的 commits，通过 author_email 搜索 GitLab 用户填充
	updatedByEmail := 0
	emailUserCache := make(map[string]*gitlab.User) // 缓存已搜索过的邮箱

	for _, commit := range unmatchedCommits {
		if commit.AuthorEmail == "" {
			continue
		}

		// 检查缓存
		if cached, checked := emailUserCache[commit.AuthorEmail]; checked {
			if cached != nil {
				c.db.Model(&model.Commit{}).Where("id = ?", commit.ID).
					Updates(map[string]interface{}{
						"git_lab_name":     cached.Name,
						"git_lab_username": cached.Username,
					})
				updatedByEmail++
			}
			continue
		}

		// 搜索 GitLab 用户
		user, err := c.client.SearchUserByEmail(commit.AuthorEmail)
		if err != nil {
			log.Printf("搜索GitLab用户失败 %s: %v", commit.AuthorEmail, err)
			emailUserCache[commit.AuthorEmail] = nil
			continue
		}

		emailUserCache[commit.AuthorEmail] = user

		if user == nil {
			continue
		}

		c.db.Model(&model.Commit{}).Where("id = ?", commit.ID).
			Updates(map[string]interface{}{
				"git_lab_name":     user.Name,
				"git_lab_username": user.Username,
			})
		updatedByEmail++

		// 避免 API 限流
		time.Sleep(100 * time.Millisecond)
	}

	log.Printf("第二步(email搜索匹配): 更新 %d 个commits", updatedByEmail)
	log.Printf("填充GitLab用户信息完成: 共更新 %d / %d 个commits (commit_to: %d, email: %d)",
		updatedByCommitTo+updatedByEmail, len(commits), updatedByCommitTo, updatedByEmail)
	return nil
}

// enrichCommitsByEvents 根据一批 commits 的时间范围拉取 Events 并匹配 git_lab_name/git_lab_username
// 在每拉取一页 commits 后立即调用，避免时间跨度过大导致 Events 遗漏
func (c *Collector) enrichCommitsByEvents(gitlabID int, commits []gitlab.Commit) {
	if len(commits) == 0 {
		return
	}

	// 计算这页 commits 的时间范围，前后各加1天冗余
	var minTime, maxTime time.Time
	for i, commit := range commits {
		if i == 0 || commit.CommittedDate.Before(minTime) {
			minTime = commit.CommittedDate
		}
		if i == 0 || commit.CommittedDate.After(maxTime) {
			maxTime = commit.CommittedDate
		}
	}
	after := minTime.AddDate(0, 0, -10).Format("2006-01-02")
	before := maxTime.AddDate(0, 0, 10).Format("2006-01-02")

	// 拉取 Events（分页）
	var allEvents []gitlab.Event
	page := 1
	perPage := 100

	for {
		events, err := c.client.ListProjectEvents(gitlabID, after, before, page, perPage)
		if err != nil {
			log.Printf("拉取Events失败 (project=%d, %s~%s): %v", gitlabID, after, before, err)
			return
		}

		allEvents = append(allEvents, events...)

		if len(events) < perPage {
			break
		}
		page++
	}

	// 构建 commit_to → Event 的映射
	commitToEvent := make(map[string]*gitlab.Event)
	for i := range allEvents {
		e := &allEvents[i]
		if e.PushData != nil && e.PushData.CommitTo != "" {
			commitToEvent[e.PushData.CommitTo] = e
		}
	}

	// 匹配这页中的每个 commit
	updatedByCommitTo := 0
	var unmatched []gitlab.Commit

	for _, commit := range commits {
		// 只处理 git_lab_name 为空的 commit
		var count int64
		c.db.Model(&model.Commit{}).Where("id = ? AND git_lab_name = ''", commit.ID).Count(&count)
		if count == 0 {
			continue
		}

		event, ok := commitToEvent[commit.ID]
		if !ok {
			unmatched = append(unmatched, commit)
			continue
		}

		c.db.Model(&model.Commit{}).Where("id = ?", commit.ID).
			Updates(map[string]interface{}{
				"git_lab_name":     event.Author.Name,
				"git_lab_username": event.Author.Username,
			})
		updatedByCommitTo++
	}

	// Fallback: 对未匹配的 commits，通过 author_email 搜索 GitLab 用户
	updatedByEmail := 0
	emailUserCache := make(map[string]*gitlab.User)

	for _, commit := range unmatched {
		if commit.AuthorEmail == "" {
			continue
		}

		if cached, checked := emailUserCache[commit.AuthorEmail]; checked {
			if cached != nil {
				c.db.Model(&model.Commit{}).Where("id = ?", commit.ID).
					Updates(map[string]interface{}{
						"git_lab_name":     cached.Name,
						"git_lab_username": cached.Username,
					})
				updatedByEmail++
			}
			continue
		}

		user, err := c.client.SearchUserByEmail(commit.AuthorEmail)
		if err != nil {
			emailUserCache[commit.AuthorEmail] = nil
			continue
		}

		emailUserCache[commit.AuthorEmail] = user

		if user == nil {
			continue
		}

		c.db.Model(&model.Commit{}).Where("id = ?", commit.ID).
			Updates(map[string]interface{}{
				"git_lab_name":     user.Name,
				"git_lab_username": user.Username,
			})
		updatedByEmail++

		time.Sleep(100 * time.Millisecond)
	}

	log.Printf("Events匹配: 本页%d个commits, commit_to更新%d, email更新%d", len(commits), updatedByCommitTo, updatedByEmail)
}

// SearchGitLabProjects 搜索GitLab项目
func (c *Collector) SearchGitLabProjects(search string) ([]gitlab.Project, error) {
	return c.client.SearchProjects(search)
}
