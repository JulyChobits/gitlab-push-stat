package report

import (
	"fmt"
	"log"
	"time"

	"gitlab-push-stat/internal/database"
	"gitlab-push-stat/internal/model"

	"gorm.io/gorm"
)

// Service 报表服务
type Service struct {
	db *gorm.DB
}

// NewService 创建报表服务
func NewService() *Service {
	return &Service{db: database.GetDB()}
}

// getDisplayName 获取显示名称：优先git_lab_name，fallback到author_name
func getDisplayName(gitlabName, authorName string) string {
	if gitlabName != "" {
		return gitlabName
	}
	if authorName != "" {
		return authorName
	}
	return "未知用户"
}

// GenerateDailyReport 生成日报表（固化数据）
func (s *Service) GenerateDailyReport(reportDate string) error {
	log.Printf("生成日报表: %s", reportDate)

	localDate, err := time.ParseInLocation("2006-01-02", reportDate, time.Local)
	if err != nil {
		return fmt.Errorf("日期格式错误: %w", err)
	}
	dayStart := localDate.UTC()
	dayEnd := localDate.AddDate(0, 0, 1).UTC()
	dayStartStr := dayStart.Format("2006-01-02 15:04:05+00:00")
	dayEndStr := dayEnd.Format("2006-01-02 15:04:05+00:00")
	log.Printf("日报表时间范围(UTC): %s ~ %s", dayStartStr, dayEndStr)

	dayStartTS := dayStart.Unix()
	dayEndTS := dayEnd.Unix()

	type Row struct {
		ProjectID         uint
		AuthorEmail       string
		AuthorName        string
		CommitCount       int64
		Additions         int64
		Deletions         int64
		FilteredAdditions int64
		FilteredDeletions int64
	}

	var rows []Row
	err = s.db.Raw(`
		SELECT 
			project_id,
			COALESCE(author_email, '') as author_email,
			COALESCE(NULLIF(git_lab_name, ''), MAX(author_name)) as author_name,
			COUNT(*) as commit_count,
			SUM(additions) as additions,
			SUM(deletions) as deletions,
			SUM(filtered_additions) as filtered_additions,
			SUM(filtered_deletions) as filtered_deletions
		FROM commits 
		WHERE strftime('%s', committed_at) >= ? AND strftime('%s', committed_at) < ? AND (is_filtered = 0 OR is_filtered = 'false')
		GROUP BY project_id, COALESCE(NULLIF(git_lab_name, ''), author_name)
	`, fmt.Sprintf("%d", dayStartTS), fmt.Sprintf("%d", dayEndTS)).Scan(&rows).Error

	if err != nil {
		return fmt.Errorf("查询提交数据失败: %w", err)
	}

	log.Printf("日报表查询到 %d 条聚合记录", len(rows))

	s.db.Where("report_date = ?", reportDate).Delete(&model.DailyReport{})

	for _, row := range rows {
		report := model.DailyReport{
			ProjectID:      row.ProjectID,
			AuthorEmail:    row.AuthorEmail,
			AuthorName:     row.AuthorName,
			ReportDate:     reportDate,
			CommitCount:    int(row.CommitCount),
			Additions:      int(row.Additions),
			Deletions:      int(row.Deletions),
			NetAdditions:   int(row.Additions) - int(row.Deletions),
			EffectiveLines: int(row.FilteredAdditions) - int(row.FilteredDeletions),
		}
		s.db.Create(&report)
	}

	log.Printf("日报表生成完成: %s, 共 %d 条记录", reportDate, len(rows))
	return nil
}

