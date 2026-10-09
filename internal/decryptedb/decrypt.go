// Package decryptedb 把 SQLCipher 4 加密的 db 文件解密成普通 SQLite 文件。
//
// 纯 Go 实现，不依赖 CGO。算法参考 SQLCipher 4 规范：
//   - Page size = 4096, reserve = 80 (IV 16 + HMAC 64)
//   - 每个 page 独立 AES-256-CBC 解密
//   - Page 1: [0:16] salt 明文, [16:4016] 加密, [4016:4032] IV, [4032:4096] HMAC
//   - Page N (N>1): [0:4016] 加密, [4016:4032] IV, [4032:4096] HMAC
package decryptedb

import (
	"crypto/aes"
	"crypto/cipher"
	"fmt"
	"os"
)

const (
	pageSize = 4096
	reserve  = 80
	ivSize   = 16
)

// DecryptFile 用 encKey 把 src 加密 db 解密到 dst。
// encKey 是 32 字节 post-PBKDF2 key（我们从内存里扫到的那种）。
func DecryptFile(src, dst string, encKey []byte) error {
	if len(encKey) != 32 {
		return fmt.Errorf("encKey 必须 32 字节，当前 %d", len(encKey))
	}
	in, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	if len(in)%pageSize != 0 {
		return fmt.Errorf("文件大小 %d 不是 pageSize(%d) 整数倍", len(in), pageSize)
	}
	block, err := aes.NewCipher(encKey)
	if err != nil {
		return err
	}
	out := make([]byte, len(in))
	pageCount := len(in) / pageSize
	for i := 0; i < pageCount; i++ {
		page := in[i*pageSize : (i+1)*pageSize]
		decryptPage(out[i*pageSize:(i+1)*pageSize], page, block, i+1)
	}
	return os.WriteFile(dst, out, 0644)
}

func decryptPage(dst, src []byte, block cipher.Block, pageNum int) {
	ivOffset := pageSize - reserve
	iv := src[ivOffset : ivOffset+ivSize]

	if pageNum == 1 {
		// page 1: [0:16] 是 salt（明文，加密前替换了 "SQLite format 3\0"）
		//         解密 [16:4016]，然后把 [0:16] 恢复成 SQLite 头
		mode := cipher.NewCBCDecrypter(block, iv)
		mode.CryptBlocks(dst[16:ivOffset], src[16:ivOffset])
		copy(dst[0:16], "SQLite format 3\x00")
	} else {
		mode := cipher.NewCBCDecrypter(block, iv)
		mode.CryptBlocks(dst[0:ivOffset], src[0:ivOffset])
	}
	for i := ivOffset; i < pageSize; i++ {
		dst[i] = 0
	}
}
