package main

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

func main() {
	db, _ := sql.Open("sqlite", `F:\wxff\data\backup\_plain\sns__sns.db`)
	defer db.Close()

	wxid := "wxid_i49id3jztuzq12"
	rows, _ := db.Query("SELECT content FROM SnsTimeLine WHERE user_name=? AND content LIKE '%thumb%' LIMIT 1", wxid)
	defer rows.Close()
	var content string
	rows.Next()
	rows.Scan(&content)
	fmt.Println(content)
}
