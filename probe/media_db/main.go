package main

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

func main() {
	root := `C:\Users\Administrator\xwechat_files\wxid_mpvlokr4ycnu22_dfd6`

	// 1. 找所有 media_*.db
	fmt.Println("=== media db 文件 ===")
	filepath.Walk(filepath.Join(root, "db_storage"), func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if filepath.Ext(p) == ".db" {
			fmt.Printf("  %s (%d B)\n", p, info.Size())
		}
		return nil
	})

	// 2. 打开解密后的 media_0.db，列出表
	mediaDb := `F:\wxff\data\decrypted\media_0.db`
	if _, err := os.Stat(mediaDb); err != nil {
		fmt.Println("\nmedia_0.db 不存在，跳过")
		return
	}
	db, err := sql.Open("sqlite", mediaDb)
	if err != nil {
		fmt.Println("open:", err)
		return
	}
	defer db.Close()

	fmt.Println("\n=== media_0.db 表 ===")
	rows, err := db.Query("SELECT name FROM sqlite_master WHERE type='table'")
	if err != nil {
		fmt.Println("tables:", err)
		return
	}
	var tables []string
	for rows.Next() {
		var n string
		rows.Scan(&n)
		tables = append(tables, n)
		fmt.Println("  ", n)
	}
	rows.Close()

	// 3. 对每个表，打印 schema + 行数
	for _, t := range tables {
		var cnt int
		db.QueryRow("SELECT count(*) FROM " + t).Scan(&cnt)
		fmt.Printf("\n--- %s (%d 行) ---\n", t, cnt)
		sr, err := db.Query("SELECT sql FROM sqlite_master WHERE name=?", t)
		if err == nil {
			for sr.Next() {
				var s string
				sr.Scan(&s)
				fmt.Println("  ", s)
			}
			sr.Close()
		}
		// 打印前 3 行
		cr, err := db.Query("SELECT * FROM " + t + " LIMIT 3")
		if err == nil {
			cols, _ := cr.Columns()
			fmt.Println("  列:", cols)
			for cr.Next() {
				vals := make([]any, len(cols))
				ptrs := make([]any, len(cols))
				for i := range vals {
					ptrs[i] = &vals[i]
				}
				cr.Scan(ptrs...)
				for i, c := range cols {
					v := vals[i]
					if b, ok := v.([]byte); ok {
						if len(b) > 60 {
							v = fmt.Sprintf("<%d bytes>", len(b))
						} else {
							v = string(b)
						}
					}
					fmt.Printf("     %s=%v\n", c, v)
				}
				fmt.Println("     ---")
			}
			cr.Close()
		}
	}
}
