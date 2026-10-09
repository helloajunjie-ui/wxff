package main

import (
	"crypto/md5"
	"database/sql"
	"encoding/hex"
	"fmt"
	"os"
	"strings"

	_ "modernc.org/sqlite"
)

func main() {
	// 列出所有 Msg_xxx 表
	db, _ := sql.Open("sqlite", `F:\wxff\data\decrypted\message_0.db`)
	defer db.Close()

	rows, _ := db.Query("SELECT name FROM sqlite_master WHERE type='table' AND name LIKE 'Msg_%'")
	tables := []string{}
	for rows.Next() {
		var n string
		rows.Scan(&n)
		tables = append(tables, n)
	}
	rows.Close()

	// 算常见 wxid 的 md5 看能不能匹配
	candidates := []string{
		"guyue119",
		"wxid_m6qaw8adu0w522",
		"wxid_i49id3jztuzq12",
	}
	fmt.Println("=== md5 匹配 ===")
	for _, wxid := range candidates {
		sum := md5.Sum([]byte(wxid))
		m := hex.EncodeToString(sum[:])
		found := ""
		for _, t := range tables {
			if strings.EqualFold(t, "Msg_"+m) {
				found = t
				break
			}
		}
		fmt.Printf("  md5(%s) = %s → %s\n", wxid, m, found)
	}

	// 反过来：遍历所有表，找最近一条消息时间
	fmt.Println("\n=== 最近活跃会话（按最新消息时间）===")
	type s struct {
		table string
		last  int64
		cnt   int64
	}
	list := []s{}
	for _, t := range tables {
		var last, cnt int64
		db.QueryRow("SELECT max(create_time), count(*) FROM "+t).Scan(&last, &cnt)
		list = append(list, s{t, last, cnt})
	}
	// 按时间倒序
	for i := 0; i < len(list); i++ {
		for j := i + 1; j < len(list); j++ {
			if list[j].last > list[i].last {
				list[i], list[j] = list[j], list[i]
			}
		}
	}
	for i := 0; i < 10 && i < len(list); i++ {
		fmt.Printf("  %s  last=%d  msgs=%d\n", list[i].table, list[i].last, list[i].cnt)
	}
	os.Exit(0)
}
