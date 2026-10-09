package dbkey

import "bytes"

// SaltNeighborhoodScanner 在内存里找 db 的 salt 字节，然后在 salt 前后 window
// 范围内试每个 32 字节块作为 enc_key。
//
// 这是 4.1+ 微信最靠谱的策略——key 不一定以 x'...' 字符串形式存在内存里，
// 但 salt 是 db 文件头里的公开数据，WCDB 打开 db 时必然在内存里构造 codec 上下文，
// salt 和 enc_key 通常相邻存放。
type SaltNeighborhoodScanner struct {
	Window int // 每个方向搜索窗口字节数，默认 64KB
}

func (s *SaltNeighborhoodScanner) Name() string { return "salt-neighborhood" }

// 只试 enc_key 模式（2 次 PBKDF2，快）。不试 password 模式（256k 次，太慢），
// master password 留给 bare-hex scanner（候选少）。
var encKeyOnly = NewMatcher(VerifyEncKey{})

func (s *SaltNeighborhoodScanner) Scan(view []byte, dbs []DB, saltIdx map[string][]int,
	matcher *Matcher, found chan<- Hit, remaining map[string]struct{}) {

	window := s.Window
	if window <= 0 {
		window = 64 * 1024
	}
	_ = matcher // salt 邻域里的候选直接当 enc_key 试，不用外部 matcher

	for saltStr := range remaining {
		salt := []byte(saltStr)
		offset := 0
		for {
			idx := bytes.Index(view[offset:], salt)
			if idx < 0 {
				break
			}
			abs := offset + idx
			start := abs - window
			if start < 0 {
				start = 0
			}
			end := abs + SaltSize + window
			if end > len(view) {
				end = len(view)
			}
			for k := start; k+KeySize <= end; k++ {
				cand := Candidate(view[k : k+KeySize])
				if !looksLikeKey(cand) {
					continue
				}
				for _, di := range saltIdx[saltStr] {
					if encKey, method, ok := encKeyOnly.Check(cand, dbs[di].Page1); ok {
						select {
						case found <- Hit{
							DBRel:  dbs[di].Rel,
							Salt:   dbs[di].Salt,
							EncKey: encKey,
							Method: method,
						}:
						default:
						}
						delete(remaining, saltStr)
						goto nextSalt
					}
				}
			}
			offset = abs + 1
		}
	nextSalt:
	}
}

// looksLikeKey 过滤明显不是 key 的块（全 0、连续两个 0）。
func looksLikeKey(b []byte) bool {
	if len(b) != KeySize {
		return false
	}
	run := 0
	for _, x := range b {
		if x == 0 {
			run++
			if run >= 2 {
				return false
			}
		} else {
			run = 0
		}
	}
	return true
}
