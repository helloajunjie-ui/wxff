package main

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/nicols/wxff/internal/imgkey"
)

func main() {
	imageKey := "b30e5d7754486daf"
	xorKey := 0xE6

	// 收集 cache 里所有图片，解密后算 md5，建索引
	root := `C:\Users\Administrator\xwechat_files\wxid_mpvlokr4ycnu22_dfd6\cache`
	imgMd5Map := map[string]string{} // md5(decrypted) -> path
	count := 0
	filepath.Walk(root, func(p string, info os.FileInfo, err error) error {
		if err != nil || info == nil || info.IsDir() {
			return nil
		}
		name := info.Name()
		if !strings.HasSuffix(name, ".dat") {
			return nil
		}
		// 只要 Message 目录的（聊天图片）
		if !strings.Contains(p, `\Message\`) {
			return nil
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return nil
		}
		if !imgkey.IsWechatV4DAT(data) {
			return nil
		}
		dec, err := imgkey.DecryptDAT(data, imageKey, xorKey)
		if err != nil {
			return nil
		}
		h := md5.Sum(dec)
		md5str := hex.EncodeToString(h[:])
		// 只存大图（大于 10KB）
		if len(dec) > 10000 {
			imgMd5Map[md5str] = p
			count++
		}
		return nil
	})
	fmt.Println("聊天图片索引:", count, "张")

	// 看看有没有测试用的 md5
	// 从之前的 XML 里找一个 md5
	testMd5 := "test"
	fmt.Println("测试 md5:", testMd5, "是否存在:", imgMd5Map[testMd5] != "")
}
