package main

import (
	"fmt"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Monitor information
type Monitor struct {
	HMonitor windows.Handle
	Info     MONITORINFO
}

// Structures
type POINT struct {
	X, Y int32
}

type RECT struct {
	Left, Top, Right, Bottom int32
}

type MONITORINFO struct {
	CbSize    uint32
	RCMonitor RECT
	RCWork    RECT
	DwFlags   uint32
}

func GetMonitors() ([]Monitor, error) {
	fmt.Println("DEBUG: Entering GetMonitors()")
	var monitors []Monitor

	enumProc := syscall.NewCallback(func(hMonitor windows.Handle, hdcMonitor windows.Handle, lprcMonitor *RECT, lParam uintptr) uintptr {
		fmt.Printf("DEBUG: Enumerating monitor: %v\n", hMonitor)
		var mi MONITORINFO
		mi.CbSize = uint32(unsafe.Sizeof(mi))
		ret, _, _ := procGetMonitorInfo.Call(
			uintptr(hMonitor),
			uintptr(unsafe.Pointer(&mi)),
		)
		if ret == 0 {
			fmt.Println("DEBUG: GetMonitorInfo failed, continuing enumeration")
			return 1 // Continue enumeration
		}
		monitors = append(monitors, Monitor{
			HMonitor: hMonitor,
			Info:     mi,
		})
		fmt.Printf("DEBUG: Added monitor: %+v\n", mi)
		return 1 // Continue enumeration
	})

	ret, _, err := procEnumDisplayMonitors.Call(
		0,
		0,
		enumProc,
		0,
	)
	if ret == 0 {
		fmt.Printf("DEBUG: EnumDisplayMonitors failed: %v\n", err)
		return nil, fmt.Errorf("EnumDisplayMonitors failed: %v", err)
	}

	fmt.Printf("DEBUG: GetMonitors() found %d monitors\n", len(monitors))
	return monitors, nil
}

func GetActiveWindow() (windows.Handle, error) {
	fmt.Println("DEBUG: Entering GetActiveWindow()")
	ret, _, err := procGetForegroundWindow.Call()
	if ret == 0 {
		fmt.Printf("DEBUG: GetForegroundWindow failed: %v\n", err)
		return 0, fmt.Errorf("GetForegroundWindow failed: %v", err)
	}
	fmt.Printf("DEBUG: Active window handle: %v\n", windows.Handle(ret))
	return windows.Handle(ret), nil
}

func GetWindowRectWrapper(hwnd windows.Handle) (*RECT, error) {
	fmt.Printf("DEBUG: Entering GetWindowRectWrapper() for handle: %v\n", hwnd)
	var rect RECT
	ret, _, err := procGetWindowRect.Call(
		uintptr(hwnd),
		uintptr(unsafe.Pointer(&rect)),
	)
	if ret == 0 {
		fmt.Printf("DEBUG: GetWindowRect failed: %v\n", err)
		return nil, fmt.Errorf("GetWindowRect failed: %v", err)
	}
	fmt.Printf("DEBUG: Window rect: %+v\n", rect)
	return &rect, nil
}

func calculateOverlap(windowRect *RECT, monitorRect *RECT) int64 {
	left := max(windowRect.Left, monitorRect.Left)
	top := max(windowRect.Top, monitorRect.Top)
	right := min(windowRect.Right, monitorRect.Right)
	bottom := min(windowRect.Bottom, monitorRect.Bottom)

	if left < right && top < bottom {
		return int64(right-left) * int64(bottom-top)
	}
	return 0
}

func max(a, b int32) int32 {
	if a > b {
		return a
	}
	return b
}

func min(a, b int32) int32 {
	if a < b {
		return a
	}
	return b
}
