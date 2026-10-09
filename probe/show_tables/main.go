package main

import (
	"database/sql"
	"fmt"
	"os"

	_ "modernc.org/sqlite"
)

func must(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "错误:", err)
		os.Exit(1)
	}
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("用法: inspect <plain.db>")
		os.Exit(1)
	}
	db, err := sql.Open("sqlite", os.Args[1])
	must(err)
	defer db.Close()

	rows, err := db.Query("SELECT name FROM sqlite_master WHERE type='table' ORDER BY name")
	must(err)
	type tinfo struct {
		name string
		cnt  int
	}
	var tables []tinfo
	for rows.Next() {
		var n string
		rows.Scan(&n)
		var c int
		db.QueryRow("SELECT count(*) FROM \"" + n + "\"").Scan(&c)
		tables = append(tables, tinfo{n, c})
	}
	rows.Close()
	fmt.Println("=== 表 ===")
	for _, t := range tables {
		fmt.Printf("  %-40s %d\n", t.name, t.cnt)
	}
}
