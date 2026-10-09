// sns_thumb 对比 XML 中记录的图片尺寸（原图）与缓存实际尺寸，判断是否混入缩略图。
package main

import (
	"database/sql"
	"fmt"
	"regexp"
	"sort"

	_ "modernc.org/sqlite"
)

func main() {
	db, err := sql.Open("sqlite", `data\decrypted\sns_plain.db`)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	rows, err := db.Query(`SELECT content FROM SnsTimeLine`)
	if err != nil {
		panic(err)
	}
	defer rows.Close()

	// XML 里 <size width=".." height=".."/> 是原图尺寸
	sizeRe := regexp.MustCompile(`<size width="(\d+)" height="(\d+)"`)
	type dim struct{ w, h int }
	xmlDims := map[dim]int{}
	total := 0
	for rows.Next() {
		var content string
		rows.Scan(&content)
		for _, m := range sizeRe.FindAllStringSubmatch(content, -1) {
			var w, h int
			fmt.Sscanf(m[1], "%d", &w)
			fmt.Sscanf(m[2], "%d", &h)
			xmlDims[dim{w, h}]++
			total++
		}
	}

	fmt.Printf("XML 中记录的图片尺寸条目总数: %d，去重尺寸种类: %d\n", total, len(xmlDims))

	type kv struct {
		d dim
		n int
	}
	var list []kv
	for d, n := range xmlDims {
		list = append(list, kv{d, n})
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].d.w*list[i].d.h > list[j].d.w*list[j].d.h
	})

	fmt.Println("\n=== XML 记录的原图尺寸（按面积降序，前 30）===")
	for i, e := range list {
		if i >= 30 {
			break
		}
		fmt.Printf("  %5d x %-5d  x%d\n", e.d.w, e.d.h, e.n)
	}

	// 分档
	var small, mid, big, huge int
	for d, n := range xmlDims {
		area := d.w * d.h
		switch {
		case area < 500*500:
			small += n
		case area < 1500*1500:
			mid += n
		case area < 3000*3000:
			big += n
		default:
			huge += n
		}
	}
	fmt.Printf("\n=== XML 原图尺寸分档 ===\n")
	fmt.Printf("  <500x500: %d\n", small)
	fmt.Printf("  500x500~1500x1500: %d\n", mid)
	fmt.Printf("  1500x1500~3000x3000: %d\n", big)
	fmt.Printf("  >=3000x3000: %d\n", huge)
}
