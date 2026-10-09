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

	// 列名
	rows, _ := db.Query("PRAGMA table_info(contact)")
	fmt.Println("=== contact 列 ===")
	for rows.Next() {
		var cid int
		var name, ctype string
		var notnull, pk int
		var dflt sql.NullString
		rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk)
		fmt.Printf("  %2d. %-30s %s\n", cid, name, ctype)
	}
	rows.Close()

	// 看一条样例
	fmt.Println("\n=== 样例好友（前 5 个非空 nickname）===")
	rows, err := db.Query(`SELECT * FROM contact LIMIT 5`)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	cols, _ := rows.Columns()
	for rows.Next() {
		vals := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		rows.Scan(ptrs...)
		for i, c := range cols {
			b, _ := vals[i].([]byte)
			s := string(b)
			if len(s) > 60 {
				s = s[:60] + "..."
			}
			if s != "" && s != "<nil>" {
				fmt.Printf("  %-25s = %s\n", c, s)
			}
		}
		fmt.Println("  ---")
	}
}
