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

	wxid := "wxid_i49id3jztuzq12"
	rows, _ := db.Query("SELECT content FROM SnsTimeLine WHERE user_name=? AND content LIKE '%thumb%' LIMIT 2", wxid)
	defer rows.Close()
	i := 0
	for rows.Next() {
		var content string
		rows.Scan(&content)
		i++
		fmt.Printf("=== 朋友圈 %d ===\n", i)
		re := regexp.MustCompile(`<thumb[^>]*>([^<]+)</thumb>`)
		matches := re.FindAllStringSubmatch(content, -1)
		fmt.Println("thumb URLs:")
		for _, m := range matches {
			fmt.Println(" ", m[1])
		}
		re2 := regexp.MustCompile(`<url[^>]*>([^<]+)</url>`)
		matches2 := re2.FindAllStringSubmatch(content, -1)
		fmt.Println("url URLs:")
		for _, m := range matches2 {
			fmt.Println(" ", m[1])
		}
		fmt.Println()
	}
}
