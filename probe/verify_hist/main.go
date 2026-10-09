package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/nicols/wxff/internal/imgkey"
)

func main() {
	root := `C:\Users\Administrator\xwechat_files\wxid_mpvlokr4ycnu22_dfd6\msg\attach`
	imageKey := "b30e5d7754486daf"
	xorKey := 0xE6

	total := 0
	v4 := 0
	ok := 0
	filepath.Walk(root, func(p string, info os.FileInfo, err error) error {
		if err != nil || info == nil || info.IsDir() {
			return nil
		}
		if info.Size() < 20 {
			return nil
		}
		total++
		data, err := os.ReadFile(p)
		if err != nil {
			return nil
		}
		if !imgkey.IsWechatV4DAT(data) {
			return nil
		}
		v4++
		dec, err := imgkey.DecryptDAT(data, imageKey, xorKey)
		if err != nil || len(dec) < 4 {
			return nil
		}
		if dec[0] == 0xFF && dec[1] == 0xD8 && dec[2] == 0xFF {
			ok++
			if ok <= 5 {
				rel, _ := filepath.Rel(root, p)
				fmt.Printf("  ✓ JPEG %d字节  %s\n", len(dec), rel)
				out := fmt.Sprintf("F:\\wxff\\data\\proof\\hist_%d.jpg", ok)
				os.WriteFile(out, dec, 0644)
			}
		}
		return nil
	})
	fmt.Printf("\n总文件 %d, V4格式 %d, 解密成功 %d\n", total, v4, ok)
}
