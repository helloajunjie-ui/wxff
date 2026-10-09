package main

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

func main() {
	db, err := sql.Open("sqlite", `F:\wxff\data\decrypted\contact__contact.db`)
	if err != nil {
		fmt.Println("open:", err)
		return
	}
	defer db.Close()

	for _, wxid := range []string{"wxid_mpvlokr4ycnu22", "wxid_mpvlokr4ycnu22_dfd6"} {
		var nick, remark, alias string
		err := db.QueryRow("SELECT nick_name, remark, alias FROM contact WHERE username=?", wxid).Scan(&nick, &remark, &alias)
		fmt.Printf("wxid=%s  err=%v  nick=%q remark=%q alias=%q\n", wxid, err, nick, remark, alias)
	}

	// 兜底：看 contact 表里有没有 local_type 表示「自己」的行
	rows, err := db.Query("SELECT username, nick_name, remark, local_type FROM contact WHERE username LIKE '%mpvlokr4ycnu22%'")
	if err != nil {
		fmt.Println("query:", err)
		return
	}
	defer rows.Close()
	fmt.Println("--- LIKE 匹配 ---")
	for rows.Next() {
		var u, n, r string
		var lt int
		rows.Scan(&u, &n, &r, &lt)
		fmt.Printf("  username=%q nick=%q remark=%q local_type=%d\n", u, n, r, lt)
	}
}
