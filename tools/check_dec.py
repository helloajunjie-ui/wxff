import sqlite3
import zstandard as zstd

db = sqlite3.connect(r'F:\wxff\data\backup\_plain\message__message_0.db')
cur = db.cursor()

import hashlib
table = "Msg_" + hashlib.md5("e595400290".encode()).hexdigest()
cur.execute(f"SELECT message_content FROM {table} WHERE local_type=3 LIMIT 1")
content = cur.fetchone()[0]
print("content 长度:", len(content))
print("前 4 字节:", content[:4].hex())

dctx = zstd.ZstdDecompressor()
try:
    out = dctx.decompress(content)
    print("解压后长度:", len(out))
    print("解压后前 500 字:")
    print(out[:500].decode('utf-8', errors='replace'))
except Exception as e:
    print("解压失败:", e)
