package main

import (
	"database/sql"
	"fmt"
	"os"

	_ "modernc.org/sqlite"
)

func main() {
	db, err := sql.Open("sqlite", `F:\wxff\data\decrypted\sns_plain.db`)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	// 表结构
	rows, _ := db.Query("PRAGMA table_info(SnsTimeLine)")
	fmt.Println("=== SnsTimeLine 列 ===")
	for rows.Next() {
		var cid int
		var name, ctype string
		var notnull, pk int
		var dflt sql.NullString
		rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk)
		fmt.Printf("  %2d. %-20s %s\n", cid, name, ctype)
	}
	rows.Close()

	// 取最新 3 条
	fmt.Println("\n=== 最新 3 条朋友圈 ===")
	rows, err = db.Query(`SELECT user_name, length(content), substr(content, 1, 200)
		FROM SnsTimeLine ORDER BY tid DESC LIMIT 3`)
	if err != nil {
		fmt.Println("查询失败:", err)
		os.Exit(1)
	}
	for rows.Next() {
		var user, preview string
		var clen int
		rows.Scan(&user, &clen, &preview)
		fmt.Printf("  user=%s  content_len=%d\n  预览: %q\n\n", user, clen, preview)
	}
	rows.Close()
}
