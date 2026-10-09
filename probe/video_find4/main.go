// video_find4 确认：1) 视频是否本地有文件 2) 语音 SILK 与消息的对应关系
package main

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/klauspost/compress/zstd"
	_ "modernc.org/sqlite"
)

func main() {
	root := `C:\Users\Administrator\xwechat_files\wxid_mpvlokr4ycnu22_dfd6`

	// 1. 全盘找 .mp4/.mov（含所有子目录，排除已知）
	fmt.Println("=== 1. 全盘视频文件（.mp4/.mov/.avi/.mkv） ===")
	exts := map[string]bool{".mp4": true, ".mov": true, ".avi": true, ".mkv": true}
	filepath.Walk(root, func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if exts[strings.ToLower(filepath.Ext(p))] {
			rel, _ := filepath.Rel(root, p)
			fmt.Printf("%-90s %d bytes\n", rel, info.Size())
		}
		return nil
	})

	// 2. 语音：media_0.db VoiceInfo 与 message db voicemsg 的对应
	fmt.Println("\n=== 2. 语音对应关系 ===")
	mdb := `F:\wxff\data\decrypted\message__message_0.db`
	db, _ := sql.Open("sqlite", mdb)
	defer db.Close()
	dec, _ := zstd.NewReader(nil)
	defer dec.Close()

	// 找所有 voicemsg，打印 create_time + fromusername + voicelength
	tabs, _ := db.Query("SELECT name FROM sqlite_master WHERE type='table' AND name LIKE 'Msg_%'")
	var names []string
	for tabs.Next() {
		var n string
		tabs.Scan(&n)
		names = append(names, n)
	}
	tabs.Close()

	voiceCount := 0
	for _, t := range names {
		rows, err := db.Query(fmt.Sprintf("SELECT local_id, create_time, message_content FROM %s WHERE local_type=34", t))
		if err != nil {
			continue
		}
		for rows.Next() {
			var lid, ct int64
			var mc []byte
			rows.Scan(&lid, &ct, &mc)
			plain := mc
			if len(mc) > 4 && mc[0] == 0x28 && mc[1] == 0xb5 {
				if out, err := dec.DecodeAll(mc, nil); err == nil {
					plain = out
				}
			}
			if voiceCount < 5 {
				fmt.Printf("[%s] lid=%d ct=%d\n  %s\n", t, lid, ct, truncate(string(plain), 300))
			}
			voiceCount++
		}
		rows.Close()
	}
	fmt.Printf("语音消息总数: %d\n", voiceCount)

	// 3. media_0.db VoiceInfo 样本
	fmt.Println("\n=== 3. media_0.db VoiceInfo 样本 ===")
	vdb, _ := sql.Open("sqlite", `F:\wxff\data\decrypted\media_0.db`)
	defer vdb.Close()
	vr, _ := vdb.Query("SELECT chat_name_id, create_time, local_id, svr_id, length(voice_data) FROM VoiceInfo ORDER BY create_time LIMIT 5")
	for vr.Next() {
		var cn, ct, lid, svr, ln int64
		vr.Scan(&cn, &ct, &lid, &svr, &ln)
		fmt.Printf("chat_name_id=%d create_time=%d local_id=%d svr_id=%d voice_data=%d bytes\n", cn, ct, lid, svr, ln)
	}
	vr.Close()

	// Name2Id 全量
	fmt.Println("\n=== 4. Name2Id 映射 ===")
	nr, _ := vdb.Query("SELECT rowid, user_name FROM Name2Id")
	for nr.Next() {
		var rid int64
		var un string
		nr.Scan(&rid, &un)
		fmt.Printf("rowid=%d user_name=%s\n", rid, un)
	}
	nr.Close()
}

func truncate(s string, n int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) > n {
		return s[:n] + "..."
	}
	return s
}
