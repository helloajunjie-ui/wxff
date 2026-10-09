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

	// 查 local_type=3 的消息，看 message_content 长度
	rows, _ := db.Query("SELECT length(message_content), message_content FROM " + table + " WHERE local_type=3 LIMIT 3")
	defer rows.Close()
	i := 0
	for rows.Next() {
		var l int
		var content []byte
		rows.Scan(&l, &content)
		fmt.Printf("消息 %d: 长度=%d, 前10字节: % X\n", i, l, content[:10])
		i++
	}
}
