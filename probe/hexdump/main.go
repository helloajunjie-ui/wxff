package main

import (
	"encoding/hex"
	"fmt"
	"os"
)

func main() {
	data, _ := os.ReadFile(`F:\wxff\data\decrypted\sns_plain.db`)
	fmt.Printf("前 64 字节: %s\n", hex.EncodeToString(data[:64]))
	fmt.Printf("ASCII: %q\n", data[:64])
}
