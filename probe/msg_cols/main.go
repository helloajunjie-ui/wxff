package main

import (
	"database/sql"
	"fmt"
	"os"

	_ "modernc.org/sqlite"
)

func main() {
	db, _ := sql.Open("sqlite", `F:\wxff\data\decrypted\message_0.db`)
	defer db.Close()

	// 找消息最多的那张表（13430 条那个）
	const t = "Msg_c41483b0a7c9b84d7237e00b19256971"

	// 列
	rows, _ := db.Query("PRAGMA table_info(" + t + ")")
	fmt.Printf("=== %s 列 ===\n", t)
	cols := []string{}
	for rows.Next() {
		var cid int
		var n, ty string
		var nn, pk int
		var d sql.NullString
		rows.Scan(&cid, &n, &ty, &nn, &d, &pk)
		cols = append(cols, n)
		fmt.Printf("  %2d. %-30s %s\n", cid, n, ty)
	}
	rows.Close()

	// 取 3 条最新消息
	fmt.Println("\n=== 最新 3 条消息 ===")
	rows, err := db.Query("SELECT * FROM " + t + " ORDER BY create_time DESC LIMIT 3")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer rows.Close()
	for rows.Next() {
		vals := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		rows.Scan(ptrs...)
		fmt.Println("  ---")
		for i, c := range cols {
			b, _ := vals[i].([]byte)
			s := string(b)
			if len(s) > 100 {
				s = s[:100] + "..."
			}
			fmt.Printf("    %-25s = %q\n", c, s)
		}
	}
	os.Exit(0)
}
