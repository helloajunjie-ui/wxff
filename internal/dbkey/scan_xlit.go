package dbkey

import (
	"encoding/hex"
	"regexp"
)

// XLiteralScanner 找 `x'<64-192 hex>'` 这种 SQL PRAGMA key 字面量。
// 老版本微信（4.0.x）常用；4.1+ 可能不再出现，但作为兜底保留。
type XLiteralScanner struct{}

func (s *XLiteralScanner) Name() string { return "x-literal" }

var xlitRe = regexp.MustCompile(`x'([0-9a-fA-F]{64,192})'`)

func (s *XLiteralScanner) Scan(view []byte, dbs []DB, saltIdx map[string][]int,
	matcher *Matcher, found chan<- Hit, remaining map[string]struct{}) {

	for _, sub := range xlitRe.FindAll(view, -1) {
		hexStr := string(sub[2 : len(sub)-1]) // 去掉 x' 和 '
		raw, err := hex.DecodeString(hexStr)
		if err != nil || len(raw) < KeySize {
			continue
		}
		cand := Candidate(raw[:KeySize])
		// 如果带了 salt（96 hex），只试匹配的 db
		if len(hexStr) >= 96 {
			saltHex := hexStr[len(hexStr)-32:]
			saltBytes, _ := hex.DecodeString(saltHex)
			if dbIdxs, ok := saltIdx[string(saltBytes)]; ok {
				if _, still := remaining[string(saltBytes)]; still {
					for _, di := range dbIdxs {
						if encKey, method, hit := matcher.Check(cand, dbs[di].Page1); hit {
							found <- Hit{DBRel: dbs[di].Rel, Salt: dbs[di].Salt, EncKey: encKey, Method: method}
							delete(remaining, string(saltBytes))
							break
						}
					}
				}
			}
			continue
		}
		// 不带 salt，试所有未命中的
		for saltStr := range remaining {
			for _, di := range saltIdx[saltStr] {
				if encKey, method, hit := matcher.Check(cand, dbs[di].Page1); hit {
					found <- Hit{DBRel: dbs[di].Rel, Salt: dbs[di].Salt, EncKey: encKey, Method: method}
					delete(remaining, saltStr)
					break
				}
			}
		}
	}
}
