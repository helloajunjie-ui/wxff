"""从 candidate_keys.txt 找到 salt 匹配的 key，验证 HMAC。"""
import hashlib
import hmac
import struct

DB = r"F:\wxff\data\decrypted\sns.db"
CANDIDATES = r"F:\wxff\data\candidate_keys.txt"

# 读 db salt
with open(DB, "rb") as f:
    page1 = f.read(4096)
salt = page1[:16]
print(f"[db salt] {salt.hex()}")

# 在候选里找
with open(CANDIDATES) as f:
    candidates = [line.strip() for line in f if line.strip()]

matched = None
for line in candidates:
    if len(line) != 96:
        continue
    if bytes.fromhex(line[64:96]) == salt:
        matched = line
        break

if not matched:
    print("[!] 没找到 salt 匹配的候选")
    raise SystemExit(1)

print(f"[匹配] {matched}")
enc_key = bytes.fromhex(matched[0:64])
print(f"[enc_key] {enc_key.hex()}")

# 派生 HMAC key：PBKDF2(enc_key, salt, 2, 32, SHA512)
hmac_key = hashlib.pbkdf2_hmac("sha512", enc_key, salt, 2, 32)

# 验证 page 1 HMAC
# SQLCipher 4: reserve = IV(16) + HMAC(64) = 80
# page1[16:4016] 加密数据, page1[4016:4032] IV, page1[4032:4096] HMAC
stored_hmac = page1[4032:4096]
page_num = struct.pack("<I", 1)
hmac_input = page1[16:4032] + page_num
computed = hmac.new(hmac_key, hmac_input, hashlib.sha512).digest()

print(f"[存储 HMAC] {stored_hmac.hex()[:32]}...")
print(f"[计算 HMAC] {computed.hex()[:32]}...")

if hmac.compare_digest(stored_hmac, computed):
    print("\n[成功] KEY 验证通过！")
    # 保存最终 key 供后续用
    with open(r"F:\wxff\data\final_key.txt", "w") as f:
        f.write(matched)
    print("[OK] 已写入 F:\\wxff\\data\\final_key.txt")
else:
    print("\n[失败] HMAC 不匹配")
