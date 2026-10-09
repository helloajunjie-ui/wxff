import sqlite3
c = sqlite3.connect(r'F:\wxff\data\backup\_plain\message__message_0.db')
tables = c.execute("SELECT name FROM sqlite_master WHERE type='table' AND name LIKE 'Msg_%'").fetchall()
print(f"有聊天记录的会话: {len(tables)}")
# 取前10个看看
for t in tables[:10]:
    print(f"  {t[0]}")
