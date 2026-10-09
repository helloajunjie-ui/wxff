package main

import (
	"bytes"
	"database/sql"
	"fmt"
	"os"

	"github.com/klauspost/compress/zstd"
	_ "modernc.org/sqlite"
)

func main() {
	db, _ := sql.Open("sqlite", `F:\wxff\data\decrypted\message_0.db`)
	defer db.Close()

	const t = "Msg_c41483b0a7c9b84d7237e00b19256971"

	dec, _ := zstd.NewReader(nil)
	defer dec.Close()

	rows, err := db.Query("SELECT local_id, local_type, create_time, source FROM " + t + " ORDER BY local_id DESC LIMIT 5")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer rows.Close()
	for rows.Next() {
		var id, mtype, ctime int64
		var src []byte
		rows.Scan(&id, &mtype, &ctime, &src)
		fmt.Println("---")
		fmt.Printf("  id=%d type=%d time=%d\n", id, mtype, ctime)
		if len(src) > 4 && src[0] == 0x28 && src[1] == 0xb5 && src[2] == 0x2f && src[3] == 0xfd {
			plain, err := dec.DecodeAll(src, nil)
			if err != nil {
				fmt.Println("  zstd err:", err)
				continue
			}
			s := string(plain)
			if len(s) > 300 {
				s = s[:300] + "..."
			}
			fmt.Printf("  内容: %s\n", s)
		} else {
			fmt.Printf("  (非 zstd) source前4字节: %x\n", src[:4])
		}
	}
	_ = bytes.MinRead
	os.Exit(0)
}
