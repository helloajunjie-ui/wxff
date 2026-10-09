package main

import (
	"database/sql"
	"fmt"
	"regexp"

	_ "modernc.org/sqlite"
)

func main() {
	db, _ := sql.Open("sqlite", `F:\wxff\data\backup\_plain\sns__sns.db`)
	defer db.Close()

	rows, _ := db.Query("SELECT content FROM SnsTimeLine WHERE content LIKE '%thumb%' LIMIT 1")
	defer rows.Close()
	var content string
	rows.Next()
	rows.Scan(&content)

	// 提取所有 thumb URL
	re := regexp.MustCompile(`<thumb[^>]*>([^<]+)</thumb>`)
	matches := re.FindAllStringSubmatch(content, -1)
	fmt.Println("thumb URLs:")
	for _, m := range matches {
		fmt.Println(" ", m[1])
	}

	// 提取 media 里的 url
	re2 := regexp.MustCompile(`<url[^>]*>([^<]+)</url>`)
	matches2 := re2.FindAllStringSubmatch(content, -1)
	fmt.Println("\nurl URLs:")
	for _, m := range matches2 {
		fmt.Println(" ", m[1])
	}
}
