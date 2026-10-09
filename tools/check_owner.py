import sqlite3
c = sqlite3.connect(r'F:\wxff\data\backup\_plain\contact__contact.db')
# 看所有字段
cols = [r[1] for r in c.execute('PRAGMA table_info(contact)').fetchall()]
print("所有字段:", cols)
# 看有没有 owner / account 相关字段
owner_cols = [col for col in cols if 'owner' in col.lower() or 'account' in col.lower() or 'wxid' in col.lower()]
print("归属相关字段:", owner_cols)
# 看看 local_type 分布对应谁
print("\nlocal_type=1 的 username 前缀:")
for row in c.execute("SELECT DISTINCT substr(username,1,10) FROM contact WHERE local_type=1 LIMIT 10"):
    print(f"  {row[0]}")
