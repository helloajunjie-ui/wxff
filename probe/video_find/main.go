// video_find 调研微信视频存储位置。
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
	root := `C:\Users\Administrator\xwechat_files\wxid_mpvlokr4ycnu22_dfd6`

	// 1. 遍历 msg 下所有目录，找视频相关
	fmt.Println("=== 1. msg 目录树（含文件数） ===")
	msgRoot := filepath.Join(root, "msg")
	filepath.Walk(msgRoot, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			// 统计该目录下文件数
			ents, _ := os.ReadDir(p)
			n := 0
			for _, e := range ents {
				if !e.IsDir() {
					n++
				}
			}
			rel, _ := filepath.Rel(msgRoot, p)
			fmt.Printf("[D] %-50s files=%d\n", rel, n)
		}
		return nil
	})

	// 2. 找所有 .mp4 / .mov / .avi / .silk / .amr / .wav
	fmt.Println("\n=== 2. 视频/音频文件（全盘 msg+cache） ===")
	exts := map[string]bool{".mp4": true, ".mov": true, ".avi": true, ".silk": true, ".amr": true, ".wav": true, ".mp3": true, ".m4a": true}
	for _, sub := range []string{"msg", "cache"} {
		base := filepath.Join(root, sub)
		filepath.Walk(base, func(p string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() {
				return nil
			}
			ext := strings.ToLower(filepath.Ext(p))
			if exts[ext] {
				rel, _ := filepath.Rel(root, p)
				fmt.Printf("%-80s %d bytes\n", rel, info.Size())
			}
			return nil
		})
	}

	// 3. 查 message_0.db 里的视频消息
	fmt.Println("\n=== 3. message_0.db 视频消息 ===")
	mdb := `F:\wxff\data\decrypted\message__message_0.db`
	db, err := sql.Open("sqlite", mdb)
	if err != nil {
		fmt.Println("open err:", err)
		return
	}
	defer db.Close()

	// 列出所有表
	rows, _ := db.Query("SELECT name FROM sqlite_master WHERE type='table'")
	var tables []string
	for rows.Next() {
		var n string
		rows.Scan(&n)
		tables = append(tables, n)
	}
	rows.Close()
	fmt.Println("tables:", strings.Join(tables, ", "))

	// 找含 video 的表
	for _, t := range tables {
		if strings.Contains(strings.ToLower(t), "video") {
			fmt.Printf("\n--- 表 %s 结构 ---\n", t)
			cr, _ := db.Query(fmt.Sprintf("SELECT sql FROM sqlite_master WHERE name='%s'", t))
			for cr.Next() {
				var s string
				cr.Scan(&s)
				fmt.Println(s)
			}
			cr.Close()
			cnt := 0
			db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s", t)).Scan(&cnt)
			fmt.Printf("行数: %d\n", cnt)
		}
	}
}
