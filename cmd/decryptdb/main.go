// decryptdb 解密单个 db 文件。
//
// 用法: decryptdb -in <encrypted.db> -out <plain.db> -key <64hex>
package main

import (
	"database/sql"
	"encoding/hex"
	"flag"
	"fmt"
	"os"

	"github.com/nicols/wxff/internal/decryptedb"
	_ "modernc.org/sqlite" // 纯 Go SQLite 驱动
)

func main() {
	in := flag.String("in", "", "加密 db 路径")
	out := flag.String("out", "", "明文 db 输出路径")
	keyHex := flag.String("key", "", "32 字节 enc_key (64 hex)")
	dryRun := flag.Bool("tables", false, "解密后列出表和行数")
	flag.Parse()

	if *in == "" || *out == "" || *keyHex == "" {
		flag.Usage()
		os.Exit(1)
	}
	key, err := hex.DecodeString(*keyHex)
	if err != nil {
		fmt.Fprintln(os.Stderr, "[错误] key hex:", err)
		os.Exit(2)
	}
	if err := decryptedb.DecryptFile(*in, *out, key); err != nil {
		fmt.Fprintln(os.Stderr, "[错误] 解密:", err)
		os.Exit(3)
	}
	fmt.Printf("[OK] 解密完成 → %s\n", *out)

	if *dryRun {
		db, err := sql.Open("sqlite", *out)
		if err != nil {
			fmt.Fprintln(os.Stderr, "[错误] 打开明文 db:", err)
			os.Exit(4)
		}
		defer db.Close()
		rows, err := db.Query("SELECT name FROM sqlite_master WHERE type='table' ORDER BY name")
		if err != nil {
			fmt.Fprintln(os.Stderr, "[错误] 查询表:", err)
			os.Exit(5)
		}
		defer rows.Close()
		fmt.Println("\n=== 表 ===")
		for rows.Next() {
			var name string
			rows.Scan(&name)
			var cnt int
			db.QueryRow("SELECT count(*) FROM " + name).Scan(&cnt)
			fmt.Printf("  %-40s %d 行\n", name, cnt)
		}
	}
}
