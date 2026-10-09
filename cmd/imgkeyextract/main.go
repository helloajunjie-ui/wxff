// imgkeyextract 从 Weixin.exe 内存提取图片 AES key。
//
// 用法: imgkeyextract -attach <msg\attach 目录>
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/nicols/wxff/internal/imgkey"
	"github.com/nicols/wxff/internal/procwin"
)

func main() {
	attachRoot := flag.String("attach",
		`C:\Users\Administrator\xwechat_files\wxid_mpvlokr4ycnu22_dfd6\msg\attach`,
		"msg/attach 目录")
	flag.Parse()

	// 1. 找模板
	tplPath, encBlock, xorKey, err := imgkey.FindTemplate(*attachRoot)
	if err != nil {
		fmt.Println("[!] 找模板失败:", err)
		os.Exit(1)
	}
	fmt.Printf("[OK] 模板: %s\n", tplPath)
	fmt.Printf("[OK] XOR key: ")
	if xorKey != nil {
		fmt.Printf("0x%02X\n", *xorKey)
	} else {
		fmt.Println("(未推出)")
	}

	// 2. 开进程
	pid, err := procwin.FindMainWeixinPID()
	if err != nil {
		fmt.Println("[!]", err)
		os.Exit(2)
	}
	proc, err := procwin.Open(pid)
	if err != nil {
		fmt.Println("[!]", err)
		os.Exit(3)
	}
	defer proc.Close()
	fmt.Printf("[OK] Weixin PID=%d, 开始扫内存...\n", pid)

	regions, _ := proc.Regions()
	buf := make([]byte, 8*1024*1024)
	found := ""
	for _, r := range regions {
		if int(r.Size) > len(buf) {
			buf = make([]byte, r.Size)
		}
		n, _ := proc.Read(r.Addr, buf[:r.Size])
		if n == 0 {
			continue
		}
		view := buf[:n]
		// 找所有 16 位连续 ASCII 字母数字串
		for i := 0; i < len(view); {
			if !isAlnum(view[i]) {
				i++
				continue
			}
			start := i
			for i < len(view) && isAlnum(view[i]) {
				i++
			}
			run := view[start:i]
			// 每个 offset 都试 16 字节窗口
			for off := 0; off+16 <= len(run); off++ {
				cand := run[off : off+16]
				if imgkey.ValidateKey(cand, encBlock) {
					found = string(cand)
					break
				}
			}
			if found != "" {
				break
			}
		}
		if found != "" {
			break
		}
	}

	if found == "" {
		fmt.Println("[!] 没找到 image key。请先在微信里点开几张大图，再重试。")
		os.Exit(4)
	}
	fmt.Printf("\n[命中] image_key = %s\n", found)
	if xorKey != nil {
		fmt.Printf("[命中] xor_key  = 0x%02X\n", *xorKey)
	}

	// 3. 用这个 key 解密模板，验证一下
	data, _ := os.ReadFile(tplPath)
	xk := 0
	if xorKey != nil {
		xk = *xorKey
	}
	dec, err := imgkey.DecryptDAT(data, found, xk)
	if err != nil {
		fmt.Println("[!] 解密失败:", err)
		os.Exit(5)
	}
	out := strings.ReplaceAll(tplPath, ".dat", "_decrypted.bin")
	os.WriteFile(out, dec, 0644)
	fmt.Printf("[OK] 解密样例 → %s (%d 字节)\n", out, len(dec))
	os.Exit(0)
}

func isAlnum(b byte) bool {
	return (b >= '0' && b <= '9') || (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z')
}
