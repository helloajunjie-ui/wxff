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

	wxid := "e595400290"
	sum := md5.Sum([]byte(wxid))
	talkerMd5 := hex.EncodeToString(sum[:])
	fmt.Println("talkerMd5:", talkerMd5)

	root := `C:\Users\Administrator\xwechat_files\wxid_mpvlokr4ycnu22_dfd6\cache`
	count := 0
	decCount := 0
	filepath.Walk(root, func(p string, info os.FileInfo, err error) error {
		if err != nil || info == nil || info.IsDir() {
			return nil
		}
		if !strings.Contains(p, talkerMd5) {
			return nil
		}
		if !strings.Contains(p, `\Message\`) {
			return nil
		}
		count++
		data, err := os.ReadFile(p)
		if err != nil {
			return nil
		}
		if !imgkey.IsWechatV4DAT(data) {
			return nil
		}
		decImg, err := imgkey.DecryptDAT(data, imageKey, xorKey)
		if err != nil {
			return nil
		}
		if len(decImg) > 10000 {
			decCount++
			h := md5.Sum(decImg)
			fmt.Println("解密成功:", filepath.Base(p), "md5:", hex.EncodeToString(h[:]))
		}
		return nil
	})
	fmt.Println("总文件数:", count, "解密成功:", decCount)
}
