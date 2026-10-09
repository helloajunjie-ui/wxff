"""M1 验证：从 Weixin.exe 内存提取 4.x 数据库 key。"""
import sys
import json

try:
    from wxutil.utils import get_wechat4_key
except Exception as e:
    print(f"[导入失败] {e}")
    sys.exit(1)

try:
    key = get_wechat4_key()
    print(f"[OK] 拿到 key: {key}")
    # 保存到文件供后续 Go 用
    with open(r"F:\wxff\data\key.txt", "w", encoding="utf-8") as f:
        f.write(key if isinstance(key, str) else key.hex())
    print("[OK] key 已写入 F:\\wxff\\data\\key.txt")
except Exception as e:
    import traceback
    print(f"[失败] {e}")
    traceback.print_exc()
    sys.exit(2)
