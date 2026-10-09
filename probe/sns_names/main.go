// sns_names 检查朋友圈发布者在 contact.db 里到底有没有昵称。
package main

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

func main() {
	cdb, err := sql.Open("sqlite", "data/decrypted/contact_plain.db")
	if err != nil {
		panic(err)
	}
	defer cdb.Close()

	sdb, err := sql.Open("sqlite", "data/decrypted/sns_plain.db")
	if err != nil {
		panic(err)
	}
	defer sdb.Close()

	// 列出 contact 表结构
	fmt.Println("=== contact 表结构 ===")
	cols, err := cdb.Query("PRAGMA table_info(contact)")
	if err == nil {
		for cols.Next() {
			var cid int
			var name, typ string
			var notnull, pk int
			var dflt sql.NullString
			cols.Scan(&cid, &name, &typ, &notnull, &dflt, &pk)
			fmt.Printf("  %s (%s)\n", name, typ)
		}
		cols.Close()
	}

	// 取所有发布者
	rows, err := sdb.Query("SELECT user_name, count(*) FROM SnsTimeLine GROUP BY user_name ORDER BY count(*) DESC")
	if err != nil {
		panic(err)
	}
	type p struct {
		wxid string
		n    int
	}
	var list []p
	for rows.Next() {
		var x string
		var n int
		rows.Scan(&x, &n)
		list = append(list, p{x, n})
	}
	rows.Close()

	fmt.Printf("\n=== 发布者共 %d 人，逐个查 contact ===\n", len(list))
	hit, miss := 0, 0
	for _, it := range list {
		var nick, remark, alias string
		err := cdb.QueryRow("SELECT nick_name, remark, alias FROM contact WHERE username=?", it.wxid).Scan(&nick, &remark, &alias)
		if err == sql.ErrNoRows {
			miss++
			fmt.Printf("  [缺失] %-45s posts=%d\n", it.wxid, it.n)
			continue
		}
		if err != nil {
			fmt.Printf("  [错误] %-45s %v\n", it.wxid, err)
			continue
		}
		hit++
		fmt.Printf("  [命中] %-45s nick=%-20q remark=%-20q alias=%q posts=%d\n", it.wxid, nick, remark, alias, it.n)
	}
	fmt.Printf("\n命中 %d / 缺失 %d\n", hit, miss)
}
