package main

import (
	"database/sql"
	"fmt"
	"os"
	"regexp"

	"github.com/klauspost/compress/zstd"
	_ "modernc.org/sqlite"
)

func main() {
	db, _ := sql.Open("sqlite", `F:\wxff\data\decrypted\message_0.db`)
	defer db.Close()

	// 女王大人 wxid_5wi83mtrxlpt21 → md5 = 25cd413cef8aed1f6591d353a93555ad
	table := "Msg_25cd413cef8aed1f6591d353a93555ad"

	dec, _ := zstd.NewReader(nil)
	defer dec.Close()

	rows, _ := db.Query("SELECT local_id, local_type, create_time, message_content FROM " + table +
		" WHERE local_type=3 ORDER BY local_id DESC LIMIT 3")
	defer rows.Close()
	for rows.Next() {
		var id, mtype, ctime int64
		var content []byte
		rows.Scan(&id, &mtype, &ctime, &content)
		text := string(content)
		if len(text) >= 4 && text[0] == 0x28 && text[1] == 0xb5 && text[2] == 0x2f && text[3] == 0xfd {
			if out, err := dec.DecodeAll(content, nil); err == nil {
				text = string(out)
			}
		}
		fmt.Printf("--- id=%d time=%d ---\n", id, ctime)
		// 抽 md5 和 aeskey
		md5Re := regexp.MustCompile(`md5="([^"]+)"`)
		aesRe := regexp.MustCompile(`aeskey="([^"]+)"`)
		if m := md5Re.FindStringSubmatch(text); len(m) > 1 {
			fmt.Printf("  md5: %s\n", m[1])
		}
		if m := aesRe.FindStringSubmatch(text); len(m) > 1 {
			fmt.Printf("  aeskey: %s\n", m[1])
		}
		// 完整 XML 前 500
		if len(text) > 500 {
			text = text[:500] + "..."
		}
		fmt.Printf("  XML: %s\n\n", text)
	}
	os.Exit(0)
}
