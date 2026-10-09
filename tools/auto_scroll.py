"""
自动刷朋友圈完整流程
用法: python auto_scroll.py <nickname> <scroll_times>
"""
import sys
import time
import ctypes
import pyautogui
import psutil

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

def activate_window(hwnd):
    """可靠地激活窗口：先 AttachThreadInput 绕过 Windows 焦点限制"""
    user32 = ctypes.windll.user32
    kernel32 = ctypes.windll.kernel32

    # 1. 先还原窗口
    user32.ShowWindow(hwnd, 9)  # SW_RESTORE
    time.sleep(0.5)

    # 2. AttachThreadInput 绕过焦点限制
    foreground_hwnd = user32.GetForegroundWindow()
    foreground_tid = user32.GetWindowThreadProcessId(foreground_hwnd, None)
    current_tid = kernel32.GetCurrentThreadId()
    target_tid = user32.GetWindowThreadProcessId(hwnd, None)

    user32.AttachThreadInput(current_tid, foreground_tid, True)
    user32.AttachThreadInput(current_tid, target_tid, True)

    user32.SetForegroundWindow(hwnd)
    user32.BringWindowToTop(hwnd)

    user32.AttachThreadInput(current_tid, foreground_tid, False)
    user32.AttachThreadInput(current_tid, target_tid, False)

    time.sleep(1)

    # 3. 验证：检查是不是真的前置了
    active_hwnd = user32.GetForegroundWindow()
    if active_hwnd == hwnd:
        print(f"✓ 微信窗口已前置 (hwnd={hwnd})")
        return True
    else:
        print(f"⚠️ 窗口前置失败，当前前台窗口 hwnd={active_hwnd}")
        return False

def main():
    nickname = sys.argv[1] if len(sys.argv) > 1 else ""
    scroll_times = int(sys.argv[2]) if len(sys.argv) > 2 else 30

    print(f"=== 开始刷 {nickname} 的朋友圈 ===")
    print(f"滚动次数: {scroll_times}")
    print("10 秒后开始，期间不要动鼠标键盘...")
    print("-" * 40)
    time.sleep(10)

    # 步骤1：找窗口
    print("[1/5] 查找微信窗口...")
    hwnd = find_wechat_window()
    if hwnd == 0:
        print("✗ 没找到微信窗口")
        return
    print(f"  找到微信窗口 hwnd={hwnd}")

    # 步骤2：前置窗口
    print("[2/5] 前置微信窗口...")
    if not activate_window(hwnd):
        print("⚠️ 窗口前置可能失败，继续尝试...")

    # 步骤3：点通讯录图标（左侧第三个图标，从截图算大概 y=170）
    print("[3/5] 点通讯录图标...")
    pyautogui.click(20, 170)
    time.sleep(1.5)

    # 步骤4：搜索好友
    print(f"[4/5] 搜索好友: {nickname}...")
    pyautogui.click(100, 50)
    time.sleep(0.5)
    pyautogui.hotkey('ctrl', 'a')
    pyautogui.press('backspace')
    time.sleep(0.3)
    pyautogui.typewrite(nickname, interval=0.05)
    time.sleep(1.5)
    pyautogui.press('enter')
    time.sleep(2)

    # 步骤5：滚动朋友圈
    print(f"[5/5] 滚动朋友圈 {scroll_times} 次...")
    screen_w, screen_h = pyautogui.size()
    pyautogui.moveTo(int(screen_w * 0.85), int(screen_h * 0.5))
    time.sleep(0.5)

    for i in range(scroll_times):
        pyautogui.scroll(-500)
        print(f"  滚动 {i+1}/{scroll_times}")
        time.sleep(1.5)

    print("-" * 40)
    print("✓ 刷完了！")

if __name__ == "__main__":
    main()
