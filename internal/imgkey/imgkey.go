// Package imgkey 从 Weixin.exe 内存提取图片 AES-128-ECB key，并解密 .dat 文件。
//
// 算法参考 r266-tech/wxkey:
//   - image key 是 16 个 ASCII 字符（直接作为 AES-128 key，不是 hex decode）
//   - 验证：AES-128-ECB 解密 .dat offset 15 处 16 字节，看是不是 JPEG/PNG 头
//   - XOR key 从 .dat 尾部算：tail[-2]^0xff == tail[-1]^0xd9
package imgkey

import (
	"bytes"
	"crypto/aes"
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// DecryptedHeader 是解密后校验通过的结果。
type DecryptedHeader struct {
	Key    string // 16 ASCII chars
	XORKey int
	IsJPEG bool
	IsPNG  bool
}

// IsWechatV4DAT 判断文件头是不是微信 V4 图片 dat。
func IsWechatV4DAT(data []byte) bool {
	return len(data) >= 15 &&
		data[0] == 0x07 && data[1] == 0x08 && data[2] == 0x56 &&
		(data[3] == 0x31 || data[3] == 0x32) &&
		data[4] == 0x08 && data[5] == 0x07 &&
		binary.LittleEndian.Uint32(data[6:10]) > 0
}

// FindTemplate 从 attach 目录找一个最新的 _t.dat（缩略图）作为验证模板。
func FindTemplate(attachRoot string) (path string, encryptedBlock []byte, xorKey *int, err error) {
	var best string
	var bestMtime int64
	filepath.Walk(attachRoot, func(p string, info os.FileInfo, e error) error {
		if e != nil || info == nil || info.IsDir() {
			return nil
		}
		name := strings.ToLower(info.Name())
		if !strings.HasSuffix(name, "_t.dat") {
			return nil
		}
		data, e := os.ReadFile(p)
		if e != nil || !IsWechatV4DAT(data) || len(data) < 15+aes.BlockSize {
			return nil
		}
		if info.ModTime().UnixNano() > bestMtime {
			bestMtime = info.ModTime().UnixNano()
			best = p
		}
		return nil
	})
	if best == "" {
		return "", nil, nil, fmt.Errorf("没找到 V4 .dat 模板")
	}
	data, _ := os.ReadFile(best)
	enc := append([]byte{}, data[15:15+aes.BlockSize]...)
	xk := deriveXORKey(data)
	return best, enc, xk, nil
}

func deriveXORKey(data []byte) *int {
	if !IsWechatV4DAT(data) || len(data) < 17 {
		return nil
	}
	xorSize := binary.LittleEndian.Uint32(data[10:14])
	if xorSize < 2 {
		return nil
	}
	body := data[15:]
	if uint32(len(body)) < xorSize {
		return nil
	}
	tail := body[len(body)-int(xorSize):]
	k0 := int(tail[len(tail)-2] ^ 0xff)
	k1 := int(tail[len(tail)-1] ^ 0xd9)
	if k0 != k1 {
		return nil
	}
	return &k0
}

// ValidateKey 用 candidate（16 ASCII chars）尝试解密 encBlock（.dat offset 15 的 16 字节）。
func ValidateKey(candidate []byte, encBlock []byte) bool {
	if len(candidate) < 16 || len(encBlock) < 16 {
		return false
	}
	block, err := aes.NewCipher(candidate[:16])
	if err != nil {
		return false
	}
	dec := make([]byte, 16)
	block.Decrypt(dec, encBlock[:16])
	return bytes.HasPrefix(dec, []byte{0xff, 0xd8, 0xff}) ||
		bytes.HasPrefix(dec, []byte{0x89, 0x50, 0x4e, 0x47})
}

// DecryptDAT 用 imageKey 和 xorKey 解密整个 .dat 文件。
//
// 实测（PC 微信 4.1.9.57）真实布局为：
//
//	body = data[15:]
//	AES-128-ECB 加密 body 的前 (len(body) - xorSize) 字节；
//	末尾 xorSize 字节做单字节 XOR。
//
// 注意：头部 [6:10] 的 aesSize 字段并不是 AES 区域长度（实测常为 1024，
// 与真实 AES 区域不符），真正决定 AES 区域的是 xorSize：
// AES 区域 = len(body) - xorSize。用 aesSize 会导致大图（如 _h.dat）
// 中间大段数据未被解密而损坏。
func DecryptDAT(data []byte, imageKey string, xorKey int) ([]byte, error) {
	if !IsWechatV4DAT(data) {
		return nil, fmt.Errorf("不是 V4 dat")
	}
	xorSize := binary.LittleEndian.Uint32(data[10:14])
	body := data[15:]
	out := make([]byte, len(body))

	block, err := aes.NewCipher([]byte(imageKey)[:16])
	if err != nil {
		return nil, err
	}

	// AES 区域 = body 去掉末尾 xorSize 字节。
	nAES := len(body) - int(xorSize)
	if nAES < 0 {
		nAES = 0
	}
	if nAES > len(body) {
		nAES = len(body)
	}
	// AES 解密要求长度是 16 的倍数。
	nAES &= ^15
	// cipher.Block.Decrypt 一次只解密一个 16 字节块，必须逐块循环，
	// 否则只有第一块被解密，其余保持密文（导致图片损坏）。
	for off := 0; off+16 <= nAES; off += 16 {
		block.Decrypt(out[off:off+16], body[off:off+16])
	}
	// 中间（AES 区域与 XOR 区域之间的对齐余数）保持原样。
	copy(out[nAES:], body[nAES:])
	// 末尾 xorSize 字节单字节 XOR。
	nXor := int(xorSize)
	if nXor > 0 && nXor <= len(out) {
		for i := len(out) - nXor; i < len(out); i++ {
			out[i] ^= byte(xorKey)
		}
	}
	return out, nil
}
