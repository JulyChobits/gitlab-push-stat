package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func main() {
	db, err := gorm.Open(sqlite.Open("data/gitlab-stat.db"), &gorm.Config{})
	if err != nil {
		log.Fatal("open db:", err)
	}

	type Row struct {
		ID          string `gorm:"column:id"`
		AuthorName  string `gorm:"column:author_name"`
		AuthorEmail string `gorm:"column:author_email"`
	}

	var rows []Row
	db.Raw("SELECT id, author_name, author_email FROM commits").Scan(&rows)

	updated := 0
	for _, r := range rows {
		if r.AuthorEmail == "" {
			continue
		}
		prefix := r.AuthorEmail
		if idx := strings.Index(r.AuthorEmail, "@"); idx > 0 {
			prefix = r.AuthorEmail[:idx]
		}
		if r.AuthorName != prefix {
			db.Exec("UPDATE commits SET author_name = ? WHERE id = ?", prefix, r.ID)
			updated++
		}
	}
	fmt.Printf("Total: %d, Updated: %d\n", len(rows), updated)
}
