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

	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&count)
	if err != nil {
		fmt.Println("查询失败:", err)
		return
	}
	fmt.Println("总消息数:", count)

	// 按类型统计
	rows, _ := db.Query("SELECT local_type, COUNT(*) FROM " + table + " GROUP BY local_type")
	defer rows.Close()
	for rows.Next() {
		var t, c int
		rows.Scan(&t, &c)
		fmt.Printf("  type=%d: %d 条\n", t, c)
	}
}
