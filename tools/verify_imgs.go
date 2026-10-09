package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	baseDir := `F:\wxff\data\backup\wxid_mpvlokr4ycnu22\聊天记录\化晓婷`
	f, _ := os.Open(filepath.Join(baseDir, "messages.jsonl"))
	defer f.Close()

	scanner := bufio.NewScanner(f)
	imgCount := 0
	missing := 0
	for scanner.Scan() {
		line := scanner.Text()
		var m map[string]any
		json.Unmarshal([]byte(line), &m)
		if m["type"] == "image" {
			imgCount++
			file := fmt.Sprintf("%v", m["file"])
			fullPath := filepath.Join(baseDir, filepath.FromSlash(file))
			if _, err := os.Stat(fullPath); err != nil {
				missing++
				fmt.Println("缺失:", file)
			}
		}
	}
	fmt.Printf("图片消息总数: %d, 缺失: %d\n", imgCount, missing)
}
