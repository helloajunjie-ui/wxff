import sqlite3
c = sqlite3.connect(r'F:\wxff\data\backup\_plain\message__message_0.db')
# 傲慢的泪的表是 Msg_fd323731058f3a8c3d9fd23c9e31a9f6
print("local_type 分布:")
for row in c.execute("SELECT local_type, count(*) FROM Msg_fd323731058f3a8c3d9fd23c9e31a9f6 GROUP BY local_type"):
    print(f"  {row[0]}: {row[1]}")
