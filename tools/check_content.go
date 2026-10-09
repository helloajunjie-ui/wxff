package main

import (
	"crypto/md5"
	"database/sql"
	"encoding/hex"
	"fmt"

	_ "modernc.org/sqlite"
)

func main() {
	db, _ := sql.Open("sqlite", `F:\wxff\data\backup\_plain\message__message_0.db`)
	defer db.Close()

	sum := md5.Sum([]byte("e595400290"))
	table := "Msg_" + hex.EncodeToString(sum[:])
	fmt.Println("表名:", table)

	// 先查总共有多少条 local_type=3 的消息
	var count int
	db.QueryRow("SELECT COUNT(*) FROM " + table + " WHERE local_type=3").Scan(&count)
	fmt.Println("local_type=3 总数:", count)

	// 查 3 条，看 content 长度
	rows, _ := db.Query("SELECT local_id, length(message_content), message_content FROM " + table + " WHERE local_type=3 LIMIT 3")
	defer rows.Close()
	i := 0
	for rows.Next() {
		var id, l int
		var content []byte
		rows.Scan(&id, &l, &content)
		fmt.Printf("消息 %d: id=%d, length=%d, content_len=%d\n", i, id, l, len(content))
		if len(content) > 0 {
			fmt.Printf("  前 10 字节: % X\n", content[:10])
		}
		i++
	}
}
