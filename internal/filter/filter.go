package filter

import (
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"gitlab-push-stat/internal/config"
	"gitlab-push-stat/internal/gitlab"
	"gitlab-push-stat/internal/model"

	"gorm.io/gorm"
)

// Engine 过滤引擎
type Engine struct {
	cfg         *config.FilterConfig
	dbRules     []model.FilterRule
	pathRegexps []*regexp.Regexp
	extMap      map[string]bool
	filenameMap map[string]bool
	msgRegexps  []*regexp.Regexp
	mu          sync.RWMutex
}

// NewEngine 创建过滤引擎
func NewEngine(cfg *config.FilterConfig) *Engine {
	e := &Engine{
		cfg:         cfg,
		extMap:      make(map[string]bool),
		filenameMap: make(map[string]bool),
	}
	e.loadFromConfig()
	return e
}

// loadFromConfig 从配置文件加载规则
func (e *Engine) loadFromConfig() {
	e.mu.Lock()
	defer e.mu.Unlock()

	// 加载路径正则
	for _, pattern := range e.cfg.ExcludePaths {
		if re, err := regexp.Compile(pattern); err == nil {
			e.pathRegexps = append(e.pathRegexps, re)
		}
	}

	// 加载扩展名
	for _, ext := range e.cfg.ExcludeExtensions {
		e.extMap[strings.ToLower(ext)] = true
	}

	// 加载文件名
	for _, name := range e.cfg.ExcludeFilenames {
		e.filenameMap[strings.ToLower(name)] = true
	}

	// 加载提交信息正则
	for _, pattern := range e.cfg.ExcludeMessages {
		if re, err := regexp.Compile(pattern); err == nil {
			e.msgRegexps = append(e.msgRegexps, re)
		}
	}
}

// LoadFromDB 从数据库加载规则
func (e *Engine) LoadFromDB(db *gorm.DB) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	var rules []model.FilterRule
	if err := db.Where("enabled = ? AND deleted_at IS NULL", true).Find(&rules).Error; err != nil {
		return err
	}

	e.dbRules = rules

	// 合并数据库规则
	for _, rule := range rules {
		switch rule.RuleType {
		case "path":
			if re, err := regexp.Compile(rule.Pattern); err == nil {
				e.pathRegexps = append(e.pathRegexps, re)
			}
		case "extension":
			e.extMap[strings.ToLower(rule.Pattern)] = true
		case "filename":
			e.filenameMap[strings.ToLower(rule.Pattern)] = true
		case "message":
			if re, err := regexp.Compile(rule.Pattern); err == nil {
				e.msgRegexps = append(e.msgRegexps, re)
			}
		}
	}

	return nil
}

// ShouldFilterCommit 判断是否应该过滤整个提交
func (e *Engine) ShouldFilterCommit(message string) bool {
	e.mu.RLock()
	defer e.mu.RUnlock()

	for _, re := range e.msgRegexps {
		if re.MatchString(message) {
			return true
		}
	}
	return false
}

// ShouldFilterFile 判断是否应该过滤某个文件
func (e *Engine) ShouldFilterFile(filePath string) bool {
	e.mu.RLock()
	defer e.mu.RUnlock()

	normalizedPath := strings.ReplaceAll(filePath, "\\", "/")
	lowerPath := strings.ToLower(normalizedPath)
	fileName := strings.ToLower(filepath.Base(normalizedPath))

	// 检查文件名
	if e.filenameMap[fileName] {
		return true
	}

	// 检查扩展名
	for ext := range e.extMap {
		if strings.HasSuffix(lowerPath, ext) {
			return true
		}
	}

	// 检查路径正则
	for _, re := range e.pathRegexps {
		if re.MatchString(normalizedPath) {
			return true
		}
	}

	return false
}

// FilterDiffs 过滤diff列表，返回有效的新增/删除行数
func (e *Engine) FilterDiffs(diffs []gitlab.Diff) (filteredAdditions, filteredDeletions int) {
	for _, diff := range diffs {
		filePath := diff.NewPath
		if filePath == "" {
			filePath = diff.OldPath
		}

		if e.ShouldFilterFile(filePath) {
			continue
		}

		// 解析diff内容统计行数
		adds, dels := parseDiffStats(diff.Diff)
		filteredAdditions += adds
		filteredDeletions += dels
	}
	return
}

// parseDiffStats 解析diff内容统计新增/删除行数
func parseDiffStats(diffContent string) (additions, deletions int) {
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

// GetFilterStats 获取过滤统计信息
type FilterStats struct {
	TotalFiles     int `json:"total_files"`
	FilteredFiles  int `json:"filtered_files"`
	EffectiveFiles int `json:"effective_files"`
}

// AnalyzeDiffs 分析diff列表，返回过滤统计
func (e *Engine) AnalyzeDiffs(diffs []gitlab.Diff) (FilterStats, int, int) {
	stats := FilterStats{TotalFiles: len(diffs)}
	var filteredAdds, filteredDels int

	for _, diff := range diffs {
		filePath := diff.NewPath
		if filePath == "" {
			filePath = diff.OldPath
		}

		if e.ShouldFilterFile(filePath) {
			stats.FilteredFiles++
		} else {
			stats.EffectiveFiles++
			adds, dels := parseDiffStats(diff.Diff)
			filteredAdds += adds
			filteredDels += dels
		}
	}

	return stats, filteredAdds, filteredDels
}

// GetConfig 返回配置文件中的过滤规则配置
func (e *Engine) GetConfig() *config.FilterConfig {
	return e.cfg
}

// Reload 重新加载过滤规则
func (e *Engine) Reload(cfg *config.FilterConfig, db *gorm.DB) error {
	e.mu.Lock()
	// 清空旧规则
	e.pathRegexps = nil
	e.extMap = make(map[string]bool)
	e.filenameMap = make(map[string]bool)
	e.msgRegexps = nil
	e.dbRules = nil
	e.cfg = cfg
	e.mu.Unlock()

	e.loadFromConfig()
	return e.LoadFromDB(db)
}
