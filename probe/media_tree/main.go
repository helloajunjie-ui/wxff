package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func main() {
	root := `C:\Users\Administrator\xwechat_files\wxid_mpvlokr4ycnu22_dfd6`

	// 1. 列出 msg 顶层
	fmt.Println("=== msg 顶层 ===")
	entries, err := os.ReadDir(filepath.Join(root, "msg"))
	if err != nil {
		fmt.Println("err:", err)
	} else {
		for _, e := range entries {
			fmt.Printf("  %s  dir=%v\n", e.Name(), e.IsDir())
		}
	}

	// 2. 对每个候选目录，统计文件数 + 扩展名分布 + 前几个样例
	candidates := []string{
		filepath.Join(root, "msg", "video"),
		filepath.Join(root, "msg", "attach"),
		filepath.Join(root, "msg", "file"),
		filepath.Join(root, "msg", "migrate"),
		filepath.Join(root, "msg", "attach", "video"),
		filepath.Join(root, "msg", "attach", "attach"),
	}
	for _, c := range candidates {
		statMedia(c)
	}

	// 3. 全盘扫 msg 下所有扩展名分布（限深度）
	fmt.Println("\n=== msg 下扩展名分布（全递归）===")
	extCount := map[string]int{}
	var total int64
	filepath.Walk(filepath.Join(root, "msg"), func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(p))
		if ext == "" {
			ext = "(无扩展名)"
		}
		extCount[ext]++
		total++
		return nil
	})
	type kv struct {
		k string
		n int
	}
	var list []kv
	for k, n := range extCount {
		list = append(list, kv{k, n})
	}
	sort.Slice(list, func(i, j int) bool { return list[i].n > list[j].n })
	for _, e := range list {
		fmt.Printf("  %-16s %d\n", e.k, e.n)
	}
	fmt.Printf("  合计 %d 个文件\n", total)
}

func statMedia(dir string) {
	fmt.Printf("\n=== %s ===\n", dir)
	entries, err := os.ReadDir(dir)
	if err != nil {
		fmt.Println("  err:", err)
		return
	}
	dirs, files := 0, 0
	var samples []string
	for _, e := range entries {
		if e.IsDir() {
			dirs++
		} else {
			files++
			if len(samples) < 8 {
				info, _ := e.Info()
				sz := int64(0)
				if info != nil {
					sz = info.Size()
				}
				samples = append(samples, fmt.Sprintf("%s (%d B)", e.Name(), sz))
			}
		}
	}
	fmt.Printf("  子目录 %d，文件 %d\n", dirs, files)
	for _, s := range samples {
		fmt.Println("   ", s)
	}
	// 若全是子目录，深入一层看样例
	if files == 0 && dirs > 0 {
		for i, e := range entries {
			if i >= 3 {
				break
			}
			if e.IsDir() {
				sub := filepath.Join(dir, e.Name())
				subEntries, _ := os.ReadDir(sub)
				fmt.Printf("   [%s] 下 %d 项：\n", e.Name(), len(subEntries))
				for j, se := range subEntries {
					if j >= 5 {
						break
					}
					info, _ := se.Info()
					sz := int64(0)
					if info != nil {
						sz = info.Size()
					}
					fmt.Printf("      %s (dir=%v, %d B)\n", se.Name(), se.IsDir(), sz)
				}
			}
		}
	}
}
