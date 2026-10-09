package main

import (
	"bytes"
	"crypto/md5"
	"database/sql"
	"encoding/hex"
	"fmt"
	"io"

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

	dec, _ := zstd.NewReader(bytes.NewReader(content))
	defer dec.Close()
	out, err := io.ReadAll(dec)
	if err != nil {
		fmt.Println("解压失败:", err)
		return
	}
	fmt.Println("解压后长度:", len(out))
	fmt.Println("解压后内容:")
	fmt.Println(string(out[:min(500, len(out))]))
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
