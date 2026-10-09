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

	// wxid_i49id3jztuzq12 的 md5
	sum := md5.Sum([]byte("wxid_i49id3jztuzq12"))
	table := "Msg_" + hex.EncodeToString(sum[:])
	fmt.Println("表名:", table)

	rows, _ := db.Query("SELECT message_content FROM " + table + " WHERE local_type=3 LIMIT 3")
	defer rows.Close()

	md5Re := regexp.MustCompile(`md5="([^"]+)"`)
	i := 0
	for rows.Next() {
		var content []byte
		rows.Scan(&content)
		text := string(content)
		// 检查是不是 zstd
		if len(text) >= 4 && text[0] == 0x28 && text[1] == 0xb5 && text[2] == 0x2f && text[3] == 0xfd {
			fmt.Println("图片消息是 zstd 压缩的，需要解压")
			continue
		}
		m := md5Re.FindStringSubmatch(text)
		if len(m) > 1 {
			fmt.Printf("消息 %d: md5=%s\n", i, m[1])
		} else {
			fmt.Printf("消息 %d: 没找到 md5, 前200字: %s\n", i, text[:200])
		}
		i++
	}
}
