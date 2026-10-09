package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	root := `C:\Users\Administrator\xwechat_files\wxid_mpvlokr4ycnu22_dfd6\cache`
	count := 0
	filepath.Walk(root, func(p string, info os.FileInfo, err error) error {
		if err != nil || info == nil || info.IsDir() {
			return nil
		}
		name := info.Name()
		if strings.HasSuffix(name, ".dat") {
			count++
			if count <= 5 {
				fmt.Println(name, info.Size())
			}
		}
		return nil
	})
	fmt.Println("总图片数:", count)
}
