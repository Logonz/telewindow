package window

import (
	"log"
	"unsafe"

	"golang.org/x/sys/windows"
)

// SizeByPixel controls if the window should be resized by pixel or percentage if the resolution of the monitors is different
var SizeByPixel bool = false

// Globals
var (
	user32                 = windows.NewLazySystemDLL("user32.dll")
	procMoveWindow         = user32.NewProc("MoveWindow")
	procShowWindow         = user32.NewProc("ShowWindow")
	procGetWindowPlacement = user32.NewProc("GetWindowPlacement")
	// procSetWindowPos       = user32.NewProc("SetWindowPos")
	// procSetWindowPlacement = user32.NewProc("SetWindowPlacement")
	// procSendMessage        = user32.NewProc("SendMessageW")
	// procInvalidateRect     = user32.NewProc("InvalidateRect")
	// procUpdateWindow       = user32.NewProc("UpdateWindow")
)

const (
	// ShowWindow commands
	SW_MAXIMIZE = 3
	SW_RESTORE  = 9

	// ShowWindow flags
	SW_SHOWMAXIMIZED = 3
	SW_SHOWNORMAL    = 1
	SW_SHOWMINIMIZED = 2

	// SetWindowPos flags
	SWP_NOZORDER       = 0x0004
	SWP_NOACTIVATE     = 0x0010
	SWP_NOSENDCHANGING = 0x0400
	SWP_NOREDRAW       = 0x0008
	SWP_ASYNCWINDOWPOS = 0x4000
	SWP_NOSIZE         = 0x0001
	SWP_NOMOVE         = 0x0002

	// WM_SETREDRAW message
	WM_SETREDRAW = 0x000B

	// WPF
	WPF_SETMINPOSITION     = 0x0001
	WPF_RESTORETOMAXIMIZED = 0x0002
)

type WINDOWPLACEMENT struct {
	Length           uint32
	Flags            uint32
	ShowCmd          uint32
	PtMinPosition    Point
	PtMaxPosition    Point
	RcNormalPosition RECT
}

