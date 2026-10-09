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

	rows, _ := db.Query("SELECT message_content FROM " + table + " WHERE local_type=3 LIMIT 1")
	defer rows.Close()
	var content []byte
	rows.Scan(&content)

	dec, _ := zstd.NewReader(nil)
	defer dec.Close()
	out, err := dec.DecodeAll(content, nil)
	if err != nil {
		fmt.Println("解压失败:", err)
		return
	}
	fmt.Println("解压后 XML 前 1000 字:")
	fmt.Println(string(out[:1000]))
}
