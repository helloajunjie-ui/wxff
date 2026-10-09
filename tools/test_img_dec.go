package main

import (
	"fmt"
	"os"

	"github.com/nicols/wxff/internal/imgkey"
)

func main() {
	// 找一张聊天图片，解密看看
	path := `C:\Users\Administrator\xwechat_files\wxid_mpvlokr4ycnu22_dfd6\cache\2025-11\Message\9844714273a84709986a483c0ad1b50d\007b03196106f599c598affcde29204e_b.dat`
	data, _ := os.ReadFile(path)
	fmt.Println("原始大小:", len(data))

	imageKey := "b30e5d7754486daf"
	xorKey := 0xE6
	dec, err := imgkey.DecryptDAT(data, imageKey, xorKey)
	if err != nil {
		fmt.Println("解密失败:", err)
		return
	}
	fmt.Println("解密后大小:", len(dec))
	fmt.Println("前 10 字节:", fmt.Sprintf("% X", dec[:10]))

	// 保存到文件
	os.WriteFile(`F:\wxff\test_decrypted.jpg`, dec, 0644)
	fmt.Println("已保存到 F:\\wxff\\test_decrypted.jpg")
}