// GenerateWeeklyReport 生成周报表
func (s *Service) GenerateWeeklyReport(weekStart, weekEnd string) error {
	log.Printf("生成周报表: %s ~ %s", weekStart, weekEnd)

	type Row struct {
		ProjectID         uint
		AuthorEmail       string
		AuthorName        string
		CommitCount       int64
		Additions         int64
		Deletions         int64
		FilteredAdditions int64
		FilteredDeletions int64
	}

	localStart, err := time.ParseInLocation("2006-01-02", weekStart, time.Local)
	if err != nil {
		return fmt.Errorf("周起始日期格式错误: %w", err)
	}
	localEnd, err := time.ParseInLocation("2006-01-02", weekEnd, time.Local)
	if err != nil {
		return fmt.Errorf("周结束日期格式错误: %w", err)
	}
	weekStartTS := localStart.UTC().Unix()
	weekEndTS := localEnd.AddDate(0, 0, 1).UTC().Unix()

	var rows []Row
	err = s.db.Raw(`
		SELECT 
			project_id,
			COALESCE(author_email, '') as author_email,
			COALESCE(NULLIF(git_lab_name, ''), MAX(author_name)) as author_name,
			COUNT(*) as commit_count,
			SUM(additions) as additions,
			SUM(deletions) as deletions,
			SUM(filtered_additions) as filtered_additions,
			SUM(filtered_deletions) as filtered_deletions
		FROM commits
		WHERE strftime('%s', committed_at) >= ? AND strftime('%s', committed_at) < ? AND (is_filtered = 0 OR is_filtered = 'false')
		GROUP BY project_id, COALESCE(NULLIF(git_lab_name, ''), author_name)
	`, fmt.Sprintf("%d", weekStartTS), fmt.Sprintf("%d", weekEndTS)).Scan(&rows).Error

	if err != nil {
		return fmt.Errorf("查询提交数据失败: %w", err)
	}

	for _, row := range rows {
		report := model.WeeklyReport{
			ProjectID:      row.ProjectID,
			AuthorEmail:    row.AuthorEmail,
			AuthorName:     row.AuthorName,
			WeekStart:      weekStart,
			WeekEnd:        weekEnd,
			CommitCount:    int(row.CommitCount),
			Additions:      int(row.Additions),
			Deletions:      int(row.Deletions),
			NetAdditions:   int(row.Additions) - int(row.Deletions),
			EffectiveLines: int(row.FilteredAdditions) - int(row.FilteredDeletions),
		}

		result := s.db.Where("project_id = ? AND author_name = ? AND week_start = ?",
			report.ProjectID, report.AuthorName, report.WeekStart).First(&model.WeeklyReport{})

		if result.Error == gorm.ErrRecordNotFound {
			s.db.Create(&report)
		} else {
			s.db.Model(&model.WeeklyReport{}).
				Where("project_id = ? AND author_name = ? AND week_start = ?",
					report.ProjectID, report.AuthorName, report.WeekStart).
				Updates(map[string]interface{}{
					"author_email":    report.AuthorEmail,
					"commit_count":    report.CommitCount,
					"additions":       report.Additions,
					"deletions":       report.Deletions,
					"net_additions":   report.NetAdditions,
					"effective_lines": report.EffectiveLines,
				})
		}
	}

	log.Printf("周报表生成完成: %s ~ %s, 共 %d 条记录", weekStart, weekEnd, len(rows))
	return nil
}

// GenerateMonthlyReport 生成月报表
func (s *Service) GenerateMonthlyReport(yearMonth string) error {
	log.Printf("生成月报表: %s", yearMonth)

	localMonth, err := time.ParseInLocation("2006-01", yearMonth, time.Local)
	if err != nil {
		return fmt.Errorf("月份格式错误: %w", err)
	}
	monthStartTS := localMonth.UTC().Unix()
	monthEndTS := localMonth.AddDate(0, 1, 0).UTC().Unix()

	type Row struct {
		ProjectID         uint
		AuthorEmail       string
		AuthorName        string
		CommitCount       int64
		Additions         int64
		Deletions         int64
		FilteredAdditions int64
		FilteredDeletions int64
	}

	var rows []Row
	err = s.db.Raw(`
		SELECT 
			project_id,
			COALESCE(author_email, '') as author_email,
			COALESCE(NULLIF(git_lab_name, ''), MAX(author_name)) as author_name,
			COUNT(*) as commit_count,
			SUM(additions) as additions,
			SUM(deletions) as deletions,
			SUM(filtered_additions) as filtered_additions,
			SUM(filtered_deletions) as filtered_deletions
		FROM commits
		WHERE strftime('%s', committed_at) >= ? AND strftime('%s', committed_at) < ? AND (is_filtered = 0 OR is_filtered = 'false')
		GROUP BY project_id, COALESCE(NULLIF(git_lab_name, ''), author_name)
	`, fmt.Sprintf("%d", monthStartTS), fmt.Sprintf("%d", monthEndTS)).Scan(&rows).Error

	if err != nil {
		return fmt.Errorf("查询提交数据失败: %w", err)
	}

	for _, row := range rows {
		report := model.MonthlyReport{
			ProjectID:      row.ProjectID,
			AuthorEmail:    row.AuthorEmail,
			AuthorName:     row.AuthorName,
			YearMonth:      yearMonth,
			CommitCount:    int(row.CommitCount),
			Additions:      int(row.Additions),
			Deletions:      int(row.Deletions),
			NetAdditions:   int(row.Additions) - int(row.Deletions),
			EffectiveLines: int(row.FilteredAdditions) - int(row.FilteredDeletions),
		}

		result := s.db.Where("project_id = ? AND author_name = ? AND year_month = ?",
			report.ProjectID, report.AuthorName, report.YearMonth).First(&model.MonthlyReport{})

		if result.Error == gorm.ErrRecordNotFound {
			s.db.Create(&report)
		} else {
			s.db.Model(&model.MonthlyReport{}).
				Where("project_id = ? AND author_name = ? AND year_month = ?",
					report.ProjectID, report.AuthorName, report.YearMonth).
				Updates(map[string]interface{}{
					"author_email":    report.AuthorEmail,
					"commit_count":    report.CommitCount,
					"additions":       report.Additions,
					"deletions":       report.Deletions,
					"net_additions":   report.NetAdditions,
					"effective_lines": report.EffectiveLines,
				})
		}
	}

	log.Printf("月报表生成完成: %s, 共 %d 条记录", yearMonth, len(rows))
	return nil
}

