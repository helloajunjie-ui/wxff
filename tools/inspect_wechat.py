"""
用 UI Automation 找微信控件
"""
import uiautomation as auto
import psutil

# 找 Weixin.exe 的 PID
wechat_pids = []
for proc in psutil.process_iter(['pid', 'name']):
    if proc.info['name'] and 'weixin' in proc.info['name'].lower():
        wechat_pids.append(proc.info['pid'])
print(f"Weixin.exe PIDs: {wechat_pids}")

# 遍历所有顶层窗口，找属于 Weixin.exe 的
print("\n=== 找微信窗口 ===")
root = auto.GetRootControl()
for win in root.GetChildren():
    pid = win.ProcessId
    if pid in wechat_pids:
        print(f"找到窗口: {win.Name} (PID={pid}, ClassName={win.ClassName})")
        print(f"  位置: {win.BoundingRectangle}")
        print("\n  === 子控件 ===")
        for child in win.GetChildren()[:10]:
            print(f"    {child.ControlTypeName}: {child.Name}")
