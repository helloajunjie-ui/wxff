package main

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

func main() {
	db, err := sql.Open("sqlite", `F:\wxff\data\decrypted\media_0.db`)
	if err != nil {
		fmt.Println("open:", err)
		return
	}
	defer db.Close()

	for _, t := range []string{"VoiceInfo", "Name2Id", "TimeStamp"} {
		fmt.Printf("\n========== %s ==========\n", t)
		sr, _ := db.Query("SELECT sql FROM sqlite_master WHERE name=?", t)
		for sr.Next() {
			var s string
			sr.Scan(&s)
			fmt.Println("SCHEMA:", s)
		}
		sr.Close()

		cr, err := db.Query("SELECT * FROM " + t + " LIMIT 5")
		if err != nil {
			fmt.Println("query:", err)
			continue
		}
		cols, _ := cr.Columns()
		fmt.Println("列:", cols)
		for cr.Next() {
			vals := make([]any, len(cols))
			ptrs := make([]any, len(cols))
			for i := range vals {
				ptrs[i] = &vals[i]
			}
			cr.Scan(ptrs...)
			for i, c := range cols {
				v := vals[i]
				if b, ok := v.([]byte); ok {
					if len(b) > 80 {
						v = fmt.Sprintf("<%d bytes: %x...>", len(b), b[:40])
					} else {
						v = fmt.Sprintf("%q", string(b))
					}
				}
				fmt.Printf("   %s = %v\n", c, v)
			}
			fmt.Println("   ---")
		}
		cr.Close()
	}
}
