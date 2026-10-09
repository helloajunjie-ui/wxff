package main

import (
	"database/sql"
	"fmt"
	"os"

	_ "modernc.org/sqlite"
)

func main() {
	db, _ := sql.Open("sqlite", `F:\wxff\data\decrypted\contact_plain.db`)
	defer db.Close()

	// 1. 查 guyue119
	fmt.Println("=== guyue119 ===")
	var wxid, nick, remark, alias, head string
	err := db.QueryRow(`SELECT username, nick_name, remark, alias, small_head_url
		FROM contact WHERE username=?`, "guyue119").
		Scan(&wxid, &nick, &remark, &alias, &head)
	if err != nil {
		fmt.Println("  not found:", err)
	} else {
		fmt.Printf("  wxid: %s\n  昵称: %s\n  备注: %s\n  微信号: %s\n", wxid, nick, remark, alias)
	}

	// 2. 列前 20 个好友（备注或昵称非空）
	fmt.Println("\n=== 你的好友列表（前 20）===")
	rows, _ := db.Query(`SELECT username, nick_name, remark, alias FROM contact
		WHERE local_type=0 AND (nick_name != '' OR remark != '')
		LIMIT 20`)
	defer rows.Close()
	for rows.Next() {
		var u, n, r, a string
		rows.Scan(&u, &n, &r, &a)
		show := n
		if r != "" {
			show = r + " (" + n + ")"
		}
		fmt.Printf("  %-30s %s\n", u, show)
	}
	os.Exit(0)
}
