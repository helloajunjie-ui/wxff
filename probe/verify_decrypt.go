package main

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/hex"
	"fmt"
	"os"
)

func main() {
	keyHex := "661c1de513f2406ce27eaba76d7a377ea954b9899415de214e5772dc93e532ea"
	dbPath := `C:\Users\Administrator\xwechat_files\wxid_mpvlokr4ycnu22_dfd6\db_storage\bizchat\bizchat.db`

	key, _ := hex.DecodeString(keyHex)
	data, _ := os.ReadFile(dbPath)
	page1 := data[:4096]

	// page1[16:4016] 加密内容, page1[4016:4032] IV
	iv := page1[4016:4032]
	ct := page1[16:4016]

	block, err := aes.NewCipher(key)
	if err != nil {
		panic(err)
	}
	mode := cipher.NewCBCDecrypter(block, iv)
	pt := make([]byte, len(ct))
	mode.CryptBlocks(pt, ct)

	fmt.Printf("解密后前 64 字节: %q\n", pt[:64])
	if string(pt[:16]) == "SQLite format 3\x00" {
		fmt.Println("[成功] 看到 SQLite 头！key 正确！")
	} else {
		fmt.Println("[失败] 不是 SQLite 头")
	}
}
