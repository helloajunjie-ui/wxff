// cache_id 测试朋友圈缓存 30-hex id 与 XML md5/key 的关系。
package main

import (
	"crypto/md5"
	"database/sql"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	_ "modernc.org/sqlite"
)

func main() {
	cacheRoot := `C:\Users\Administrator\xwechat_files\wxid_mpvlokr4ycnu22_dfd6\cache`
	dbPath := `F:\wxff\data\backup\_plain\sns__sns.db`

	// 1. 收集所有缓存 30-hex id
	cacheIDs := map[string]string{} // id -> path
	filepath.Walk(cacheRoot, func(p string, info os.FileInfo, err error) error {
		if err != nil || info == nil || info.IsDir() {
			return nil
		}
		if !strings.Contains(p, `\Sns\Img\`) {
			return nil
		}
		stem := filepath.Base(p)
		if i := strings.LastIndex(stem, "."); i > 0 {
			stem = stem[:i]
		}
		if len(stem) == 30 {
			cacheIDs[strings.ToLower(stem)] = p
		}
		return nil
	})
	fmt.Printf("缓存 30-hex id 总数: %d\n", len(cacheIDs))

	// 2. 从 sns.db 提取所有 md5 与 key
	db, _ := sql.Open("sqlite", dbPath)
	defer db.Close()
	rows, _ := db.Query("SELECT content FROM SnsTimeLine")
	md5Re := regexp.MustCompile(`md5="([0-9a-f]{32})"`)
	keyRe := regexp.MustCompile(`key="(\d+)"`)
	md5Set := map[string]bool{}
	keySet := map[string]bool{}
	for rows.Next() {
		var c string
		rows.Scan(&c)
		for _, m := range md5Re.FindAllStringSubmatch(c, -1) {
			md5Set[m[1]] = true
		}
		for _, m := range keyRe.FindAllStringSubmatch(c, -1) {
			keySet[m[1]] = true
		}
	}
	rows.Close()
	fmt.Printf("XML 中 md5 总数: %d, key 总数: %d\n", len(md5Set), len(keySet))

	// 3. 测试各种变换：md5 -> 30hex?
	fmt.Println("\n=== 测试 md5 变换是否命中缓存 id ===")
	hit := 0
	for m := range md5Set {
		// 直接前30
		if _, ok := cacheIDs[m[:30]]; ok {
			hit++
			continue
		}
		// md5(md5)
		h := md5.Sum([]byte(m))
		hh := hex.EncodeToString(h[:])
		if _, ok := cacheIDs[hh[:30]]; ok {
			hit++
			continue
		}
		// md5(md5) 后30
		if _, ok := cacheIDs[hh[2:]]; ok {
			hit++
			continue
		}
	}
	fmt.Printf("md5 变换命中: %d / %d\n", hit, len(md5Set))

	// 4. 测试 key 变换
	fmt.Println("\n=== 测试 key 变换是否命中缓存 id ===")
	hit2 := 0
	for k := range keySet {
		h := md5.Sum([]byte(k))
		hh := hex.EncodeToString(h[:])
		if _, ok := cacheIDs[hh[:30]]; ok {
			hit2++
			continue
		}
		if _, ok := cacheIDs[hh[2:]]; ok {
			hit2++
			continue
		}
	}
	fmt.Printf("key 变换命中: %d / %d\n", hit2, len(keySet))

	// 5. 打印几个缓存 id 与几个 md5 供人工比对
	fmt.Println("\n=== 缓存 id 样本 ===")
	n := 0
	for id := range cacheIDs {
		fmt.Println("  ", id)
		n++
		if n >= 5 {
			break
		}
	}
	fmt.Println("=== md5 样本 ===")
	n = 0
	for m := range md5Set {
		fmt.Println("  ", m)
		n++
		if n >= 5 {
			break
		}
	}
}
