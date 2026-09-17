package main

import (
	"embed"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"gitlab-push-stat/internal/collector"
	"gitlab-push-stat/internal/config"
	"gitlab-push-stat/internal/database"
	"gitlab-push-stat/internal/filter"
	"gitlab-push-stat/internal/handler"
	"gitlab-push-stat/internal/report"
	"gitlab-push-stat/internal/scheduler"

	"github.com/gin-gonic/gin"
)

//go:embed web/dist
var webFS embed.FS

func main() {
	// 确定配置文件路径
	configPath := "config/config.yaml"
	if envPath := os.Getenv("CONFIG_PATH"); envPath != "" {
		configPath = envPath
	}

	// 加载配置
	cfg, err := config.Load(configPath)
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// 初始化数据库
	if err := database.Init(cfg.Database.Path); err != nil {
		log.Fatalf("初始化数据库失败: %v", err)
	}

	// 初始化过滤引擎
	filterEngine := filter.NewEngine(&cfg.Filter)
	if err := filterEngine.LoadFromDB(database.GetDB()); err != nil {
		log.Printf("加载数据库过滤规则失败: %v", err)
	}

	// 初始化数据采集器
	coll := collector.NewCollector(cfg, filterEngine)

	// 初始化报表服务
	reportSvc := report.NewService()

	// 初始化定时任务调度器
	sched := scheduler.NewScheduler(cfg, coll, reportSvc, filterEngine)
	if err := sched.Start(); err != nil {
		log.Printf("启动定时任务失败: %v", err)
	}

	// 设置Gin
	gin.SetMode(cfg.Server.Mode)
	r := gin.Default()

	// 设置API路由
	handler.SetupRoutes(r, sched, coll, filterEngine)

	// 嵌入前端静态文件
	webDist, err := fs.Sub(webFS, "web/dist")
	if err != nil {
		log.Printf("加载前端资源失败: %v", err)
	} else {
		staticFS := http.FS(webDist)

		// 设置静态资源路径（assets目录）
		assetsSub, _ := fs.Sub(webFS, "web/dist/assets")
		r.StaticFS("/assets", http.FS(assetsSub))

		// 使用 NoRoute 处理 SPA 路由（返回 index.html）
		r.NoRoute(func(c *gin.Context) {
			c.FileFromFS("/", staticFS)
		})
	}

	// 优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		addr := fmt.Sprintf(":%d", cfg.Server.Port)
		log.Printf("========================================")
		log.Printf("  GitLab Push Stat 服务器启动")
		log.Printf("  访问地址: http://localhost:%d", cfg.Server.Port)
		log.Printf("  GitLab: %s", cfg.GitLab.URL)
		log.Printf("========================================")
		if err := r.Run(addr); err != nil && err != http.ErrServerClosed {
			log.Fatalf("服务器启动失败: %v", err)
		}
	}()

	<-quit
	log.Println("正在关闭服务器...")
	sched.Stop()
	log.Println("服务器已关闭")
}
