// probe/video_hunt 全盘搜索所有可能的视频文件（按扩展名 + 文件头魔数）。
// 目的：确认聊天视频到底有没有本地文件，藏在哪。
package main

import (
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var dataRoot = `C:\Users\Administrator\xwechat_files\wxid_mpvlokr4ycnu22_dfd6`

// 视频文件头魔数
var videoMagics = map[string]string{
	"00000018": "mp4(ftyp)",
	"00000020": "mp4(ftyp)",
	"1a45dfa3": "mkv/webm",
	"464c5601": "flv",
	"52494646": "avi/wav(RIFF)",
	"000001ba": "mpg",
	"000001b3": "mpg",
}

func isVideoHead(head []byte) string {
	if len(head) < 4 {
		return ""
	}
	h := hex.EncodeToString(head[:4])
	if name, ok := videoMagics[h]; ok {
		return name
	}
	// mp4: 前 4 字节是 box size，5-8 字节是 'ftyp'
	if len(head) >= 8 && string(head[4:8]) == "ftyp" {
		return "mp4(ftyp)"
	}
	return ""
}

func main() {
	extCount := map[string]int{}
	headCount := map[string]int{}
	var videoFiles []string

	filepath.Walk(dataRoot, func(p string, info os.FileInfo, err error) error {
		if err != nil || info == nil || info.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(p))
		// 1. 按扩展名统计
		if ext == ".mp4" || ext == ".mov" || ext == ".avi" || ext == ".mkv" ||
			ext == ".flv" || ext == ".wmv" || ext == ".m4v" || ext == ".ts" ||
			ext == ".mpg" || ext == ".mpeg" || ext == ".3gp" {
			extCount[ext]++
			videoFiles = append(videoFiles, "[EXT] "+p)
		}
		// 2. 按文件头魔数（只查 >4KB 的文件，避免噪音）
		if info.Size() > 4096 {
			f, e := os.Open(p)
			if e == nil {
				head := make([]byte, 16)
				n, _ := f.Read(head)
				f.Close()
				if n >= 8 {
					if name := isVideoHead(head[:n]); name != "" {
						headCount[name]++
						videoFiles = append(videoFiles, fmt.Sprintf("[HEAD:%s] %s", name, p))
					}
				}
			}
		}
		return nil
	})

	fmt.Println("=== 按扩展名统计 ===")
	for k, v := range extCount {
		fmt.Printf("  %s: %d\n", k, v)
	}
	fmt.Println("=== 按文件头统计 ===")
	for k, v := range headCount {
		fmt.Printf("  %s: %d\n", k, v)
	}
	fmt.Println("=== 视频文件清单 ===")
	for _, f := range videoFiles {
		fmt.Println(" ", f)
	}
	fmt.Printf("总计: %d\n", len(videoFiles))
}
