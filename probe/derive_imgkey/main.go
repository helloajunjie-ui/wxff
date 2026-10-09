package main

import (
	"crypto/md5"
	"fmt"
)

func main() {
	codes := []string{"3483575526"}
	wxids := []string{
		"wxid_mpvlokr4ycnu22",
		"wxid_mpvlokr4ycnu22_dfd6",
	}
	for _, code := range codes {
		for _, wxid := range wxids {
			sum := md5.Sum([]byte(code + wxid))
			key := fmt.Sprintf("%x", sum)[:16]
			fmt.Printf("md5(%s + %s) = %s\n", code, wxid, key)
		}
	}
	fmt.Println("\n目标 key = b30e5d7754486daf")
}
