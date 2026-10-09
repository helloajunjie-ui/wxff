import sqlite3
c = sqlite3.connect(r'F:\wxff\data\backup\_plain\contact__contact.db')
# 总数
total = c.execute("SELECT count(*) FROM contact").fetchone()[0]
print(f"总记录: {total}")
# local_type 分布
for row in c.execute("SELECT local_type, count(*) FROM contact GROUP BY local_type"):
    print(f"  local_type={row[0]}: {row[1]}")
# 好友（非群非公众号）
friends = c.execute("SELECT count(*) FROM contact WHERE username NOT LIKE '%@chatroom' AND username NOT LIKE 'gh_%'").fetchone()[0]
print(f"非群非公众号: {friends}")
# 看看 local_type=0 是什么
print("\nlocal_type=0 的前5条:")
for row in c.execute("SELECT username, nick_name FROM contact WHERE local_type=0 LIMIT 5"):
    print(f"  {row[0]}: {row[1]}")
print("\nlocal_type=1 的前5条:")
for row in c.execute("SELECT username, nick_name FROM contact WHERE local_type=1 LIMIT 5"):
    print(f"  {row[0]}: {row[1]}")
