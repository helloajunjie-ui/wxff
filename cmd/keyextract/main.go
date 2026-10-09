// keyextract 从 Weixin.exe 内存提取 SQLCipher 4 db key。
//
// 用法:
//
//	keyextract -root C:\...\db_storage -o keys.txt
//
// 加新扫描策略：在 internal/dbkey/ 下新写一个 scanner 文件，实现 dbkey.Scanner
// 接口，然后在下方 scanners 切片里注册即可。
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/nicols/wxff/internal/dbkey"
	"github.com/nicols/wxff/internal/procwin"
)

func main() {
	dataRoot := flag.String("root",
		`C:\Users\Administrator\xwechat_files\wxid_mpvlokr4ycnu22_dfd6\db_storage`,
		"db_storage 目录")
	outFile := flag.String("o", "", "结果输出文件 (rel\\tsalt\\tenc_key 每行一条)")
	window := flag.Int("window", 64*1024, "salt-neighborhood 搜索窗口字节")
	flag.Parse()

	// 1. 收集 db
	dbs, saltIdx, err := dbkey.Collect(*dataRoot)
	if err != nil {
		fmt.Fprintln(os.Stderr, "[错误] 收集 db:", err)
		os.Exit(1)
	}
	fmt.Printf("[OK] 收集到 %d 个 db，%d 个不同 salt\n", len(dbs), len(saltIdx))

	// 2. 开进程
	pid, err := procwin.FindMainWeixinPID()
	if err != nil {
		fmt.Fprintln(os.Stderr, "[错误]", err)
		os.Exit(2)
	}
	proc, err := procwin.Open(pid)
	if err != nil {
		fmt.Fprintln(os.Stderr, "[错误]", err)
		os.Exit(3)
	}
	defer proc.Close()
	fmt.Printf("[OK] Weixin PID=%d\n", pid)

	// 3. 注册 scanner（要加新策略就在这里加）
	scanners := dbkey.NewRegistry(
		&dbkey.SaltNeighborhoodScanner{Window: *window},
		&dbkey.XLiteralScanner{},
	)

	matcher := dbkey.DefaultMatcher()
	remaining := map[string]struct{}{}
	for s := range saltIdx {
		remaining[s] = struct{}{}
	}
	found := make(chan dbkey.Hit, 16)
	done := make(chan struct{})
	var hits []dbkey.Hit
	go func() {
		for h := range found {
			hits = append(hits, h)
			fmt.Printf("[命中] %-40s method=%s key=%x\n", h.DBRel, h.Method, h.EncKey[:8])
		}
		close(done)
	}()

	// 4. 遍历内存区域，喂给所有 scanner
	regions, _ := proc.Regions()
	buf := make([]byte, 8*1024*1024)
	scanned := 0
	for _, r := range regions {
		if len(remaining) == 0 {
			break
		}
		if int(r.Size) > len(buf) {
			buf = make([]byte, r.Size)
		}
		n, _ := proc.Read(r.Addr, buf[:r.Size])
		if n == 0 {
			continue
		}
		view := buf[:n]
		scanned += n
		for _, sc := range scanners.Scanners() {
			if len(remaining) == 0 {
				break
			}
			sc.Scan(view, dbs, saltIdx, matcher, found, remaining)
		}
	}
	close(found)
	<-done

	fmt.Printf("\n[完成] 扫描 %d 区域, %d MB, 命中 %d/%d 个 db\n",
		len(regions), scanned/1024/1024, len(hits), len(dbs))

	// 输出
	var sb strings.Builder
	for _, h := range hits {
		sb.WriteString(fmt.Sprintf("%s\t%s\t%x\n", h.DBRel, h.Salt, h.EncKey))
	}
	if *outFile != "" {
		os.WriteFile(*outFile, []byte(sb.String()), 0644)
		fmt.Printf("[OK] 写入 %s\n", *outFile)
	}
}
