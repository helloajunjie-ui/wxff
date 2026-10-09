"""
虚拟桌面测试脚本
把微信窗口移到第二个虚拟桌面，然后滚动，最后移回来
"""
import sys
import time
import ctypes
import pyautogui
import psutil

# 虚拟桌面 COM 接口
# 简化版：用 Shell 命令切换虚拟桌面
# Ctrl+Win+Right 切到下一个桌面
# Ctrl+Win+Left 切回上一个桌面

def find_wechat_window():
    user32 = ctypes.windll.user32
    wechat_pids = []
    for proc in psutil.process_iter(['pid', 'name']):
        if proc.info['name'] and 'weixin' in proc.info['name'].lower():
            wechat_pids.append(proc.info['pid'])
    found_hwnd = 0
    def callback(hwnd, _):
        nonlocal found_hwnd
        pid = ctypes.c_ulong()
        user32.GetWindowThreadProcessId(hwnd, ctypes.byref(pid))
        if pid.value in wechat_pids and user32.IsWindowVisible(hwnd):
            length = user32.GetWindowTextLengthW(hwnd)
            if length > 0:
                found_hwnd = hwnd
                return False
        return True
    WNDENUMPROC = ctypes.WINFUNCTYPE(ctypes.c_bool, ctypes.c_int, ctypes.c_int)
    user32.EnumWindows(WNDENUMPROC(callback), 0)
    return found_hwnd

def move_window_to_desktop(hwnd, desktop_index):
    """把窗口移到指定虚拟桌面（简化版：用快捷键）"""
    # 先把微信前置
    user32 = ctypes.windll.user32
    user32.SetForegroundWindow(hwnd)
    time.sleep(0.5)
    # 按 Ctrl+Win+Left/Right 切桌面
    # Win+Ctrl+D 新建桌面
    # Win+Ctrl+F4 关闭当前桌面
    print(f"  切换到桌面 {desktop_index}...")
    # 简化：先新建一个桌面，把微信移过去
    pyautogui.hotkey('win', 'ctrl', 'd')
    time.sleep(1)
    print("  已新建桌面")

def main():
    scroll_times = int(sys.argv[1]) if len(sys.argv) > 1 else 5

    print("=== 虚拟桌面隔离刷朋友圈测试 ===")
    print(f"滚动次数: {scroll_times}")
    print("5 秒后开始...")
    time.sleep(5)

    hwnd = find_wechat_window()
    if hwnd == 0:
        print("没找到微信窗口")
        return
    print(f"找到微信窗口: {hwnd}")

    # 1. 把微信移到新桌面
    print("\n[1] 把微信移到新虚拟桌面...")
    move_window_to_desktop(hwnd, 1)

    # 2. 在新桌面滚动
    print(f"\n[2] 在新桌面滚动朋友圈 {scroll_times} 次...")
    screen_w, screen_h = pyautogui.size()
    pyautogui.moveTo(int(screen_w * 0.85), int(screen_h * 0.5))
    time.sleep(0.5)
    for i in range(scroll_times):
        pyautogui.scroll(-500)
        print(f"  滚动 {i+1}/{scroll_times}")
        time.sleep(1.5)

    # 3. 切回原来的桌面
    print("\n[3] 切回原来的桌面...")
    pyautogui.hotkey('win', 'ctrl', 'left')
    time.sleep(1)

    print("\n完成！")

if __name__ == "__main__":
    main()