// GenerateYesterdayReport 生成昨天的日报表
func (s *Service) GenerateYesterdayReport() error {
	yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	return s.GenerateDailyReport(yesterday)
}

// GenerateTodayReport 生成今天的日报表
func (s *Service) GenerateTodayReport() error {
	today := time.Now().Format("2006-01-02")
	return s.GenerateDailyReport(today)
}

// GenerateLastWeekReport 生成上周的周报表
func (s *Service) GenerateLastWeekReport() error {
	now := time.Now()
	weekday := int(now.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	lastMonday := now.AddDate(0, 0, -(weekday + 6))
	lastSunday := now.AddDate(0, 0, -(weekday))

	return s.GenerateWeeklyReport(
		lastMonday.Format("2006-01-02"),
		lastSunday.Format("2006-01-02"),
	)
}

// GenerateLastMonthReport 生成上月的月报表
func (s *Service) GenerateLastMonthReport() error {
	now := time.Now()
	lastMonth := time.Date(now.Year(), now.Month()-1, 1, 0, 0, 0, 0, now.Location())
	return s.GenerateMonthlyReport(lastMonth.Format("2006-01"))
}

// DailyReportQuery 日报表查询参数
type DailyReportQuery struct {
	StartDate  string `form:"start_date"`
	EndDate    string `form:"end_date"`
	ProjectID  uint   `form:"project_id"`
	AuthorName string `form:"author_name"`
	Page       int    `form:"page,default=1"`
	PageSize   int    `form:"page_size,default=20"`
}

// QueryDailyReports 查询日报表
func (s *Service) QueryDailyReports(q DailyReportQuery) ([]model.DailyReport, int64, error) {
	var reports []model.DailyReport
	var total int64

	query := s.db.Model(&model.DailyReport{}).Preload("Project")

	if q.StartDate != "" {
		query = query.Where("report_date >= ?", q.StartDate)
	}
	if q.EndDate != "" {
		query = query.Where("report_date <= ?", q.EndDate)
	}
	if q.ProjectID > 0 {
		query = query.Where("project_id = ?", q.ProjectID)
	}
	if q.AuthorName != "" {
		query = query.Where("author_name LIKE ?", "%"+q.AuthorName+"%")
	}

	query.Count(&total)

	offset := (q.Page - 1) * q.PageSize
	err := query.Order("report_date DESC, commit_count DESC").
		Offset(offset).Limit(q.PageSize).
		Find(&reports).Error

	return reports, total, err
}

// WeeklyReportQuery 周报表查询参数
type WeeklyReportQuery struct {
	WeekStart  string `form:"week_start"`
	WeekEnd    string `form:"week_end"`
	ProjectID  uint   `form:"project_id"`
	AuthorName string `form:"author_name"`
	Page       int    `form:"page,default=1"`
	PageSize   int    `form:"page_size,default=20"`
}

// QueryWeeklyReports 查询周报表
func (s *Service) QueryWeeklyReports(q WeeklyReportQuery) ([]model.WeeklyReport, int64, error) {
	var reports []model.WeeklyReport
	var total int64

	query := s.db.Model(&model.WeeklyReport{}).Preload("Project")

	if q.WeekStart != "" {
		query = query.Where("week_start >= ?", q.WeekStart)
	}
	if q.WeekEnd != "" {
		query = query.Where("week_end <= ?", q.WeekEnd)
	}
	if q.ProjectID > 0 {
		query = query.Where("project_id = ?", q.ProjectID)
	}
	if q.AuthorName != "" {
		query = query.Where("author_name LIKE ?", "%"+q.AuthorName+"%")
	}

	query.Count(&total)

	offset := (q.Page - 1) * q.PageSize
	err := query.Order("week_start DESC, commit_count DESC").
		Offset(offset).Limit(q.PageSize).
		Find(&reports).Error

	return reports, total, err
}

// MonthlyReportQuery 月报表查询参数
type MonthlyReportQuery struct {
	YearMonth  string `form:"year_month"`
	ProjectID  uint   `form:"project_id"`
	AuthorName string `form:"author_name"`
	Page       int    `form:"page,default=1"`
	PageSize   int    `form:"page_size,default=20"`
}

// QueryMonthlyReports 查询月报表
func (s *Service) QueryMonthlyReports(q MonthlyReportQuery) ([]model.MonthlyReport, int64, error) {
	var reports []model.MonthlyReport
	var total int64

	query := s.db.Model(&model.MonthlyReport{}).Preload("Project")

	if q.YearMonth != "" {
		query = query.Where("year_month = ?", q.YearMonth)
	}
	if q.ProjectID > 0 {
		query = query.Where("project_id = ?", q.ProjectID)
	}
	if q.AuthorName != "" {
		query = query.Where("author_name LIKE ?", "%"+q.AuthorName+"%")
	}

	query.Count(&total)

	offset := (q.Page - 1) * q.PageSize
	err := query.Order("year_month DESC, commit_count DESC").
		Offset(offset).Limit(q.PageSize).
		Find(&reports).Error

	return reports, total, err
}

// AuthorSummary 按人汇总结构体
type AuthorSummary struct {
	ReportDate     string `json:"report_date"`
	WeekStart      string `json:"week_start"`
	WeekEnd        string `json:"week_end"`
	YearMonth      string `json:"year_month"`
	AuthorName     string `json:"author_name"`
	CommitCount    int    `json:"commit_count"`
	Additions      int    `json:"additions"`
	Deletions      int    `json:"deletions"`
	NetAdditions   int    `json:"net_additions"`
	EffectiveLines int    `json:"effective_lines"`
}

// RawDailySummary 日报原始汇总（按名称聚合）
type RawDailySummary struct {
	ReportDate     string `json:"report_date"`
	AuthorName     string
	CommitCount    int `json:"commit_count"`
	Additions      int `json:"additions"`
	Deletions      int `json:"deletions"`
	NetAdditions   int `json:"net_additions"`
	EffectiveLines int `json:"effective_lines"`
}

// RawWeeklySummary 周报原始汇总（按名称聚合）
type RawWeeklySummary struct {
	WeekStart      string `json:"week_start"`
	WeekEnd        string `json:"week_end"`
	AuthorName     string
	CommitCount    int `json:"commit_count"`
	Additions      int `json:"additions"`
	Deletions      int `json:"deletions"`
	NetAdditions   int `json:"net_additions"`
	EffectiveLines int `json:"effective_lines"`
}

// RawMonthlySummary 月报原始汇总（按名称聚合）
type RawMonthlySummary struct {
	YearMonth      string `json:"year_month"`
	AuthorName     string
	CommitCount    int `json:"commit_count"`
	Additions      int `json:"additions"`
	Deletions      int `json:"deletions"`
	NetAdditions   int `json:"net_additions"`
	EffectiveLines int `json:"effective_lines"`
}

// QueryDailyAuthorSummary 日报按人汇总
func (s *Service) QueryDailyAuthorSummary(q DailyReportQuery) ([]AuthorSummary, int64, error) {
	var rawResults []RawDailySummary
	var total int64

	query := s.db.Model(&model.DailyReport{}).Select(
		"report_date, author_name, SUM(commit_count) as commit_count, SUM(additions) as additions, SUM(deletions) as deletions, SUM(net_additions) as net_additions, SUM(effective_lines) as effective_lines",
	).Group("report_date, author_name")

	if q.StartDate != "" {
		query = query.Where("report_date >= ?", q.StartDate)
	}
	if q.EndDate != "" {
		query = query.Where("report_date <= ?", q.EndDate)
	}
	if q.ProjectID > 0 {
		query = query.Where("project_id = ?", q.ProjectID)
	}
	if q.AuthorName != "" {
		query = query.Where("author_name LIKE ?", "%"+q.AuthorName+"%")
	}

	countQuery := s.db.Model(&model.DailyReport{}).Select("COUNT(DISTINCT report_date || author_name)")
	if q.StartDate != "" {
		countQuery = countQuery.Where("report_date >= ?", q.StartDate)
	}
	if q.EndDate != "" {
		countQuery = countQuery.Where("report_date <= ?", q.EndDate)
	}
	if q.ProjectID > 0 {
		countQuery = countQuery.Where("project_id = ?", q.ProjectID)
	}
	if q.AuthorName != "" {
		countQuery = countQuery.Where("author_name LIKE ?", "%"+q.AuthorName+"%")
	}
	countQuery.Count(&total)

	offset := (q.Page - 1) * q.PageSize
	err := query.Order("report_date DESC, SUM(commit_count) DESC").
		Offset(offset).Limit(q.PageSize).
		Find(&rawResults).Error

	var results []AuthorSummary
	for _, r := range rawResults {
		results = append(results, AuthorSummary{
			ReportDate:     r.ReportDate,
			AuthorName:     r.AuthorName,
			CommitCount:    r.CommitCount,
			Additions:      r.Additions,
			Deletions:      r.Deletions,
			NetAdditions:   r.NetAdditions,
			EffectiveLines: r.EffectiveLines,
		})
	}

	return results, total, err
}

// QueryWeeklyAuthorSummary 周报按人汇总
func (s *Service) QueryWeeklyAuthorSummary(q WeeklyReportQuery) ([]AuthorSummary, int64, error) {
	var rawResults []RawWeeklySummary
	var total int64

	query := s.db.Model(&model.WeeklyReport{}).Select(
		"week_start, week_end, author_name, SUM(commit_count) as commit_count, SUM(additions) as additions, SUM(deletions) as deletions, SUM(net_additions) as net_additions, SUM(effective_lines) as effective_lines",
	).Group("week_start, week_end, author_name")

	if q.WeekStart != "" {
		query = query.Where("week_start >= ?", q.WeekStart)
	}
	if q.WeekEnd != "" {
		query = query.Where("week_end <= ?", q.WeekEnd)
	}
	if q.ProjectID > 0 {
		query = query.Where("project_id = ?", q.ProjectID)
	}
	if q.AuthorName != "" {
		query = query.Where("author_name LIKE ?", "%"+q.AuthorName+"%")
	}

	countQuery := s.db.Model(&model.WeeklyReport{}).Select("COUNT(DISTINCT week_start || author_name)")
	if q.WeekStart != "" {
		countQuery = countQuery.Where("week_start >= ?", q.WeekStart)
	}
	if q.WeekEnd != "" {
		countQuery = countQuery.Where("week_end <= ?", q.WeekEnd)
	}
	if q.ProjectID > 0 {
		countQuery = countQuery.Where("project_id = ?", q.ProjectID)
	}
	if q.AuthorName != "" {
		countQuery = countQuery.Where("author_name LIKE ?", "%"+q.AuthorName+"%")
	}
	countQuery.Count(&total)

	offset := (q.Page - 1) * q.PageSize
	err := query.Order("week_start DESC, SUM(commit_count) DESC").
		Offset(offset).Limit(q.PageSize).
		Find(&rawResults).Error

	var results []AuthorSummary
	for _, r := range rawResults {
		results = append(results, AuthorSummary{
			WeekStart:      r.WeekStart,
			WeekEnd:        r.WeekEnd,
			AuthorName:     r.AuthorName,
			CommitCount:    r.CommitCount,
			Additions:      r.Additions,
			Deletions:      r.Deletions,
			NetAdditions:   r.NetAdditions,
			EffectiveLines: r.EffectiveLines,
		})
	}

	return results, total, err
}

// AnnualHeatmapResult 年度热力图结果
type AnnualHeatmapResult struct {
	Year    int                   `json:"year"`
	Authors []AnnualHeatmapAuthor `json:"authors"`
}

// AnnualHeatmapAuthor 年度热力图-人员
type AnnualHeatmapAuthor struct {
	AuthorName string                      `json:"author_name"`
	Days       map[string]AnnualHeatmapDay `json:"days"`
}

// AnnualHeatmapDay 年度热力图-每日数据
type AnnualHeatmapDay struct {
	CommitCount int `json:"commit_count"`
	Additions   int `json:"additions"`
	Deletions   int `json:"deletions"`
}

// QueryAnnualHeatmap 查询年度热力图数据
func (s *Service) QueryAnnualHeatmap(year int) (*AnnualHeatmapResult, error) {
	startDate := fmt.Sprintf("%d-01-01", year)
	endDate := fmt.Sprintf("%d-12-31", year)

	type Row struct {
		AuthorName  string
		ReportDate  string
		CommitCount int
		Additions   int
		Deletions   int
	}

	var rows []Row
	err := s.db.Model(&model.Commit{}).
		Select("COALESCE(NULLIF(git_lab_name, ''), author_name) as author_name, strftime('%Y-%m-%d', committed_at) as report_date, COUNT(*) as commit_count, SUM(additions) as additions, SUM(deletions) as deletions").
		Where("strftime('%Y-%m-%d', committed_at) >= ? AND strftime('%Y-%m-%d', committed_at) <= ? AND (is_filtered = 0 OR is_filtered = 'false')", startDate, endDate).
		Group("COALESCE(NULLIF(git_lab_name, ''), author_name), strftime('%Y-%m-%d', committed_at)").
		Order("author_name, report_date").
		Find(&rows).Error

	if err != nil {
		return nil, fmt.Errorf("查询年度热力图失败: %w", err)
	}

	authorMap := make(map[string]map[string]AnnualHeatmapDay)
	for _, row := range rows {
		name := row.AuthorName
		if name == "" {
			name = "未知用户"
		}
		if _, ok := authorMap[name]; !ok {
			authorMap[name] = make(map[string]AnnualHeatmapDay)
		}
		authorMap[name][row.ReportDate] = AnnualHeatmapDay{
			CommitCount: row.CommitCount,
			Additions:   row.Additions,
			Deletions:   row.Deletions,
		}
	}

	var authors []AnnualHeatmapAuthor
	for name, days := range authorMap {
		authors = append(authors, AnnualHeatmapAuthor{
			AuthorName: name,
			Days:       days,
		})
	}

	for i := 0; i < len(authors); i++ {
		for j := i + 1; j < len(authors); j++ {
			if authors[i].AuthorName > authors[j].AuthorName {
				authors[i], authors[j] = authors[j], authors[i]
			}
		}
	}

	return &AnnualHeatmapResult{
		Year:    year,
		Authors: authors,
	}, nil
}

// QueryMonthlyAuthorSummary 月报按人汇总
func (s *Service) QueryMonthlyAuthorSummary(q MonthlyReportQuery) ([]AuthorSummary, int64, error) {
	var rawResults []RawMonthlySummary
	var total int64

	query := s.db.Model(&model.MonthlyReport{}).Select(
		"year_month, author_name, SUM(commit_count) as commit_count, SUM(additions) as additions, SUM(deletions) as deletions, SUM(net_additions) as net_additions, SUM(effective_lines) as effective_lines",
	).Group("year_month, author_name")

	if q.YearMonth != "" {
		query = query.Where("year_month = ?", q.YearMonth)
	}
	if q.ProjectID > 0 {
		query = query.Where("project_id = ?", q.ProjectID)
	}
	if q.AuthorName != "" {
		query = query.Where("author_name LIKE ?", "%"+q.AuthorName+"%")
	}

	countQuery := s.db.Model(&model.MonthlyReport{}).Select("COUNT(DISTINCT year_month || author_name)")
	if q.YearMonth != "" {
		countQuery = countQuery.Where("year_month = ?", q.YearMonth)
	}
	if q.ProjectID > 0 {
		countQuery = countQuery.Where("project_id = ?", q.ProjectID)
	}
	if q.AuthorName != "" {
		countQuery = countQuery.Where("author_name LIKE ?", "%"+q.AuthorName+"%")
	}
	countQuery.Count(&total)

	offset := (q.Page - 1) * q.PageSize
	err := query.Order("year_month DESC, SUM(commit_count) DESC").
		Offset(offset).Limit(q.PageSize).
		Find(&rawResults).Error

	var results []AuthorSummary
	for _, r := range rawResults {
		results = append(results, AuthorSummary{
			YearMonth:      r.YearMonth,
			AuthorName:     r.AuthorName,
			CommitCount:    r.CommitCount,
			Additions:      r.Additions,
			Deletions:      r.Deletions,
			NetAdditions:   r.NetAdditions,
			EffectiveLines: r.EffectiveLines,
		})
	}

	return results, total, err
}
