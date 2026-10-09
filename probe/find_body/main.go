package main

import (
	"database/sql"
	"fmt"
	"os"

	"github.com/klauspost/compress/zstd"
	_ "modernc.org/sqlite"
)

func tryDecode(b []byte) string {
	if len(b) == 0 {
		return ""
	}
	if len(b) >= 4 && b[0] == 0x28 && b[1] == 0xb5 && b[2] == 0x2f && b[3] == 0xfd {
		d, err := zstd.NewReader(nil)
		if err != nil {
			return "(zstd init err)"
		}
		defer d.Close()
		out, err := d.DecodeAll(b, nil)
		if err != nil {
			return "(zstd err: " + err.Error() + ")"
		}
		return string(out)
	}
	return "(非zstd: " + string(b[:min(40, len(b))]) + ")"
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func main() {
	db, _ := sql.Open("sqlite", `F:\wxff\data\decrypted\message_0.db`)
	defer db.Close()

	const t = "Msg_c41483b0a7c9b84d7237e00b19256971"

	rows, err := db.Query(`SELECT local_id, local_type, create_time, source, message_content, compress_content
		FROM ` + t + ` ORDER BY local_id DESC LIMIT 8`)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer rows.Close()
	for rows.Next() {
		var id, mtype, ctime int64
		var src, mc, cc []byte
		rows.Scan(&id, &mtype, &ctime, &src, &mc, &cc)
		fmt.Printf("--- id=%d type=%d time=%d\n", id, mtype, ctime)
		fmt.Printf("  source len=%d: %s\n", len(src), truncate(tryDecode(src), 200))
		fmt.Printf("  msg_content len=%d: %s\n", len(mc), truncate(tryDecode(mc), 200))
		fmt.Printf("  compress len=%d: %s\n", len(cc), truncate(tryDecode(cc), 200))
	}
	os.Exit(0)
}

func truncate(s string, n int) string {
	if len(s) > n {
		return s[:n] + "..."
	}
	return s
}
