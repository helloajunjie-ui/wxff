import sqlite3
import zstandard as zstd
import hashlib

db = sqlite3.connect(r'F:\wxff\data\backup\_plain\message__message_0.db')
cur = db.cursor()

table = "Msg_" + hashlib.md5("e595400290".encode()).hexdigest()
cur.execute(f"SELECT message_content FROM {table} WHERE local_type=3 LIMIT 1")
content = cur.fetchone()[0]

dctx = zstd.ZstdDecompressor()
out = dctx.decompress(content)
print("完整 XML:")
print(out.decode('utf-8', errors='replace'))
