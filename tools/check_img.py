import sqlite3, zstd, re

db = sqlite3.connect(r'F:\wxff\data\backup\_plain\message__message_0.db')
cur = db.cursor()

# 找一个有图片消息的表
cur.execute("SELECT name FROM sqlite_master WHERE type='table' AND name LIKE 'Msg_%'")
tables = [t[0] for t in cur.fetchall()]
print(f"共 {len(tables)} 个表")

# 找第一个有图片的表
for table in tables:
    cur.execute(f"SELECT message_content FROM {table} WHERE local_type=3 LIMIT 1")
    row = cur.fetchone()
    if row:
        data = row[0]
        # 检查是不是 zstd
        if data[:4] == b'\x28\xb5\x2f\xfd':
            dec = zstd.decompress(data)
            text = dec.decode('utf-8', errors='replace')
        else:
            text = data.decode('utf-8', errors='replace')
        print(f"\n表: {table}")
        print(f"图片 XML 前 500 字:")
        print(text[:500])
        break
