import sqlite3
c = sqlite3.connect(r'F:\wxff\data\backup\_plain\contact__contact.db')
cur = c.execute('PRAGMA table_info(contact)')
cols = [r[1] for r in cur.fetchall()]
print("COLUMNS:", cols)
# 看一行头像字段
cur2 = c.execute("SELECT * FROM contact LIMIT 1")
row = cur2.fetchone()
for i, col in enumerate(cols):
    val = str(row[i])[:100] if row[i] else ""
    print(f"  {col}: {val}")
