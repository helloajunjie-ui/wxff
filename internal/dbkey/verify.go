package dbkey

import (
	"crypto/hmac"
	"crypto/pbkdf2"
	"crypto/sha512"
	"encoding/binary"
)

// VerifyEncKey 把候选当成已经过 PBKDF2 的 enc_key，直接验证。
// 成本：1 次 2 轮 PBKDF2 + 1 次 HMAC。
type VerifyEncKey struct{}

func (VerifyEncKey) Name() string { return "enc_key" }

func (VerifyEncKey) Check(cand Candidate, page1 []byte) ([]byte, bool) {
	if len(cand) != KeySize || len(page1) < PageSize {
		return nil, false
	}
	if !verifyMAC(cand, page1) {
		return nil, false
	}
	out := make([]byte, KeySize)
	copy(out, cand)
	return out, true
}

// VerifyPassword 把候选当成 master password，先做 256000 轮 PBKDF2 派生 enc_key。
// 成本：1 次 256k PBKDF2（慢）+ 1 次 2 轮 PBKDF2。
type VerifyPassword struct{}

func (VerifyPassword) Name() string { return "password" }

func (VerifyPassword) Check(cand Candidate, page1 []byte) ([]byte, bool) {
	if len(cand) != KeySize || len(page1) < PageSize {
		return nil, false
	}
	salt := page1[:SaltSize]
	encKey, err := pbkdf2.Key(sha512.New, string(cand), salt, KDFIter, KeySize)
	if err != nil {
		return nil, false
	}
	if !verifyMAC(encKey, page1) {
		return nil, false
	}
	return encKey, true
}

// verifyMAC 是 SQLCipher 4 page-1 HMAC 校验。
//
//	macSalt = salt XOR 0x3A
//	macKey  = PBKDF2(encKey, macSalt, 2, 32, SHA512)
//	HMAC    = HMAC-SHA512(macKey, page1[16:4032] || LE32(pageNum=1))
//	对比     page1[4032:4096]
func verifyMAC(encKey, page1 []byte) bool {
	salt := page1[:SaltSize]
	macSalt := make([]byte, SaltSize)
	for i, b := range salt {
		macSalt[i] = b ^ 0x3A
	}
	macKey, err := pbkdf2.Key(sha512.New, string(encKey), macSalt, 2, KeySize)
	if err != nil {
		return false
	}
	h := hmac.New(sha512.New, macKey)
	h.Write(page1[SaltSize : PageSize-HMACSize]) // [16:4032] = content + IV
	var pn [4]byte
	binary.LittleEndian.PutUint32(pn[:], 1)
	h.Write(pn[:])
	return hmac.Equal(h.Sum(nil), page1[PageSize-HMACSize:PageSize])
}

// DefaultMatcher 按成本从低到高排好序：先试 enc_key（快），再试 password（慢）。
func DefaultMatcher() *Matcher {
	return NewMatcher(VerifyEncKey{}, VerifyPassword{})
}
