"""用扫到的 key 解密 sns.db，dump schema 和前几条记录。"""
import os
import shutil
import sqlite3
import sys

KEY = "856e5a26a59c62408288962773f6985a960257022c1f134172cbbb1d481d1da55237edf0900289caf3358227bf43174a"
SRC_DIR = r"C:\Users\Administrator\xwechat_files\wxid_mpvlokr4ycnu22_dfd6\db_storage\sns"
DST_DIR = r"F:\wxff\data\decrypted"

os.makedirs(DST_DIR, exist_ok=True)

# 拷贝 db + wal + shm（共享读）
for ext in ["db", "db-wal", "db-shm"]:
    src = os.path.join(SRC_DIR, f"sns.{ext}")
    dst = os.path.join(DST_DIR, f"sns.{ext}")
    if os.path.exists(src):
        with open(src, "rb") as fsrc, open(dst, "wb") as fdst:
            shutil.copyfileobj(fsrc, fdst)
        print(f"[copy] sns.{ext}")

# 用 sqlcipher 打开
try:
    from pysqlcipher3 import dbapi2 as sqlcipher
except ImportError:
    print("[!] pysqlcipher3 未安装，尝试安装...")
    os.system(f'"{sys.executable}" -m pip install pysqlcipher3-binary')
    from pysqlcipher3 import dbapi2 as sqlcipher

conn = sqlcipher.connect(os.path.join(DST_DIR, "sns.db"))
cur = conn.cursor()
# SQLCipher 4 默认参数
cur.execute(f"PRAGMA key = \"x'{KEY}'\";")
cur.execute("PRAGMA cipher_page_size = 4096;")
cur.execute("PRAGMA kdf_iter = 256000;")
cur.execute("PRAGMA cipher_hmac_algorithm = HMAC_SHA512;")
cur.execute("PRAGMA cipher_kdf_algorithm = PBKDF2_HMAC_SHA512;")
cur.execute("PRAGMA cipher_memory_security = OFF;")

# 试一个简单查询
try:
    cur.execute("SELECT count(*) FROM sqlite_master;")
    print(f"[OK] 解密成功！sqlite_master 表数: {cur.fetchone()[0]}")
except Exception as e:
    print(f"[失败] 解密失败: {e}")
    sys.exit(2)

# dump schema
cur.execute("SELECT name, type FROM sqlite_master WHERE type IN ('table','view') ORDER BY name;")
tables = cur.fetchall()
print(f"\n=== 表/视图 ===")
for name, typ in tables:
    print(f"  [{typ}] {name}")

# 找类似 SnsInfo 的主表
print(f"\n=== 主表内容预览 ===")
for name, _ in tables:
    if "sns" in name.lower() or "info" in name.lower() or "comment" in name.lower():
        try:
            cur.execute(f"SELECT count(*) FROM {name};")
            cnt = cur.fetchone()[0]
            print(f"\n[{name}] 共 {cnt} 行")
            cur.execute(f"PRAGMA table_info({name});")
            cols = cur.fetchall()
            print("  列:", [c[1] for c in cols])
        except Exception as e:
            print(f"  {name}: {e}")

conn.close()
