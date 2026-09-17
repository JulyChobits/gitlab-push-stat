package database

import (
	"log"
	"os"
	"path/filepath"

	"gitlab-push-stat/internal/model"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

// Init 初始化数据库
func Init(dbPath string) error {
	// 确保目录存在
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	var err error
	DB, err = gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		log.Printf("数据库连接失败: %v", err)
		return err
	}

	// 自动迁移
	if err := DB.AutoMigrate(
		&model.AuthorMapping{},
		&model.Project{},
		&model.Commit{},
		&model.DailyReport{},
		&model.WeeklyReport{},
		&model.MonthlyReport{},
		&model.FilterRule{},
		&model.SyncLog{},
		&model.AIReview{},
	); err != nil {
		log.Printf("数据库迁移失败: %v", err)
		return err
	}

	log.Printf("数据库初始化成功: %s", dbPath)
	return nil
}

// GetDB 获取数据库实例
func GetDB() *gorm.DB {
	return DB
}