func MoveActiveWindow(direction int) {
	log.Printf("DEBUG: Entering MoveActiveWindow() with direction: %d\n", direction)
	activeWindow, err := GetActiveWindow()
	if err != nil {
		log.Println("DEBUG: Error getting active window:", err)
		return
	}

	rect, err := GetWindowRectWrapper(activeWindow)
	if err != nil {
		log.Println("DEBUG: Error getting window rect:", err)
		return
	}

	monitors, err := GetMonitors()
	if err != nil {
		log.Println("DEBUG: Error getting monitors:", err)
		return
	}

	if len(monitors) < 2 {
		log.Println("DEBUG: Only one monitor detected.")
		return
	}

	// Find the monitor that the window is currently on
	var currentMonitor *Monitor
	var maxOverlap int64 = 0

	for _, m := range monitors {
		overlap := calculateOverlap(rect, &m.Info.RCMonitor)
		if overlap > maxOverlap {
			maxOverlap = overlap
			currentMonitor = &m
		}
	}

	if currentMonitor == nil {
		log.Println("DEBUG: Current monitor not found.")
		return
	}
	log.Printf("DEBUG: Current monitor: %+v\n", currentMonitor.Info.RCMonitor)

	// Find the monitor in the desired direction
	targetMonitor := findTargetMonitor(monitors, currentMonitor, direction)
	if targetMonitor == nil {
		log.Println("DEBUG: No monitor found in the desired direction.")
		return
	}
	log.Printf("DEBUG: Target monitor: %+v\n", targetMonitor.Info.RCMonitor)

	// Calculate the new window position
	var newX int32
	var newY int32
	var newWidth int32
	var newHeight int32

	if SizeByPixel {
		// Calculate the window's current size and position pixel based
		newWidth = rect.Right - rect.Left
		newHeight = rect.Bottom - rect.Top

		relativeX := rect.Left - currentMonitor.Info.RCMonitor.Left
		relativeY := rect.Top - currentMonitor.Info.RCMonitor.Top

		// Pixel based calculation
		newX = targetMonitor.Info.RCMonitor.Left + relativeX
		newY = targetMonitor.Info.RCMonitor.Top + relativeY
	} else {
		// Calculate the percentage of the window's size relative to the current monitor
		currentMonitorWidth := float64(currentMonitor.Info.RCMonitor.Right - currentMonitor.Info.RCMonitor.Left)
		currentMonitorHeight := float64(currentMonitor.Info.RCMonitor.Bottom - currentMonitor.Info.RCMonitor.Top)
		windowWidth := float64(rect.Right - rect.Left)
		windowHeight := float64(rect.Bottom - rect.Top)

		widthPercentage := windowWidth / currentMonitorWidth
		heightPercentage := windowHeight / currentMonitorHeight

		// Calculate the new size based on the target monitor's dimensions
		targetMonitorWidth := float64(targetMonitor.Info.RCMonitor.Right - targetMonitor.Info.RCMonitor.Left)
		targetMonitorHeight := float64(targetMonitor.Info.RCMonitor.Bottom - targetMonitor.Info.RCMonitor.Top)

		newWidth = int32(widthPercentage * targetMonitorWidth)
		newHeight = int32(heightPercentage * targetMonitorHeight)

		// Calculate the new position
		relativeXPercentage := float64(rect.Left-currentMonitor.Info.RCMonitor.Left) / currentMonitorWidth
		relativeYPercentage := float64(rect.Top-currentMonitor.Info.RCMonitor.Top) / currentMonitorHeight

		// Percentage based calculation
		newX = targetMonitor.Info.RCMonitor.Left + int32(relativeXPercentage*targetMonitorWidth)
		newY = targetMonitor.Info.RCMonitor.Top + int32(relativeYPercentage*targetMonitorHeight)
	}

	log.Printf("DEBUG: New window position: x=%d, y=%d, width=%d, height=%d\n", newX, newY, newWidth, newHeight)

	maximized, err := IsActiveWindowMaximized(&activeWindow)
	if err != nil {
		log.Println("DEBUG: Error checking if window is maximized:", err)
		return
	}
	if maximized {
		log.Println("DEBUG: Window is maximized, restoring window.")
		RestoreActiveWindow(&activeWindow)

		// Shrink the window by 5% and the newX and newY to make it centered
		amount := 0.02 // Percentage
		newWidth = int32(float64(newWidth) * (1 - amount))
		newHeight = int32(float64(newHeight) * (1 - amount))
		newX = newX + int32(float64(newWidth)*amount/2)
		newY = newY + int32(float64(newHeight)*amount/2)
	}

	log.Println("DEBUG: Moving window.")
	// Move the window
	ret, _, err := procMoveWindow.Call(
		uintptr(activeWindow),
		uintptr(newX),
		uintptr(newY),
		uintptr(newWidth),
		uintptr(newHeight),
		1, // Repaint
	)
	if ret == 0 {
		log.Println("DEBUG: MoveWindow failed:", err)
		return
	}

	if maximized {
		log.Println("DEBUG: Window was maximized, maximizing window again.")
		MaximizeActiveWindow(&activeWindow)
	}

	log.Println("DEBUG: Window moved successfully.")
}

