package main

import (
	"fmt"
	"log"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func main() {
	db, err := gorm.Open(sqlite.Open("data/gitlab-stat.db"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		log.Fatal("Error:", err)
	}

	// Add columns if they don't exist (SQLite will ignore if column already exists)
	db.Exec("ALTER TABLE commits ADD COLUMN gitlab_name VARCHAR(255)")
	db.Exec("ALTER TABLE commits ADD COLUMN gitlab_username VARCHAR(255)")

	// Create indexes
	db.Exec("CREATE INDEX IF NOT EXISTS idx_commits_gitlab_name ON commits(gitlab_name)")
	db.Exec("CREATE INDEX IF NOT EXISTS idx_commits_gitlab_username ON commits(gitlab_username)")

	fmt.Println("Done - gitlab_name and gitlab_username fields added to commits table")
}
