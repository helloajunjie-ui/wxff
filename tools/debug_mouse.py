"""
鼠标坐标调试工具
用法: python debug_mouse.py
把鼠标移到目标位置，按 Ctrl+C 退出，会打印坐标
"""
import pyautogui
import time

print("把鼠标移到目标位置，按 Ctrl+C 退出")
print("每 0.5 秒打印一次坐标...")
try:
    while True:
        x, y = pyautogui.position()
        print(f"鼠标位置: x={x}, y={y}", end='\r')
        time.sleep(0.5)
except KeyboardInterrupt:
    x, y = pyautogui.position()
    print(f"\n最终坐标: x={x}, y={y}")