func SplitActiveWindow(direction int) {
	log.Println("DEBUG: Entering SplitWindow() with direction:", direction)
	// 1. Get the active window
	activeWindow, err := GetActiveWindow()
	if err != nil {
		log.Println("DEBUG: Error getting active window:", err)
		return
	}

	// 2. Get the monitor that the window is on
	monitors, err := GetMonitors()
	if err != nil {
		log.Println("DEBUG: Error getting monitors:", err)
		return
	}

	// Find the monitor that the window is currently on
	rect, err := GetWindowRectWrapper(activeWindow)
	if err != nil {
		log.Println("DEBUG: Error getting window rect:", err)
		return
	}

	var currentMonitor *Monitor
	var maxOverlap int64 = 0

	for _, m := range monitors {
		overlap := calculateOverlap(rect, &m.Info.RCMonitor)
		if overlap > maxOverlap {
			maxOverlap = overlap
			currentMonitor = &m
		}
	}

	if currentMonitor == nil {
		log.Println("DEBUG: Current monitor not found.")
		return
	}
	log.Printf("DEBUG: Current monitor: %+v\n", currentMonitor.Info.RCMonitor)

	// 3. Get the monitor's dimensions
	monitorRect := currentMonitor.Info.RCMonitor

	// 4. Calculate the new window position and size
	var newX, newY, newWidth, newHeight int32

	if direction == -1 { // Left
		newX = monitorRect.Left
		newY = monitorRect.Top
		newHeight = monitorRect.Bottom - monitorRect.Top
		newWidth = (monitorRect.Right - monitorRect.Left) / 2
	} else if direction == 1 { // Right
		newX = monitorRect.Left + (monitorRect.Right-monitorRect.Left)/2
		newY = monitorRect.Top
		newHeight = monitorRect.Bottom - monitorRect.Top
		newWidth = (monitorRect.Right - monitorRect.Left) / 2
	} else if direction == -2 { // Up
		newX = monitorRect.Left
		newY = monitorRect.Top
		newHeight = (monitorRect.Bottom - monitorRect.Top) / 2
		newWidth = monitorRect.Right - monitorRect.Left
	} else if direction == 2 { // Down
		newX = monitorRect.Left
		newY = monitorRect.Top + (monitorRect.Bottom-monitorRect.Top)/2
		newHeight = (monitorRect.Bottom - monitorRect.Top) / 2
		newWidth = monitorRect.Right - monitorRect.Left
	} else {
		log.Println("DEBUG: Invalid direction. -1, 1, -2, 2.")
		return
	}

	log.Printf("DEBUG: New window position: x=%d, y=%d, width=%d, height=%d\n", newX, newY, newWidth, newHeight)

	// 5. If window is maximized, restore it
	maximized, err := IsActiveWindowMaximized(&activeWindow)
	if err != nil {
		log.Println("DEBUG: Error checking if window is maximized:", err)
		return
	}
	if maximized {
		log.Println("DEBUG: Window is maximized, restoring window.")
		RestoreActiveWindow(&activeWindow)
	}

	// 6. Move and resize the window
	log.Println("DEBUG: Moving and resizing window.")
	ret, _, err := procMoveWindow.Call(
		uintptr(activeWindow),
		uintptr(newX),
		uintptr(newY),
		uintptr(newWidth),
		uintptr(newHeight),
		1, // Repaint
	)
	if ret == 0 {
		log.Println("DEBUG: MoveWindow failed:", err)
		return
	}

	// // Optionally, re-maximize if the window was maximized before
	// if maximized {
	// 	log.Println("DEBUG: Window was maximized, maximizing window again.")
	// 	MaximizeActiveWindow(&activeWindow)
	// }

	log.Println("DEBUG: Window moved successfully to", direction)
}

func MaximizeActiveWindow(specificWindow *windows.Handle) {
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

	ret, _, err := procShowWindow.Call(
		uintptr(window),
		uintptr(SW_MAXIMIZE),
	)
	if ret == 0 {
		log.Println("MaximizeActiveWindow", "DEBUG: ShowWindow failed:", err)
		return
	}
	log.Println("DEBUG: Window maximized successfully.")
}

func RestoreActiveWindow(specificWindow *windows.Handle) {
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

	ret, _, err := procShowWindow.Call(
		uintptr(window),
		uintptr(SW_RESTORE),
	)
	if ret == 0 {
		log.Println("RestoreActiveWindow", "DEBUG: ShowWindow failed:", ret, err)
		return
	}
	log.Println("DEBUG: Window restored successfully.")
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
