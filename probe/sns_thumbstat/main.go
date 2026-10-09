// sns_thumbstat 统计缓存图片中"缩略图级"与"大图级"的数量占比，并抽样验证缩略图是否为独立文件。
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
	type rec struct {
		path string
		w, h int
	}
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

	// 按最长边分档
	buckets := map[string]int{}
	for _, r := range recs {
		long := r.w
		if r.h > long {
			long = r.h
		}
		var k string
		switch {
		case long <= 160:
			k = "<=160 (头像/极小缩略图)"
		case long <= 320:
			k = "161-320 (缩略图)"
		case long <= 640:
			k = "321-640 (小图)"
		case long <= 1280:
			k = "641-1280 (中图)"
		case long <= 1920:
			k = "1281-1920 (大图)"
		default:
			k = ">1920 (超大图)"
		}
		buckets[k]++
	}
	fmt.Printf("缓存图片总数: %d\n\n=== 按最长边分档 ===\n", len(recs))
	var keys []string
	for k := range buckets {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		fmt.Printf("  %-28s %d\n", k, buckets[k])
	}

	// 抽样：找一张 276x276 的，看它的文件名与同比例大图是否不同文件
	fmt.Println("\n=== 抽样：276x276 缩略图文件 ===")
	n := 0
	for _, r := range recs {
		if r.w == 276 && r.h == 276 {
			fmt.Printf("  %s\n", filepath.Base(r.path))
			n++
			if n >= 5 {
				break
			}
		}
	}
}
