// Package dbkey 负责从微信进程内存提取 SQLCipher 4 数据库密钥。
//
// 子包划分：
//   - verify: 纯算法，HMAC/PBKDF2 验证候选 key 是否正确
//   - db:     db 文件收集，读 salt 和 page1
//   - scanner: 各种内存扫描策略（salt 邻域、x'...' 字面量等），可插拔
package dbkey

import "crypto/hmac"

// Page 布局常量（SQLCipher 4 / WCDB v4）
const (
	PageSize = 4096
	SaltSize = 16
	KeySize  = 32
	HMACSize = 64
	KDFIter  = 256000
)

// Candidate 表示一个从内存里捞出来的 32 字节候选 key。
type Candidate []byte

// Hit 表示一个验证通过的 (db, key) 对。
type Hit struct {
	DBRel  string // db 路径（相对数据根目录）
	Salt   []byte // 16 字节 salt
	EncKey []byte // 32 字节 post-PBKDF2 enc_key
	Method string // "enc_key" 或 "password"
}

// Verify 是验证器，给定一个候选和 page1，判断是否能解开 db。
// 不同实现对应不同的 key 形态（raw enc_key / master password）。
type Verify interface {
	// Name 返回验证器名。
	Name() string
	// Check 用候选尝试解 page1，成功返回 post-KDF enc_key。
	Check(cand Candidate, page1 []byte) (encKey []byte, ok bool)
}

// Matcher 聚合多个 verify，按成本从低到高试。
type Matcher struct {
	verifiers []Verify
}

func NewMatcher(vs ...Verify) *Matcher {
	return &Matcher{verifiers: vs}
}

// Check 依次试每个 verifier，返回第一个命中的。
func (m *Matcher) Check(cand Candidate, page1 []byte) (encKey []byte, method string, ok bool) {
	for _, v := range m.verifiers {
		if ek, hit := v.Check(cand, page1); hit {
			return ek, v.Name(), true
		}
	}
	return nil, "", false
}

// HMACEqual 暴露给测试。
func HMACEqual(a, b []byte) bool { return hmac.Equal(a, b) }
