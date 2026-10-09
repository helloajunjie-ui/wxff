package main

import (
	"database/sql"
	"fmt"
	"os"
	"strings"

	_ "modernc.org/sqlite"
)

func main() {
	db, _ := sql.Open("sqlite", `F:\wxff\data\decrypted\sns_plain.db`)
	defer db.Close()

	// 找一条带 pics 的朋友圈
	var content string
	err := db.QueryRow(`SELECT content FROM SnsTimeLine
		WHERE content LIKE '%<pics>%' ORDER BY tid DESC LIMIT 1`).Scan(&content)
	if err != nil {
		fmt.Println("没找到带图的:", err)
		os.Exit(1)
	}
	// 提取 <pics> 段
	start := strings.Index(content, "<pics>")
	end := strings.Index(content[start:], "</pics>") + len("</pics>")
	fmt.Println("=== 完整 contentDesc ===")
	if i := strings.Index(content, "<contentDesc>"); i >= 0 {
		j := strings.Index(content[i:], "</contentDesc>")
		fmt.Println(content[i : i+j+len("</contentDesc>")])
	}
	fmt.Println("\n=== pics 段（前 1500 字符）===")
	seg := content[start : start+end]
	if len(seg) > 1500 {
		seg = seg[:1500] + "..."
	}
	fmt.Println(seg)
	os.Exit(0)
}
