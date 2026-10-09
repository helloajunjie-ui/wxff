package main

import (
	"bytes"
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

	fmt.Println("原始长度:", len(content))
	fmt.Println("前 4 字节:", fmt.Sprintf("% X", content[:4]))

	// 方式1：DecodeAll 传预分配 buffer
	dec, _ := zstd.NewReader(nil)
	defer dec.Close()
	dst := make([]byte, 0, len(content)*10)
	out, err := dec.DecodeAll(content, dst)
	fmt.Println("DecodeAll 结果: 长度=", len(out), "错误=", err)
	if len(out) > 0 {
		fmt.Println("前 100 字:", string(out[:100]))
	}

	// 方式2：用 Reader
	dec2, _ := zstd.NewReader(bytes.NewReader(content))
	defer dec2.Close()
	buf := &bytes.Buffer{}
	n, err := buf.ReadFrom(dec2)
	fmt.Println("Reader 结果: 长度=", n, "错误=", err)
	if n > 0 {
		fmt.Println("前 100 字:", buf.String()[:100])
	}
}
