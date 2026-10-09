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

	// Name2Id 表结构
	fmt.Println("=== Name2Id 列 ===")
	rows, _ := db.Query("PRAGMA table_info(Name2Id)")
	for rows.Next() {
		var cid int
		var n, t string
		var nn, pk int
		var d sql.NullString
		rows.Scan(&cid, &n, &t, &nn, &d, &pk)
		fmt.Printf("  %d. %s %s\n", cid, n, t)
	}
	rows.Close()

	// 查 guyue119 对应的 md5
	fmt.Println("\n=== guyue119 的 md5 映射 ===")
	var md5 string
	err := db.QueryRow("SELECT talker_id FROM Name2Id WHERE username=?", "guyue119").Scan(&md5)
	if err != nil {
		fmt.Println("  查询失败:", err)
	} else {
		fmt.Printf("  guyue119 → %s\n", md5)
	}

	// 列出所有 Name2Id
	fmt.Println("\n=== Name2Id 全部 ===")
	rows, err = db.Query("SELECT * FROM Name2Id")
	if err != nil {
		fmt.Println("  err:", err)
		os.Exit(1)
	}
	cols, _ := rows.Columns()
	fmt.Println("  列:", cols)
	for rows.Next() {
		vals := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		rows.Scan(ptrs...)
		line := ""
		for i, c := range cols {
			b, _ := vals[i].([]byte)
			line += fmt.Sprintf("%s=%s ", c, string(b))
		}
		fmt.Println("  " + line)
	}
	rows.Close()
}
