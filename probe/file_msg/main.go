// probe/file_msg 找出哪个好友有文件消息（type 49 + appattach），并打印 title。
package main

import (
	"database/sql"
	"fmt"
	"regexp"

	"github.com/klauspost/compress/zstd"
	_ "modernc.org/sqlite"
)

func main() {
	db, err := sql.Open("sqlite", `data\decrypted\message__message_0.db`)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	rows, _ := db.Query("SELECT name FROM sqlite_master WHERE type='table' AND name LIKE 'Msg_%'")
	var tables []string
	for rows.Next() {
		var n string
		rows.Scan(&n)
		tables = append(tables, n)
	}
	rows.Close()

	dec, _ := zstd.NewReader(nil)
	defer dec.Close()

	titleRe := regexp.MustCompile(`<title>([^<]*)</title>`)
	total := 0
	for _, t := range tables {
		r, err := db.Query("SELECT local_type, message_content FROM " + t + " WHERE (local_type & 65535)=49")
		if err != nil {
			continue
		}
		n := 0
		for r.Next() {
			var lt int64
			var content []byte
			r.Scan(&lt, &content)
			text := string(content)
			if len(content) > 4 && content[0] == 0x28 && content[1] == 0xb5 {
				if d, e := dec.DecodeAll(content, nil); e == nil {
					text = string(d)
				}
			}
			hasAttach := false
			if len(text) > 0 {
				hasAttach = regexp.MustCompile(`appattach`).MatchString(text)
			}
			if hasAttach {
				title := ""
				if m := titleRe.FindStringSubmatch(text); len(m) > 1 {
					title = m[1]
				}
				if n < 3 {
					fmt.Printf("[%s] raw=%d title=%q\n", t[:12], lt, title)
				}
				n++
				total++
			}
		}
		r.Close()
		if n > 0 {
			fmt.Printf("  ^ 表 %s 有 %d 条文件消息\n", t[:12], n)
		}
	}
	fmt.Println("总文件消息:", total)
}
