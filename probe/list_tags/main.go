package main

import (
	"database/sql"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"

	_ "modernc.org/sqlite"
)

func main() {
	db, _ := sql.Open("sqlite", `F:\wxff\data\decrypted\sns_plain.db`)
	defer db.Close()

	rows, _ := db.Query(`SELECT content FROM SnsTimeLine`)
	defer rows.Close()
	tagRe := regexp.MustCompile(`<([a-zA-Z][a-zA-Z0-9_]*)[ />]`)
	tagSet := map[string]int{}
	sample := map[string]string{}
	for rows.Next() {
		var c string
		rows.Scan(&c)
		for _, m := range tagRe.FindAllStringSubmatch(c, -1) {
			tagSet[m[1]]++
			if _, ok := sample[m[1]]; !ok {
				// 找这个标签的第一个出现位置，截一段
				idx := strings.Index(c, "<"+m[1])
				end := idx + 200
				if end > len(c) {
					end = len(c)
				}
				sample[m[1]] = c[idx:end]
			}
		}
	}
	names := make([]string, 0, len(tagSet))
	for n := range tagSet {
		names = append(names, n)
	}
	sort.Strings(names)
	fmt.Println("=== 出现的 XML 标签 ===")
	for _, n := range names {
		fmt.Printf("  %-30s %d 次\n", n, tagSet[n])
	}
	fmt.Println("\n=== 关键标签样本 ===")
	for _, key := range []string{"pic", "img", "url", "thumb", "video", "media", "object", "contentObj", "location", "detailUrl"} {
		for n := range sample {
			if strings.Contains(strings.ToLower(n), strings.ToLower(key)) {
				fmt.Printf("\n<%s>:\n  %s\n", n, sample[n])
			}
		}
	}
	os.Exit(0)
}
