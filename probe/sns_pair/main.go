// sns_pair 判断缩略图与大图是否成对：对每个宽高比，看是否存在"大图版本"。
// 目的：若过滤掉缩略图，是否会丢失只有缩略图的图片。
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

func main() {
	root := `C:\Users\Administrator\xwechat_files\wxid_mpvlokr4ycnu22_dfd6\cache`
	type rec struct{ w, h int }
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
		recs = append(recs, rec{cfg.Width, cfg.Height})
		return nil
	})

	// 按宽高比分组，记录每组的尺寸集合
	ratio := func(w, h int) string { return fmt.Sprintf("%.3f", float64(w)/float64(h)) }
	groups := map[string]map[[2]int]int{}
	for _, r := range recs {
		k := ratio(r.w, r.h)
		if groups[k] == nil {
			groups[k] = map[[2]int]int{}
		}
		groups[k][[2]int{r.w, r.h}]++
	}

	// 对每个比例组：判断是否有"大图"（长边>=720）和"缩略图"（长边<=320）
	onlyThumb := 0 // 只有缩略图，无大图
	hasBoth := 0   // 缩略图+大图都有
	onlyBig := 0   // 只有大图
	thumbCount := 0
	bigCount := 0
	for _, sizes := range groups {
		hasT, hasB := false, false
		for s, n := range sizes {
			long := s[0]
			if s[1] > long {
				long = s[1]
			}
			if long <= 320 {
				hasT = true
				thumbCount += n
			}
			if long >= 720 {
				hasB = true
				bigCount += n
			}
		}
		switch {
		case hasT && hasB:
			hasBoth++
		case hasT && !hasB:
			onlyThumb++
		case !hasT && hasB:
			onlyBig++
		}
	}

	fmt.Printf("缓存图片总数: %d\n", len(recs))
	fmt.Printf("缩略图(长边<=320)总数: %d\n", thumbCount)
	fmt.Printf("大图(长边>=720)总数: %d\n\n", bigCount)
	fmt.Printf("=== 按宽高比分组（共 %d 组）===\n", len(groups))
	fmt.Printf("  缩略图+大图都有: %d 组\n", hasBoth)
	fmt.Printf("  只有缩略图(无大图): %d 组\n", onlyThumb)
	fmt.Printf("  只有大图(无缩略图): %d 组\n", onlyBig)

	// 列出"只有缩略图"的组，看是否真的没有大图版本
	fmt.Println("\n=== 「只有缩略图」的组（前 15）===")
	type kv struct {
		k     string
		sizes map[[2]int]int
	}
	var list []kv
	for k, sizes := range groups {
		hasT, hasB := false, false
		for s := range sizes {
			long := s[0]
			if s[1] > long {
				long = s[1]
			}
			if long <= 320 {
				hasT = true
			}
			if long >= 720 {
				hasB = true
			}
		}
		if hasT && !hasB {
			list = append(list, kv{k, sizes})
		}
	}
	sort.Slice(list, func(i, j int) bool { return len(list[i].sizes) > len(list[j].sizes) })
	for i, e := range list {
		if i >= 15 {
			break
		}
		var ss []string
		for s, n := range e.sizes {
			ss = append(ss, fmt.Sprintf("%dx%d(x%d)", s[0], s[1], n))
		}
		sort.Strings(ss)
		fmt.Printf("  比例 %s: %s\n", e.k, strings.Join(ss, ", "))
	}
}
