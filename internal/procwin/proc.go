// Package procwin 封装 Windows 进程内存访问，对上层隐藏 Win32 API 细节。
package procwin

import (
	"fmt"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Region 描述一块可读内存区域。
type Region struct {
	Addr uintptr
	Size uintptr
}

// Proc 代表一个打开的进程，用完要 Close。
type Proc struct {
	Handle windows.Handle
	pid    uint32
}

// FindMainWeixinPID 找 Weixin.exe 主进程（PID 最小的那个，不是 WeChatAppEx）。
func FindMainWeixinPID() (uint32, error) {
	snap, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return 0, err
	}
	defer windows.CloseHandle(snap)

	var pe windows.ProcessEntry32
	pe.Size = uint32(unsafe.Sizeof(pe))
	if err := windows.Process32First(snap, &pe); err != nil {
		return 0, err
	}

	var minPID uint32
	for {
		name := syscall.UTF16ToString(pe.ExeFile[:])
		if equalFold(name, "Weixin.exe") {
			if minPID == 0 || pe.ProcessID < minPID {
				minPID = pe.ProcessID
			}
		}
		if err := windows.Process32Next(snap, &pe); err != nil {
			break
		}
	}
	if minPID == 0 {
		return 0, fmt.Errorf("未找到 Weixin.exe 进程")
	}
	return minPID, nil
}

// Open 打开进程用于读内存。
func Open(pid uint32) (*Proc, error) {
	h, err := windows.OpenProcess(
		windows.PROCESS_VM_READ|windows.PROCESS_QUERY_INFORMATION,
		false, pid)
	if err != nil {
		return nil, fmt.Errorf("OpenProcess (需要管理员): %w", err)
	}
	return &Proc{Handle: h, pid: pid}, nil
}

func (p *Proc) Close() { windows.CloseHandle(p.Handle) }
func (p *Proc) PID() uint32 { return p.pid }

// Regions 枚举所有"可读、非 guard、大小合理"的内存区域。
func (p *Proc) Regions() ([]Region, error) {
	var out []Region
	var addr uintptr
	for {
		var mbi windows.MemoryBasicInformation
		if err := windows.VirtualQueryEx(p.Handle, addr, &mbi, unsafe.Sizeof(mbi)); err != nil {
			break
		}
		next := uintptr(mbi.BaseAddress) + uintptr(mbi.RegionSize)
		if mbi.State == windows.MEM_COMMIT &&
			mbi.Protect&windows.PAGE_GUARD == 0 &&
			(mbi.Protect&windows.PAGE_READWRITE != 0 ||
				mbi.Protect&windows.PAGE_WRITECOPY != 0 ||
				mbi.Protect&windows.PAGE_READONLY != 0) &&
			mbi.RegionSize > 0 && mbi.RegionSize <= 500*1024*1024 {
			out = append(out, Region{Addr: uintptr(mbi.BaseAddress), Size: uintptr(mbi.RegionSize)})
		}
		if next <= addr {
			break
		}
		addr = next
	}
	return out, nil
}

// Read 在指定地址读 len(buf) 字节，返回实际读到的字节数。
func (p *Proc) Read(addr uintptr, buf []byte) (int, error) {
	var read uintptr
	err := windows.ReadProcessMemory(p.Handle, addr, &buf[0], uintptr(len(buf)), &read)
	return int(read), err
}

func equalFold(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		ca, cb := a[i], b[i]
		if ca >= 'A' && ca <= 'Z' {
			ca += 32
		}
		if cb >= 'A' && cb <= 'Z' {
			cb += 32
		}
		if ca != cb {
			return false
		}
	}
	return true
}
