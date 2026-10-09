// cache_md5 测试：缓存文件解密后的 md5 是否等于 XML 中的 md5 字段。
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

	"github.com/nicols/wxff/internal/imgkey"
	_ "modernc.org/sqlite"
)

func main() {
	cacheRoot := `C:\Users\Administrator\xwechat_files\wxid_mpvlokr4ycnu22_dfd6\cache`
	dbPath := `F:\wxff\data\backup\_plain\sns__sns.db`
	imageKey := "b30e5d7754486daf"
	xorKey := 0xE6

	// 1. 解密所有缓存文件，计算 md5(decrypted) -> path
	decMD5 := map[string]string{}
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
		if len(stem) != 30 {
			return nil
		}
		data, e := os.ReadFile(p)
		if e != nil || !imgkey.IsWechatV4DAT(data) {
			return nil
		}
		dec, e2 := imgkey.DecryptDAT(data, imageKey, xorKey)
		if e2 != nil || len(dec) < 4 {
			return nil
		}
		sum := md5.Sum(dec)
		decMD5[hex.EncodeToString(sum[:])] = p
		return nil
	})
	fmt.Printf("解密后 md5 总数: %d\n", len(decMD5))

	// 2. 从 sns.db 提取 md5
	db, _ := sql.Open("sqlite", dbPath)
	defer db.Close()
	rows, _ := db.Query("SELECT content FROM SnsTimeLine")
	md5Re := regexp.MustCompile(`md5="([0-9a-f]{32})"`)
	md5Set := map[string]bool{}
	for rows.Next() {
		var c string
		rows.Scan(&c)
		for _, m := range md5Re.FindAllStringSubmatch(c, -1) {
			md5Set[m[1]] = true
		}
	}
	rows.Close()
	fmt.Printf("XML md5 总数: %d\n", len(md5Set))

	// 3. 匹配
	hit := 0
	for m := range md5Set {
		if _, ok := decMD5[m]; ok {
			hit++
		}
	}
	fmt.Printf("\n解密后 md5 命中 XML md5: %d / %d\n", hit, len(md5Set))

	// 4. 打印几个解密后 md5 样本
	fmt.Println("\n=== 解密后 md5 样本 ===")
	n := 0
	for m := range decMD5 {
		fmt.Println("  ", m)
		n++
		if n >= 5 {
			break
		}
	}
}
