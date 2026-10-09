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
	db, _ := sql.Open("sqlite", `F:\wxff\data\backup\_plain\message__message_0.db`)
	defer db.Close()

	sum := md5.Sum([]byte("e595400290"))
	table := "Msg_" + hex.EncodeToString(sum[:])

	// 读 3 条图片消息，提取 md5
	rows, _ := db.Query("SELECT message_content FROM " + table + " WHERE local_type=3 LIMIT 3")
	defer rows.Close()

	md5Re := regexp.MustCompile(`md5="([^"]+)"`)
	var imgMd5s []string
	for rows.Next() {
		var content []byte
		rows.Scan(&content)
		text := string(content)
		// 检查 zstd
		if len(text) >= 4 && text[0] == 0x28 && text[1] == 0xb5 && text[2] == 0x2f && text[3] == 0xfd {
			fmt.Println("图片消息是 zstd 压缩的！需要解压")
			continue
		}
		m := md5Re.FindStringSubmatch(text)
		if len(m) > 1 {
			imgMd5s = append(imgMd5s, m[1])
			fmt.Println("图片 md5:", m[1])
		}
	}

	// 看 cache 里的文件名
	fmt.Println("\ncache 里的文件名:")
	cacheRoot := `C:\Users\Administrator\xwechat_files\wxid_mpvlokr4ycnu22_dfd6\cache`
	filepath.Walk(cacheRoot, func(p string, info os.FileInfo, err error) error {
		if err != nil || info == nil || info.IsDir() {
			return nil
		}
		name := info.Name()
		if strings.HasSuffix(name, ".dat") {
			base := strings.TrimSuffix(name, ".dat")
			fmt.Println(" ", base)
		}
		return nil
	})
}
