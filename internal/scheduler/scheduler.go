package scheduler

import (
	"log"
	"sync"
	"time"

	"gitlab-push-stat/internal/ai"
	"gitlab-push-stat/internal/collector"
	"gitlab-push-stat/internal/config"
	"gitlab-push-stat/internal/filter"
	"gitlab-push-stat/internal/report"

	"github.com/robfig/cron/v3"
)

// Scheduler 定时任务调度器
type Scheduler struct {
	cron         *cron.Cron
	collector    *collector.Collector
	report       *report.Service
	aiReviewer   *ai.Reviewer
	cfg          *config.Config
	filterEngine *filter.Engine
	mu           sync.Mutex
	running      bool
}

// NewScheduler 创建调度器
func NewScheduler(cfg *config.Config, coll *collector.Collector, reportSvc *report.Service, filterEngine *filter.Engine) *Scheduler {
	c := cron.New()

	var aiReviewer *ai.Reviewer
	if cfg.AI.Enabled {
		aiReviewer = ai.NewReviewer(cfg, filterEngine)
		log.Println("AI审核功能已启用")
	}

	return &Scheduler{
		cron:       c,
		collector:  coll,
		report:     reportSvc,
		aiReviewer: aiReviewer,
		cfg:        cfg,
	}
}

// Start 启动调度器
func (s *Scheduler) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return nil
	}

	// 数据同步任务（同步完成后自动生成当天日报表）
	if _, err := s.cron.AddFunc(s.cfg.Schedule.Sync, func() {
		log.Println("定时任务: 开始增量同步数据...")
		if err := s.collector.SyncIncremental(); err != nil {
			log.Printf("增量同步失败: %v", err)
		}
		log.Println("定时任务: 增量同步完成")

		// 同步完成后自动生成当天日报表（覆盖当天数据）
		log.Println("定时任务: 自动生成当天日报表...")
		if err := s.report.GenerateTodayReport(); err != nil {
			log.Printf("自动生成日报表失败: %v", err)
		}
		log.Println("定时任务: 当天日报表生成完成")
	}); err != nil {
		log.Printf("注册同步任务失败: %v", err)
	}

	// 每日报表任务 + AI审核
	if _, err := s.cron.AddFunc(s.cfg.Schedule.Daily, func() {
		log.Println("定时任务: 开始生成日报表...")
		yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")

		if err := s.report.GenerateYesterdayReport(); err != nil {
			log.Printf("生成日报表失败: %v", err)
		}
		log.Println("定时任务: 日报表生成完成")

		// 日报表生成后触发AI审核
		if s.aiReviewer != nil {
			log.Println("定时任务: 开始AI代码审核...")
			if err := s.aiReviewer.ReviewDailyCommits(yesterday); err != nil {
				log.Printf("AI审核失败: %v", err)
			}
			log.Println("定时任务: AI代码审核完成")
		}
	}); err != nil {
		log.Printf("注册日报任务失败: %v", err)
	}

	// 每周报表任务
	if _, err := s.cron.AddFunc(s.cfg.Schedule.Weekly, func() {
		log.Println("定时任务: 开始生成周报表...")
		if err := s.report.GenerateLastWeekReport(); err != nil {
			log.Printf("生成周报表失败: %v", err)
		}
		log.Println("定时任务: 周报表生成完成")
	}); err != nil {
		log.Printf("注册周报任务失败: %v", err)
	}

	// 每月报表任务
	if _, err := s.cron.AddFunc(s.cfg.Schedule.Monthly, func() {
		log.Println("定时任务: 开始生成月报表...")
		if err := s.report.GenerateLastMonthReport(); err != nil {
			log.Printf("生成月报表失败: %v", err)
		}
		log.Println("定时任务: 月报表生成完成")
	}); err != nil {
		log.Printf("注册月报任务失败: %v", err)
	}

	s.cron.Start()
	s.running = true
	log.Println("定时任务调度器已启动")

	log.Printf("调度配置: 同步=%s, 日报=%s, 周报=%s, 月报=%s",
		s.cfg.Schedule.Sync, s.cfg.Schedule.Daily, s.cfg.Schedule.Weekly, s.cfg.Schedule.Monthly)

	return nil
}

// Stop 停止调度器
func (s *Scheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return
	}

	ctx := s.cron.Stop()
	<-ctx.Done()
	s.running = false
	log.Println("定时任务调度器已停止")
}

// IsRunning 是否正在运行
func (s *Scheduler) IsRunning() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.running
}

// GetEntries 获取所有任务条目
func (s *Scheduler) GetEntries() []cron.Entry {
	return s.cron.Entries()
}

// RunSyncNow 立即执行同步，完成后自动生成当天日报表
func (s *Scheduler) RunSyncNow() error {
	log.Println("手动触发: 开始增量同步数据...")
	if err := s.collector.SyncIncremental(); err != nil {
		return err
	}
	log.Println("手动触发: 同步完成，自动生成当天日报表...")
	if err := s.report.GenerateTodayReport(); err != nil {
		log.Printf("自动生成日报表失败: %v", err)
		return err
	}
	log.Println("手动触发: 当天日报表生成完成")
	return nil
}

// RunDailyReportNow 立即生成日报表
func (s *Scheduler) RunDailyReportNow() error {
	log.Println("手动触发: 开始生成日报表...")
	yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")

	if err := s.report.GenerateYesterdayReport(); err != nil {
		return err
	}

	// 手动触发也执行AI审核
	if s.aiReviewer != nil {
		log.Println("手动触发: 开始AI代码审核...")
		if err := s.aiReviewer.ReviewDailyCommits(yesterday); err != nil {
			log.Printf("AI审核失败: %v", err)
		}
	}

	return nil
}

// RunAIReviewNow 立即执行AI审核
func (s *Scheduler) RunAIReviewNow(date string) error {
	if s.aiReviewer == nil {
		return nil
	}
	log.Printf("手动触发AI审核: %s", date)
	return s.aiReviewer.ReviewDailyCommits(date)
}

// GetAIReviewer 获取AI审核器（供handler使用）
func (s *Scheduler) GetAIReviewer() *ai.Reviewer {
	return s.aiReviewer
}

// RunWeeklyReportNow 立即生成周报表
func (s *Scheduler) RunWeeklyReportNow() error {
	log.Println("手动触发: 开始生成周报表...")
	return s.report.GenerateLastWeekReport()
}

// RunFullHistorySync 全量同步所有历史数据（仅拉取数据，不触发报表生成和AI分析）
func (s *Scheduler) RunFullHistorySync() error {
	log.Println("手动触发: 开始全量历史数据同步...")
	if err := s.collector.SyncFullHistory(); err != nil {
		log.Printf("全量历史同步失败: %v", err)
		return err
	}
	log.Println("手动触发: 全量历史数据同步完成")
	return nil
}

// RunMonthlyReportNow 立即生成月报表
func (s *Scheduler) RunMonthlyReportNow() error {
	log.Println("手动触发: 开始生成月报表...")
	return s.report.GenerateLastMonthReport()
}
