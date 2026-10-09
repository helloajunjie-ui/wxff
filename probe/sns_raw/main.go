// sns_raw 打印指定发布者某条朋友圈的原始 content XML，用于分析图片标识。
package main

import (
	"database/sql"
	"fmt"
	"os"

	_ "modernc.org/sqlite"
)

func main() {
	dbPath := `F:\wxff\data\backup\_plain\sns__sns.db`
	if len(os.Args) > 1 {
		dbPath = os.Args[1]
	}
	user := "wxid_i49id3jztuzq12"
	if len(os.Args) > 2 {
		user = os.Args[2]
	}
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	rows, err := db.Query("SELECT tid, content FROM SnsTimeLine WHERE user_name=? ORDER BY tid ASC LIMIT 3", user)
	if err != nil {
		panic(err)
	}
	defer rows.Close()
	n := 0
	for rows.Next() {
		var tid int64
		var content string
		rows.Scan(&tid, &content)
		n++
		fmt.Printf("\n========== tid=%d  len=%d ==========\n", tid, len(content))
		fmt.Println(content)
	}
	fmt.Printf("\n共打印 %d 条\n", n)
}
