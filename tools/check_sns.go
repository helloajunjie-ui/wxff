package main

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite"
)

func main() {
	db, _ := sql.Open("sqlite", `F:\wxff\data\backup\_plain\sns__sns.db`)
	defer db.Close()

	// 找一条朋友圈，看 content 里的图片 URL
	rows, _ := db.Query("SELECT content FROM SnsTimeLine LIMIT 1")
	defer rows.Close()
	var content string
	rows.Next()
	rows.Scan(&content)
	fmt.Println("朋友圈内容前 500 字:")
	fmt.Println(content[:500])

	// 收集 sns 缓存里的所有文件
	fmt.Println("\n--- sns 缓存文件 ---")
	root := `C:\Users\Administrator\xwechat_files\wxid_mpvlokr4ycnu22_dfd6\cache`
	count := 0
	filepath.Walk(root, func(p string, info os.FileInfo, err error) error {
		if err != nil || info == nil || info.IsDir() {
			return nil
		}
		if strings.Contains(p, "Sns") {
			count++
			if count <= 10 {
				fmt.Println(info.Name(), info.Size())
			}
		}
		return nil
	})
	fmt.Println("Sns 目录文件总数:", count)
}
