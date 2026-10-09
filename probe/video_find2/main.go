// video_find2 深入调研 Rec 目录与消息表视频/语音字段。
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

	// 1. 列出所有 Rec 目录下的文件（含大小、头部字节）
	fmt.Println("=== 1. Rec 目录文件 ===")
	msgRoot := filepath.Join(root, "msg")
	filepath.Walk(msgRoot, func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if strings.Contains(p, `\Rec\`) {
			rel, _ := filepath.Rel(root, p)
			head := make([]byte, 16)
			f, _ := os.Open(p)
			n, _ := f.Read(head)
			f.Close()
			fmt.Printf("%-90s %d bytes  head=%x\n", rel, info.Size(), head[:n])
		}
		return nil
	})

	// 2. 列出 migrate\Video 文件头
	fmt.Println("\n=== 2. migrate\\Video 文件头 ===")
	filepath.Walk(filepath.Join(root, "msg", "migrate"), func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		rel, _ := filepath.Rel(root, p)
		head := make([]byte, 16)
		f, _ := os.Open(p)
		n, _ := f.Read(head)
		f.Close()
		fmt.Printf("%-90s %d bytes  head=%x\n", rel, info.Size(), head[:n])
		return nil
	})

	// 3. 消息表结构 + 视频/语音消息样本
	fmt.Println("\n=== 3. 消息表结构 ===")
	mdb := `F:\wxff\data\decrypted\message__message_0.db`
	db, err := sql.Open("sqlite", mdb)
	if err != nil {
		fmt.Println("open err:", err)
		return
	}
	defer db.Close()

	// 取一个 Msg_ 表看结构
	var tbl string
	db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name LIKE 'Msg_%' LIMIT 1").Scan(&tbl)
	fmt.Println("样本表:", tbl)
	cr, _ := db.Query(fmt.Sprintf("SELECT sql FROM sqlite_master WHERE name='%s'", tbl))
	for cr.Next() {
		var s string
		cr.Scan(&s)
		fmt.Println(s)
	}
	cr.Close()

	// 统计各消息类型数量（local_type）
	fmt.Println("\n=== 4. 消息类型分布（local_type） ===")
	rows, _ := db.Query(fmt.Sprintf("SELECT local_type, COUNT(*) FROM %s GROUP BY local_type ORDER BY COUNT(*) DESC", tbl))
	for rows.Next() {
		var t, n int
		rows.Scan(&t, &n)
		fmt.Printf("local_type=%-4d count=%d\n", t, n)
	}
	rows.Close()

	// 5. 找视频消息（type 43）与语音消息（type 34）样本
	for _, typ := range []int{43, 34, 3, 49} {
		fmt.Printf("\n=== 5. local_type=%d 样本 ===\n", typ)
		r, _ := db.Query(fmt.Sprintf("SELECT local_id, create_time, message_content, compress_content FROM %s WHERE local_type=? LIMIT 2", tbl), typ)
		for r.Next() {
			var lid, ct int64
			var mc, cc []byte
			r.Scan(&lid, &ct, &mc, &cc)
			fmt.Printf("local_id=%d create_time=%d\n  message_content(%d)=%s\n  compress_content(%d)=%x\n",
				lid, ct, len(mc), truncate(string(mc), 400), len(cc), truncateBytes(cc, 64))
		}
		r.Close()
	}
}

func truncate(s string, n int) string {
	if len(s) > n {
		return s[:n] + "..."
	}
	return s
}

func truncateBytes(b []byte, n int) []byte {
	if len(b) > n {
		return b[:n]
	}
	return b
}
