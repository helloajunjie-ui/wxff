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

	rows, _ := db.Query("SELECT message_content FROM " + table + " WHERE local_type=3 LIMIT 1")
	defer rows.Close()
	var content []byte
	rows.Scan(&content)

	fmt.Println("content 长度:", len(content))
	fmt.Println("前 20 字节:")
	for i := 0; i < 20 && i < len(content); i++ {
		fmt.Printf("%02X ", content[i])
	}
	fmt.Println()
	fmt.Println("作为字符串前 200 字:")
	fmt.Println(string(content[:200]))
}
