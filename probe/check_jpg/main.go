package main

import (
	"encoding/hex"
	"fmt"
	"os"
)

func main() {
	data, _ := os.ReadFile(`C:\Users\Administrator\xwechat_files\wxid_mpvlokr4ycnu22_dfd6\msg\attach\25cd413cef8aed1f6591d353a93555ad\2026-10\Img\1bca044dd4290edcd1aa2e2fafea2ff5_t_decrypted.bin`)
	fmt.Printf("大小: %d 字节\n", len(data))
	fmt.Printf("前 16 字节: %s\n", hex.EncodeToString(data[:16]))
	fmt.Printf("前 16 ASCII: %q\n", data[:16])
	// 复制成 .jpg
	os.WriteFile(`F:\wxff\data\proof\decrypted_thumb.jpg`, data, 0644)
	fmt.Println("已保存到 F:\\wxff\\data\\proof\\decrypted_thumb.jpg")
}
