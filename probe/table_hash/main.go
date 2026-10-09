// table_hash 验证 Msg_<hash> 的 hash 到底怎么算。
package main

import (
	"crypto/md5"
	"database/sql"
	"encoding/hex"
	"fmt"
	"strings"

	_ "modernc.org/sqlite"
)

func main() {
	mdb := `F:\wxff\data\decrypted\message__message_0.db`
	db, _ := sql.Open("sqlite", mdb)
	defer db.Close()

	// 1. 列出所有 Msg_ 表
	rows, _ := db.Query("SELECT name FROM sqlite_master WHERE type='table' AND name LIKE 'Msg_%'")
	var tables []string
	for rows.Next() {
		var n string
		rows.Scan(&n)
		tables = append(tables, n)
	}
	rows.Close()
	fmt.Printf("Msg_ 表共 %d 个\n", len(tables))

	// 2. 取 Name2Id 映射
	n2i := map[int64]string{}
	nr, _ := db.Query("SELECT rowid, user_name FROM Name2Id")
	for nr.Next() {
		var id int64
		var un string
		nr.Scan(&id, &un)
		n2i[id] = un
	}
	nr.Close()
	fmt.Printf("Name2Id 共 %d 条\n", len(n2i))

	// 3. 对每个 wxid 算 md5，看是否匹配某个表名
	matched := 0
	for _, un := range n2i {
		sum := md5.Sum([]byte(un))
		h := hex.EncodeToString(sum[:])
		want := "Msg_" + h
		found := false
		for _, t := range tables {
			if t == want {
				found = true
				break
			}
		}
		if found {
			matched++
		} else {
			fmt.Printf("未匹配: %-30s md5=%s\n", un, h)
		}
	}
	fmt.Printf("md5(wxid) 匹配: %d/%d\n", matched, len(n2i))

	// 4. 打印前 5 个表名，看规律
	fmt.Println("\n前 10 个 Msg_ 表名:")
	for i, t := range tables {
		if i >= 10 {
			break
		}
		fmt.Println(" ", t)
	}

	// 5. 检查表名 hash 是否等于 md5(某 wxid) 或 md5(排序后)
	fmt.Println("\n尝试：表名 hash 与 Name2Id 的对应（用 rowid 顺序）")
	for i, t := range tables {
		if i >= 5 {
			break
		}
		h := strings.TrimPrefix(t, "Msg_")
		fmt.Printf("表 %s -> hash=%s\n", t, h)
	}
}
