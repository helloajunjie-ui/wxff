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

	// 1. 深入 msg\attach\<hash>\<month>\ 看文件
	fmt.Println("=== msg\\attach 深挖（前 3 个 hash 目录）===")
	attachRoot := filepath.Join(root, "msg", "attach")
	hashes, _ := os.ReadDir(attachRoot)
	shown := 0
	for _, h := range hashes {
		if !h.IsDir() || shown >= 3 {
			continue
		}
		shown++
		hd := filepath.Join(attachRoot, h.Name())
		months, _ := os.ReadDir(hd)
		fmt.Printf("\n[%s] %d 个月份目录\n", h.Name(), len(months))
		for _, m := range months {
			md := filepath.Join(hd, m.Name())
			files, _ := os.ReadDir(md)
			fmt.Printf("  %s: %d 项\n", m.Name(), len(files))
			for i, f := range files {
				if i >= 6 {
					break
				}
				info, _ := f.Info()
				sz := int64(0)
				if info != nil {
					sz = info.Size()
				}
				fmt.Printf("     %s (dir=%v, %d B)\n", f.Name(), f.IsDir(), sz)
			}
		}
	}

	// 2. 找所有 .dat 文件，按所在目录归类，看分布
	fmt.Println("\n=== .dat 文件分布（按父目录模式）===")
	patCount := map[string]int{}
	var datSamples []string
	filepath.Walk(filepath.Join(root, "msg"), func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if strings.ToLower(filepath.Ext(p)) != ".dat" {
			return nil
		}
		rel, _ := filepath.Rel(filepath.Join(root, "msg"), p)
		parts := strings.Split(rel, string(filepath.Separator))
		// 用「第1层/第2层」做模式
		pat := ""
		if len(parts) >= 2 {
			pat = parts[0] + "/" + parts[1]
		} else {
			pat = parts[0]
		}
		// 把 32hex 和 年-月 归一化
		pat = normalize(pat)
		patCount[pat]++
		if len(datSamples) < 10 {
			datSamples = append(datSamples, fmt.Sprintf("%s (%d B)", rel, info.Size()))
		}
		return nil
	})
	type kv struct {
		k string
		n int
	}
	var list []kv
	for k, n := range patCount {
		list = append(list, kv{k, n})
	}
	sort.Slice(list, func(i, j int) bool { return list[i].n > list[j].n })
	for _, e := range list {
		fmt.Printf("  %-40s %d\n", e.k, e.n)
	}
	fmt.Println("  样例：")
	for _, s := range datSamples {
		fmt.Println("   ", s)
	}

	// 3. 找 voice/语音相关目录
	fmt.Println("\n=== 含 voice/audio/silk 的路径 ===")
	filepath.Walk(root, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		low := strings.ToLower(p)
		if strings.Contains(low, "voice") || strings.Contains(low, "audio") || strings.Contains(low, "silk") {
			if info.IsDir() {
				fmt.Println("  [dir]", p)
			}
		}
		return nil
	})
}

func normalize(s string) string {
	parts := strings.Split(s, "/")
	for i, p := range parts {
		if isHex32(p) {
			parts[i] = "<32hex>"
		} else if isYearMonth(p) {
			parts[i] = "<年-月>"
		}
	}
	return strings.Join(parts, "/")
}

func isHex32(s string) bool {
	if len(s) != 32 {
		return false
	}
	for _, c := range s {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			return false
		}
	}
	return true
}

func isYearMonth(s string) bool {
	if len(s) != 7 || s[4] != '-' {
		return false
	}
	return true
}
