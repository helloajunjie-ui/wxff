package dbkey

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// DB 表示一个待解密的微信 db 文件。
type DB struct {
	Path  string // 绝对路径
	Rel   string // 相对数据根目录的路径（用于展示）
	Page1 []byte // 前 4096 字节
	Salt  []byte // page1[:16]
}

// Collect 遍历 root（db_storage 目录），收集所有 .db 文件（跳过 -wal/-shm）。
// 返回 db 列表和 salt→[]dbIndex 索引。
func Collect(root string) ([]DB, map[string][]int, error) {
	if root == "" {
		return nil, nil, fmt.Errorf("dbkey.Collect: empty root")
	}
	st, err := os.Stat(root)
	if err != nil {
		return nil, nil, fmt.Errorf("dbkey.Collect: %w", err)
	}
	if !st.IsDir() {
		return nil, nil, fmt.Errorf("dbkey.Collect: %s 不是目录", root)
	}

	var dbs []DB
	saltIdx := map[string][]int{}

	err = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		name := d.Name()
		if !strings.HasSuffix(name, ".db") {
			return nil
		}
		if strings.HasSuffix(name, "-wal") || strings.HasSuffix(name, "-shm") {
			return nil
		}
		f, err := os.Open(path)
		if err != nil {
			return nil
		}
		defer f.Close()
		buf := make([]byte, PageSize)
		n, err := f.Read(buf)
		if err != nil || n < PageSize {
			return nil
		}
		rel, _ := filepath.Rel(root, path)
		salt := buf[:SaltSize]
		dbs = append(dbs, DB{Path: path, Rel: rel, Page1: buf, Salt: salt})
		saltIdx[string(salt)] = append(saltIdx[string(salt)], len(dbs)-1)
		return nil
	})
	return dbs, saltIdx, err
}
