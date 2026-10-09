// media_search 在 media_0.db 中搜索朋友圈图片 md5，寻找映射关系。
package main

import (
	"database/sql"
	"fmt"
	"os"

	_ "modernc.org/sqlite"
)

func main() {
	dbPath := `F:\wxff\data\decrypted\media_0.db`
	if len(os.Args) > 1 {
		dbPath = os.Args[1]
	}
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		panic(err)
	}
	defer db.Close()

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
	fmt.Println("=== media_0.db 表 ===")
	for _, t := range tables {
		fmt.Println(" ", t)
	}

	targets := []string{"e8176b50b9edd302944f2d777b2c56a0", "d350d1aa597abc81db208d4a4110fec8", "68e745aa893c22bdfc1d37a815b43a17"}
	fmt.Println("\n=== 搜索朋友圈图片 md5 ===")
	for _, t := range tables {
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
	fmt.Println("搜索完成")
}
