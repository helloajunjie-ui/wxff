package main

import (
	"crypto/md5"
	"database/sql"
	"encoding/hex"
	"fmt"
	"regexp"

	_ "modernc.org/sqlite"
)

func main() {
	db, _ := sql.Open("sqlite", `F:\wxff\data\backup\_plain\message__message_0.db`)
	defer db.Close()

	// e595400290 的表
	sum := md5.Sum([]byte("e595400290"))
	table := "Msg_" + hex.EncodeToString(sum[:])

	rows, _ := db.Query("SELECT message_content FROM " + table + " WHERE local_type=3 LIMIT 3")
	defer rows.Close()
	i := 0
	for rows.Next() {
		var content []byte
		rows.Scan(&content)
		text := string(content)
		// zstd 解压
		if len(text) >= 4 && text[0] == 0x28 && text[1] == 0xb5 && text[2] == 0x2f && text[3] == 0xfd {
			fmt.Println("图片消息是 zstd 压缩的，需要解压后才能提取 md5")
			i++
			continue
		}
		re := regexp.MustCompile(`md5="([^"]+)"`)
		m := re.FindStringSubmatch(text)
		if len(m) > 1 {
			fmt.Printf("图片 %d: md5=%s\n", i, m[1])
		} else {
			fmt.Printf("图片 %d: 没找到 md5, 前200字: %s\n", i, text[:200])
		}
		i++
	}
}
