package main

import (
	"database/sql"
	"fmt"
	"os"
	"regexp"

	_ "modernc.org/sqlite"
)

func main() {
	db, _ := sql.Open("sqlite", `F:\wxff\data\decrypted\sns_plain.db`)
	defer db.Close()

	var content string
	// 找一条带 mediaList + 有文字的朋友圈
	err := db.QueryRow(`SELECT content FROM SnsTimeLine
		WHERE content LIKE '%<mediaList>%' AND content LIKE '%<contentDesc>%'
		AND content NOT LIKE '%<contentDesc></contentDesc>%'
		ORDER BY tid DESC LIMIT 1`).Scan(&content)
	if err != nil {
		fmt.Println("没找到:", err)
		os.Exit(1)
	}

	get := func(tag string) string {
		re := regexp.MustCompile(`<` + tag + `>([^<]*)</` + tag + `>`)
		m := re.FindStringSubmatch(content)
		if len(m) < 2 {
			return ""
		}
		return m[1]
	}
	getAttr := func(tag, attr string) string {
		re := regexp.MustCompile(`<` + tag + `[^>]*` + attr + `="([^"]*)"`)
		m := re.FindStringSubmatch(content)
		if len(m) < 2 {
			return ""
		}
		return m[1]
	}

	fmt.Println("===== 一条真实朋友圈 =====")
	fmt.Printf("发布人 wxid : %s\n", get("username"))
	fmt.Printf("时间戳      : %s\n", get("createTime"))
	fmt.Printf("文字        : %q\n\n", get("contentDesc"))

	// 抽所有 <media> 块
	mediaRe := regexp.MustCompile(`<media>(.*?)</media>`)
	medias := mediaRe.FindAllStringSubmatch(content, -1)
	fmt.Printf("图片/视频数  : %d\n", len(medias))
	for i, m := range medias {
		body := m[1]
		thumbRe := regexp.MustCompile(`<thumb[^>]*>([^<]+)</thumb>`)
		fullRe := regexp.MustCompile(`<url[^>]*>([^<]+)</url>`)
		descRe := regexp.MustCompile(`<description>([^<]*)</description>`)
		fmt.Printf("\n--- 媒体 %d ---\n", i+1)
		if mm := descRe.FindStringSubmatch(body); len(mm) > 1 {
			fmt.Printf("  描述: %q\n", mm[1])
		}
		if mm := thumbRe.FindStringSubmatch(body); len(mm) > 1 {
			fmt.Printf("  缩略图: %s\n", mm[1])
		}
		if mm := fullRe.FindStringSubmatch(body); len(mm) > 1 {
			fmt.Printf("  原图URL: %s\n", mm[1])
		}
	}
	_ = getAttr
	os.Exit(0)
}
