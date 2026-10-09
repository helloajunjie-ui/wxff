package main

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
)

func main() {
	key := "198511700483708433"
	token := "qe9uKlQ2RT8F1FD44V7hbiaGw3FzfJpsGZ832cBdvEuDWHesvC84WjGx5BZc5vDM0qbQxOgicvjoQ"

	h1 := md5.Sum([]byte(key))
	fmt.Println("md5(key):", hex.EncodeToString(h1[:]))

	h2 := md5.Sum([]byte(token))
	fmt.Println("md5(token):", hex.EncodeToString(h2[:]))

	// 看看 statExtStr 解码后是什么
	// GhQKEnd4YWFkYmFiOWQxM2VkZmYyMA==
	fmt.Println("\nstatExtStr base64 解码:")
	fmt.Println(decodeBase64("GhQKEnd4YWFkYmFiOWQxM2VkZmYyMA=="))
}

func decodeBase64(s string) string {
	// 简单实现
	return s
}
