package window

import (
	"fmt"
	"log"
	"sync"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	// Load DLLs and procedures
	moduser32                    = windows.NewLazySystemDLL("user32.dll")
	modpsapi                     = windows.NewLazySystemDLL("psapi.dll")
	procGetWindowThreadProcessId = moduser32.NewProc("GetWindowThreadProcessId")
	procGetModuleBaseNameW       = modpsapi.NewProc("GetModuleBaseNameW")

	// Map to store window sizes by process name
	processWindowSizes = make(map[string]*RECT)
	mutex              = &sync.Mutex{}
)

// GetProcessName retrieves the process name associated with a window handle.
func GetProcessName(hwnd windows.Handle) (string, error) {
	var processID uint32
	ret, _, _ := procGetWindowThreadProcessId.Call(
		uintptr(hwnd),
		uintptr(unsafe.Pointer(&processID)),
	)
	if ret == 0 || processID == 0 {
		return "", fmt.Errorf("GetWindowThreadProcessId failed")
	}

	hProcess, err := windows.OpenProcess(
		windows.PROCESS_QUERY_LIMITED_INFORMATION|windows.PROCESS_VM_READ,
		false,
		processID,
	)
	if err != nil {
		return "", fmt.Errorf("OpenProcess failed: %v", err)
	}
	defer windows.CloseHandle(hProcess)

	var exeName [windows.MAX_PATH]uint16
	ret, _, _ = procGetModuleBaseNameW.Call(
		uintptr(hProcess),
		0,
		uintptr(unsafe.Pointer(&exeName[0])),
		uintptr(len(exeName)),
	)
	if ret == 0 {
		return "", fmt.Errorf("GetModuleBaseNameW failed")
	}

	name := syscall.UTF16ToString(exeName[:])
	return name, nil
}

// RecordWindowSize saves the window size associated with the process name.
func RecordWindowSize(hwnd windows.Handle, rect *RECT) error {
	processName, err := GetProcessName(hwnd)
	if err != nil {
		return err
	}

	mutex.Lock()
	defer mutex.Unlock()

	// Save a copy of rect
	savedRect := &RECT{
		Left:   rect.Left,
		Top:    rect.Top,
		Right:  rect.Right,
		Bottom: rect.Bottom,
	}

	processWindowSizes[processName] = savedRect

	log.Println("DEBUG: Saved window size for process", processName, ":", savedRect)

	return nil
}

// GetWindowSize retrieves the saved window size for the process associated with the window handle.
func GetWindowSize(hwnd windows.Handle) (*RECT, error) {
	processName, err := GetProcessName(hwnd)
	if err != nil {
		return nil, err
	}

	mutex.Lock()
	defer mutex.Unlock()

	rect, exists := processWindowSizes[processName]
	if !exists {
		return nil, fmt.Errorf("No saved window size for process %s", processName)
	}

	return rect, nil
}
