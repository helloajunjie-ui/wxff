// video_find3 解压 message_content（zstd）看视频/语音消息 XML。
package main

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/klauspost/compress/zstd"
	_ "modernc.org/sqlite"
)

func main() {
	mdb := `F:\wxff\data\decrypted\message__message_0.db`
	db, err := sql.Open("sqlite", mdb)
	if err != nil {
		fmt.Println("open err:", err)
		return
	}
	defer db.Close()

	dec, _ := zstd.NewReader(nil)
	defer dec.Close()

	var tbl string
	db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name LIKE 'Msg_%' LIMIT 1").Scan(&tbl)

	// 遍历所有 Msg_ 表，找 type 43(视频) / 34(语音) / 3(图片) / 49(文件)
	tabs, _ := db.Query("SELECT name FROM sqlite_master WHERE type='table' AND name LIKE 'Msg_%'")
	var names []string
	for tabs.Next() {
		var n string
		tabs.Scan(&n)
		names = append(names, n)
	}
	tabs.Close()

	seen := map[int]int{}
	for _, t := range names {
		rows, err := db.Query(fmt.Sprintf("SELECT local_id, local_type, create_time, message_content FROM %s WHERE local_type IN (3,34,43,47,49,42,10000)", t))
		if err != nil {
			continue
		}
		for rows.Next() {
			var lid, lt, ct int64
			var mc []byte
			rows.Scan(&lid, &lt, &ct, &mc)
			if seen[int(lt)] >= 2 {
				continue
			}
			seen[int(lt)]++
			// 尝试 zstd 解压
			plain := mc
			if len(mc) > 4 && mc[0] == 0x28 && mc[1] == 0xb5 && mc[2] == 0x2f && mc[3] == 0xfd {
				if out, err := dec.DecodeAll(mc, nil); err == nil {
					plain = out
				}
			}
			fmt.Printf("\n===== %s local_type=%d local_id=%d =====\n%s\n", t, lt, lid, truncate(string(plain), 600))
		}
		rows.Close()
	}
}

func truncate(s string, n int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) > n {
		return s[:n] + "..."
	}
	return s
}
