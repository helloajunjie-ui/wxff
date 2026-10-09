package main

import (
	"encoding/hex"
	"fmt"
	"os"
)

func main() {
	files := []string{
		`C:\Users\Administrator\xwechat_files\wxid_mpvlokr4ycnu22_dfd6\msg\attach\25cd413cef8aed1f6591d353a93555ad\2026-10\Img\1bca044dd4290edcd1aa2e2fafea2ff5.dat`,
		`C:\Users\Administrator\xwechat_files\wxid_mpvlokr4ycnu22_dfd6\msg\attach\25cd413cef8aed1f6591d353a93555ad\2026-10\Img\1bca044dd4290edcd1aa2e2fafea2ff5_t.dat`,
	}
	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			fmt.Println("err:", err)
			continue
		}
		fmt.Printf("=== %s (%d bytes) ===\n", f, len(data))
		fmt.Printf("  前 32 字节: %s\n", hex.EncodeToString(data[:32]))
		if len(data) > 16 {
			fmt.Printf("  ASCII: %q\n", data[:32])
		}
	}
	os.Exit(0)
}
