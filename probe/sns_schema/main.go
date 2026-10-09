// sns_schema 打印 sns.db 的所有表与列，并搜索给定 md5/key 是否出现在任何表。
package main

import (
	"database/sql"
	"fmt"
	"os"
	"strings"

	_ "modernc.org/sqlite"
)

func main() {
	dbPath := `F:\wxff\data\backup\_plain\sns__sns.db`
	if len(os.Args) > 1 {
		dbPath = os.Args[1]
	}
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	// 列出所有表
	trows, err := db.Query("SELECT name FROM sqlite_master WHERE type='table'")
	if err != nil {
		panic(err)
	}
	var tables []string
	for trows.Next() {
		var n string
		trows.Scan(&n)
		tables = append(tables, n)
	}
	trows.Close()
	fmt.Println("=== 表 ===")
	for _, t := range tables {
		fmt.Println(" ", t)
	}

	// 搜索目标字符串
	targets := []string{"e8176b50b9edd302944f2d777b2c56a0", "259629083086821010", "14715872012089635445"}
	fmt.Println("\n=== 搜索目标字符串 ===")
	for _, t := range tables {
		// 取列名
		crows, err := db.Query("PRAGMA table_info(" + t + ")")
		if err != nil {
			continue
		}
		var cols []string
		for crows.Next() {
			var cid int
			var name, ctype string
			var notnull, pk int
			var dflt sql.NullString
			crows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk)
			cols = append(cols, name)
		}
		crows.Close()
		if len(cols) == 0 {
			continue
		}
		// 对每个文本列做 LIKE 搜索
		for _, col := range cols {
			for _, tg := range targets {
				q := fmt.Sprintf("SELECT count(*) FROM %s WHERE %s LIKE '%%%s%%'", t, col, tg)
				var cnt int
				if err := db.QueryRow(q).Scan(&cnt); err == nil && cnt > 0 {
					fmt.Printf("  命中: 表=%s 列=%s 目标=%s 次数=%d\n", t, col, tg, cnt)
				}
			}
		}
	}

	// 打印 SnsTimeLine 列
	fmt.Println("\n=== SnsTimeLine 列 ===")
	crows, _ := db.Query("PRAGMA table_info(SnsTimeLine)")
	for crows.Next() {
		var cid int
		var name, ctype string
		var notnull, pk int
		var dflt sql.NullString
		crows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk)
		fmt.Printf("  %s %s\n", name, ctype)
	}
	crows.Close()

	_ = strings.TrimSpace
}
