package main

import (
	"fmt"
	"os"
	"github.com/nicols/wxff/internal/imgkey"
)

func main() {
	// 找一张 Sns 图片
	path := `C:\Users\Administrator\xwechat_files\wxid_mpvlokr4ycnu22_dfd6\cache\2025-08\Sns\Img\07\03a5c326b85cf1712676fd29d14ba5`
	data, _ := os.ReadFile(path)
	fmt.Println("文件大小:", len(data))
	fmt.Println("文件头:", data[:15])

	// 用聊天的 image key 解密
	imageKey := "b30e5d7754486daf"
	xorKey := 0xE6
	dec, err := imgkey.DecryptDAT(data, imageKey, xorKey)
	if err != nil {
		fmt.Println("解密失败:", err)
		return
	}
	fmt.Println("解密后大小:", len(dec))
	fmt.Println("解密后文件头:", dec[:10])
	os.WriteFile(`F:\wxff\test_sns_decrypted.jpg`, dec, 0644)
	fmt.Println("已保存到 F:\\wxff\\test_sns_decrypted.jpg")
}
