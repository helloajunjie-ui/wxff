// sns_dup 检测缓存中是否存在"同一图片的多个分辨率副本"（缩略图+大图混存）。
// 思路：对每张缓存图计算感知指纹（宽高比 + 平均色），找出宽高比相同但尺寸不同的成对文件。
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
	block, _ := aes.NewCipher([]byte(imageKey))
	out := make([]byte, aesLen)
	for i := 0; i < aesLen; i += 16 {
		block.Decrypt(out[i:i+16], body[i:i+16])
	}
	for i := 0; i < xorSize; i++ {
		out = append(out, body[aesLen+i]^byte(xorKey))
	}
	return out, nil
}

type rec struct {
	path string
	w, h int
}

func main() {
	root := `C:\Users\Administrator\xwechat_files\wxid_mpvlokr4ycnu22_dfd6\cache`
	var recs []rec
	filepath.Walk(root, func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.Contains(p, `\Sns\Img\`) {
			return nil
		}
		stem := filepath.Base(p)
		if i := strings.LastIndex(stem, "."); i >= 0 {
			stem = stem[:i]
		}
		if len(stem) != 30 {
			return nil
		}
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
		recs = append(recs, rec{p, cfg.Width, cfg.Height})
		return nil
	})

	fmt.Printf("缓存图片总数: %d\n", len(recs))

	// 按宽高比分组（保留 3 位小数），找出同比例不同尺寸的组
	ratio := func(w, h int) float64 { return float64(w) / float64(h) }
	groups := map[string][]rec{}
	for _, r := range recs {
		key := fmt.Sprintf("%.3f", ratio(r.w, r.h))
		groups[key] = append(groups[key], r)
	}

	dupGroups := 0
	dupPairs := 0
	for _, g := range groups {
		if len(g) < 2 {
			continue
		}
		// 检查组内是否有尺寸不同的
		sizes := map[[2]int]int{}
		for _, r := range g {
			sizes[[2]int{r.w, r.h}]++
		}
		if len(sizes) < 2 {
			continue // 同比例且同尺寸，可能是完全相同的图，不算缩略图对
		}
		dupGroups++
		dupPairs += len(g) - 1
	}

	fmt.Printf("宽高比相同但尺寸不同的组数: %d（涉及约 %d 张）\n", dupGroups, dupPairs)

	// 展示几个典型组
	type kv struct {
		key string
		g   []rec
	}
	var list []kv
	for k, g := range groups {
		if len(g) < 2 {
			continue
		}
		sizes := map[[2]int]int{}
		for _, r := range g {
			sizes[[2]int{r.w, r.h}]++
		}
		if len(sizes) < 2 {
			continue
		}
		list = append(list, kv{k, g})
	}
	sort.Slice(list, func(i, j int) bool { return len(list[i].g) > len(list[j].g) })

	fmt.Println("\n=== 典型「同比例多尺寸」组（前 10）===")
	for i, e := range list {
		if i >= 10 {
			break
		}
		sizes := map[[2]int]int{}
		for _, r := range e.g {
			sizes[[2]int{r.w, r.h}]++
		}
		var ss []string
		for s, n := range sizes {
			ss = append(ss, fmt.Sprintf("%dx%d(x%d)", s[0], s[1], n))
		}
		sort.Strings(ss)
		fmt.Printf("  比例 %s: %s\n", e.key, strings.Join(ss, ", "))
	}
}
