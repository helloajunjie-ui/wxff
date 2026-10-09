// sns_dims 统计朋友圈缓存图片的像素尺寸分布，判断是否缩略图/原图混杂。
package main

import (
	"bytes"
	"crypto/aes"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const imageKey = "b30e5d7754486daf"
const xorKey = 0xE6

func decryptDAT(data []byte) ([]byte, error) {
	if len(data) < 15 {
		return nil, fmt.Errorf("too short")
	}
	body := data[15:]
	xorSize := int(data[10]) | int(data[11])<<8 | int(data[12])<<16 | int(data[13])<<24
	if xorSize < 0 || xorSize > len(body) {
		return nil, fmt.Errorf("bad xorSize")
	}
	aesLen := len(body) - xorSize
	aesLen -= aesLen % 16
	if aesLen <= 0 {
		return nil, fmt.Errorf("bad aesLen")
	}
	block, err := aes.NewCipher([]byte(imageKey))
	if err != nil {
		return nil, err
	}
	out := make([]byte, aesLen)
	for i := 0; i < aesLen; i += 16 {
		block.Decrypt(out[i:i+16], body[i:i+16])
	}
	for i := 0; i < xorSize; i++ {
		out = append(out, body[aesLen+i]^byte(xorKey))
	}
	return out, nil
}

func main() {
	root := `C:\Users\Administrator\xwechat_files\wxid_mpvlokr4ycnu22_dfd6\cache`
	type dim struct{ w, h int }
	counts := map[dim]int{}
	total, ok := 0, 0
	filepath.Walk(root, func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if !strings.Contains(p, `\Sns\Img\`) {
			return nil
		}
		stem := filepath.Base(p)
		if i := strings.LastIndex(stem, "."); i >= 0 {
			stem = stem[:i]
		}
		if len(stem) != 30 {
			return nil
		}
		total++
		raw, err := os.ReadFile(p)
		if err != nil {
			return nil
		}
		dec, err := decryptDAT(raw)
		if err != nil {
			return nil
		}
		cfg, _, err := image.DecodeConfig(bytes.NewReader(dec))
		if err != nil {
			return nil
		}
		ok++
		counts[dim{cfg.Width, cfg.Height}]++
		return nil
	})

	fmt.Printf("Sns/Img 文件总数: %d  可解码: %d\n", total, ok)

	// 按面积排序输出
	type kv struct {
		d dim
		n int
	}
	var list []kv
	for d, n := range counts {
		list = append(list, kv{d, n})
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].d.w*list[i].d.h > list[j].d.w*list[j].d.h
	})

	fmt.Println("\n=== 尺寸分布（按面积降序，前 40）===")
	for i, e := range list {
		if i >= 40 {
			break
		}
		fmt.Printf("  %5d x %-5d  x%d\n", e.d.w, e.d.h, e.n)
	}

	// 分档统计
	var thumb, mid, big, huge int
	for d, n := range counts {
		area := d.w * d.h
		switch {
		case area < 100*100:
			thumb += n
		case area < 500*500:
			mid += n
		case area < 1500*1500:
			big += n
		default:
			huge += n
		}
	}
	fmt.Printf("\n=== 分档 ===\n")
	fmt.Printf("  <100x100 (缩略图级): %d\n", thumb)
	fmt.Printf("  100x100~500x500 (小图): %d\n", mid)
	fmt.Printf("  500x500~1500x1500 (中图): %d\n", big)
	fmt.Printf("  >=1500x1500 (大图/原图级): %d\n", huge)
}
