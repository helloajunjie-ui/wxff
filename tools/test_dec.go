package main

import (
	"crypto/md5"
	"database/sql"
	"encoding/hex"
	"fmt"

	"github.com/klauspost/compress/zstd"
	_ "modernc.org/sqlite"
)

func main() {
	db, _ := sql.Open("sqlite", `F:\wxff\data\backup\_plain\message__message_0.db`)
	defer db.Close()

	sum := md5.Sum([]byte("e595400290"))
	table := "Msg_" + hex.EncodeToString(sum[:])

	var content []byte
	db.QueryRow("SELECT message_content FROM " + table + " WHERE local_type=3 LIMIT 1").Scan(&content)
	fmt.Println("content 长度:", len(content))

	// 解压
	dec, _ := zstd.NewReader(nil)
	defer dec.Close()
	dst := make([]byte, 0, len(content)*10)
	out, err := dec.DecodeAll(content, dst)
	fmt.Println("解压后长度:", len(out), "错误:", err)
	if len(out) > 0 {
		fmt.Println("前 200 字:")
		fmt.Println(string(out[:200]))
	}
}
