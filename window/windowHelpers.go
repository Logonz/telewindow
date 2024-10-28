package window

import (
	"fmt"
	"log"
	"unsafe"

	"golang.org/x/sys/windows"
)

func IsDWMCompositionEnabled() (bool, error) {
	var enabled int32
	hr, _, _ := dwmapi.NewProc("DwmIsCompositionEnabled").Call(
		uintptr(unsafe.Pointer(&enabled)),
	)
	if hr != 0 {
		return false, fmt.Errorf("DwmIsCompositionEnabled failed: HRESULT=0x%X", hr)
	}
	return enabled != 0, nil
}

func DisableWindowTransitions(hwnd windows.Handle) error {
	log.Println("DEBUG: Entering DisableWindowTransitions()")
	var value int32 = 1 // TRUE to disable transitions
	hr, _, _ := procDwmSetWindowAttribute.Call(
		uintptr(hwnd),
		uintptr(DWMWA_TRANSITIONS_FORCEDISABLED),
		uintptr(unsafe.Pointer(&value)),
		uintptr(unsafe.Sizeof(value)),
	)
	if hr != 0 {
		log.Printf("DEBUG: DwmSetWindowAttribute failed: HRESULT=0x%X", hr)
		return fmt.Errorf("DwmSetWindowAttribute failed: HRESULT=0x%X", hr)
	}
	log.Println("DEBUG: Window transitions disabled successfully.")
	return nil
}

func EnableWindowTransitions(hwnd windows.Handle) error {
	log.Println("DEBUG: Entering EnableWindowTransitions()")
	var value int32 = 0 // FALSE to enable transitions
	hr, _, _ := procDwmSetWindowAttribute.Call(
		uintptr(hwnd),
		uintptr(DWMWA_TRANSITIONS_FORCEDISABLED),
		uintptr(unsafe.Pointer(&value)),
		uintptr(unsafe.Sizeof(value)),
	)
	if hr != 0 {
		log.Printf("DEBUG: DwmSetWindowAttribute failed: HRESULT=0x%X", hr)
		return fmt.Errorf("DwmSetWindowAttribute failed: HRESULT=0x%X", hr)
	}
	log.Println("DEBUG: Window transitions enabled successfully.")
	return nil
}

func IsActiveWindowMaximized(specificWindow *windows.Handle) (bool, error) {
	log.Println("DEBUG: Entering IsActiveWindowMaximized()")
	var window windows.Handle
	if specificWindow == nil {
		activeWindow, err := GetActiveWindow()
		if err != nil {
			log.Println("DEBUG: Error getting active window:", err)
			return false, err
		} else {
			window = activeWindow
		}
	} else {
		window = *specificWindow
	}

	var wp WINDOWPLACEMENT
	wp.Length = uint32(unsafe.Sizeof(wp))

	ret, _, err := procGetWindowPlacement.Call(
		uintptr(window),
		uintptr(unsafe.Pointer(&wp)),
	)
	if ret == 0 {
		log.Println("DEBUG: GetWindowPlacement failed:", err)
		return false, err
	}

	log.Printf("DEBUG: Window show command: %d\n", wp.ShowCmd)
	return wp.ShowCmd == SW_SHOWMAXIMIZED, nil
}

func IsActiveWindowMinimized(specificWindow *windows.Handle) (bool, error) {
	log.Println("DEBUG: Entering IsActiveWindowMinimized()")
	var window windows.Handle
	if specificWindow == nil {
		activeWindow, err := GetActiveWindow()
		if err != nil {
			log.Println("DEBUG: Error getting active window:", err)
			return false, err
		} else {
			window = activeWindow
		}
	} else {
		window = *specificWindow
	}

	var wp WINDOWPLACEMENT
	wp.Length = uint32(unsafe.Sizeof(wp))

	ret, _, err := procGetWindowPlacement.Call(
		uintptr(window),
		uintptr(unsafe.Pointer(&wp)),
	)
	if ret == 0 {
		log.Println("DEBUG: GetWindowPlacement failed:", err)
		return false, err
	}

	log.Printf("DEBUG: Window show command: %d\n", wp.ShowCmd)
	return wp.ShowCmd == SW_SHOWMINIMIZED, nil
}

func MaximizeActiveWindow(specificWindow *windows.Handle, disableAnimation bool) {
	log.Println("DEBUG: Entering MaximizeActiveWindow()")
	var window windows.Handle
	if specificWindow == nil {
		activeWindow, err := GetActiveWindow()
		if err != nil {
			log.Println("DEBUG: Error getting active window:", err)
			return
		} else {
			window = activeWindow
		}
	} else {
		window = *specificWindow
	}

	// Check if DWM composition is enabled
	isEnabled, err := IsDWMCompositionEnabled()
	if err != nil {
		log.Println("DEBUG: Error checking DWM composition:", err)
	}
	if isEnabled && disableAnimation {
		// Disable window transitions
		err = DisableWindowTransitions(window)
		if err != nil {
			log.Println("DEBUG: Error disabling window transitions:", err)
			// Proceed anyway
		}
	}

	ret, _, err := procShowWindow.Call(
		uintptr(window),
		uintptr(SW_MAXIMIZE),
	)
	if ret == 0 {
		log.Println("MaximizeActiveWindow", "DEBUG: ShowWindow failed:", err)
		return
	}

	if isEnabled {
		err = EnableWindowTransitions(window)
		if err != nil {
			log.Println("DEBUG: Error enabling window transitions:", err)
			// Proceed anyway
		}
	}

	log.Println("DEBUG: Window maximized successfully.")
}

func RestoreActiveWindow(specificWindow *windows.Handle, disableAnimation bool) {
	log.Println("DEBUG: Entering RestoreActiveWindow()")
	var window windows.Handle
	if specificWindow == nil {
		activeWindow, err := GetActiveWindow()
		if err != nil {
			log.Println("DEBUG: Error getting active window:", err)
			return
		} else {
			window = activeWindow
		}
	} else {
		window = *specificWindow
	}

	// Check if DWM composition is enabled
	isEnabled, err := IsDWMCompositionEnabled()
	if err != nil {
		log.Println("DEBUG: Error checking DWM composition:", err)
	}
	if isEnabled && disableAnimation {
		// Disable window transitions
		err = DisableWindowTransitions(window)
		if err != nil {
			log.Println("DEBUG: Error disabling window transitions:", err)
			// Proceed anyway
		}
	}

	ret, _, err := procShowWindow.Call(
		uintptr(window),
		uintptr(SW_RESTORE),
	)
	if ret == 0 {
		log.Println("RestoreActiveWindow", "DEBUG: ShowWindow failed:", ret, err)
		return
	}

	if isEnabled && disableAnimation {
		err = EnableWindowTransitions(window)
		if err != nil {
			log.Println("DEBUG: Error enabling window transitions:", err)
			// Proceed anyway
		}
	}

	log.Println("DEBUG: Window restored successfully.")
}
