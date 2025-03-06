package window

import (
	"log"

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
	procSetWindowPos       = user32.NewProc("SetWindowPos")
	// procSetWindowPlacement = user32.NewProc("SetWindowPlacement")
	// procSendMessage        = user32.NewProc("SendMessageW")
	// procInvalidateRect     = user32.NewProc("InvalidateRect")
	// procUpdateWindow       = user32.NewProc("UpdateWindow")

	dwmapi                    = windows.NewLazySystemDLL("dwmapi.dll")
	procDwmSetWindowAttribute = dwmapi.NewProc("DwmSetWindowAttribute")
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
	SWP_SHOWWINDOW     = 0x0040

	// WM_SETREDRAW message
	WM_SETREDRAW = 0x000B

	// WPF
	WPF_SETMINPOSITION     = 0x0001
	WPF_RESTORETOMAXIMIZED = 0x0002

	//DWMWA
	DWMWA_TRANSITIONS_FORCEDISABLED = 3
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

	maximized, err := IsActiveWindowMaximized(&activeWindow)
	if err != nil {
		log.Println("DEBUG: Error checking if window is maximized:", err)
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

	log.Println("DEBUG: Window is maximized, restoring window.")
	RestoreActiveWindow(&activeWindow, true)

	// Refresh the window rect after restoring
	rect, err = GetWindowRectWrapper(activeWindow)
	if err != nil {
		log.Println("DEBUG: Error getting window rect:", err)
		return
	}

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
		MaximizeActiveWindow(&activeWindow, true)
	}

	log.Println("DEBUG: Window moved successfully.")
}

// ? SplitActiveWindow splits the active window in the specified direction
// ? direction: -1 = left, 1 = right, -2 = up, 2 = down
func SplitActiveWindow(direction int) {
	log.Println("DEBUG: Entering SplitWindow() with direction:", direction)
	// 1. Get the active window
	activeWindow, err := GetActiveWindow()
	if err != nil {
		log.Println("DEBUG: Error getting active window:", err)
		return
	}

	maximized, err := IsActiveWindowMaximized(&activeWindow)
	if err != nil {
		log.Println("DEBUG: Error checking if window is maximized:", err)
		return
	}
	if maximized {
		RestoreActiveWindow(&activeWindow, true)
		MaximizeActiveWindow(&activeWindow, true)
		// log.Println(GetWindowRectWrapper(activeWindow))
	} else {
		MaximizeActiveWindow(&activeWindow, true)
		// log.Println(GetWindowRectWrapper(activeWindow))
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

	// Calculate the difference between the window's position and the monitor's position
	// This is because they are minus values so we need to add them to the new position
	// ? Basically it is a quick way to do monitorWidth - windowWidth, monitorHeight - windowHeight etc
	rectDiff := *rect
	rectDiff.Left = rectDiff.Left - currentMonitor.Info.RCMonitor.Left
	rectDiff.Right = rectDiff.Right - currentMonitor.Info.RCMonitor.Right
	rectDiff.Top = rectDiff.Top - currentMonitor.Info.RCMonitor.Top
	rectDiff.Bottom = rectDiff.Bottom - currentMonitor.Info.RCMonitor.Bottom

	log.Println("DEBUG: Rect diff:", rectDiff)

	// 3. Get the monitor's dimensions
	monitorRect := currentMonitor.Info.RCMonitor
	log.Println("DEBUG: Monitor rect:", monitorRect)

	// 4. Calculate the new window position and size
	var newX, newY, newWidth, newHeight int32

	if direction == -1 { // Left
		newX = rect.Left
		newY = rect.Top
		newHeight = rect.Bottom - rect.Top
		newWidth = (monitorRect.Right - monitorRect.Left) / 2
		// Adjust to make the windows take up half the screen (When maximized the border is invisible, so we need to adjust)
		newWidth += rectDiff.Right * 2
	} else if direction == 1 { // Right
		newX = monitorRect.Left + (rect.Right-rect.Left)/2
		newY = rect.Top
		newHeight = rect.Bottom - rect.Top
		newWidth = (monitorRect.Right - monitorRect.Left) / 2
		// Adjust to make the windows take up half the screen (When maximized the border is invisible, so we need to adjust)
		newX += rectDiff.Left * 2
		newWidth += rectDiff.Right * 2
	} else if direction == -2 { // Up
		// ? These are a bit funky... there seems to be a bar at the top of the window
		// ? For VSCode its white, firefox is black so unknown how to handle this
		newX = rect.Left
		// Pin to the top of the monitor (similar to how "Left" pins to rect.Left)
		newY = monitorRect.Top
		// Keep original window width
		newWidth = rect.Right - rect.Left
		// Use half monitor height
		newHeight = (monitorRect.Bottom - monitorRect.Top) / 2
		// Adjust for taskbar
		// newY -= rectDiff.Top
	} else if direction == 2 { // Down
		// ? These are a bit funky... there seems to be a bar at the top of the window
		// ? For VSCode its white, firefox is black so unknown how to handle this
		newX = rect.Left
		// Start halfway down the monitor (similar to how "Right" starts halfway across)
		newY = monitorRect.Top + (rect.Bottom-rect.Top)/2
		// Keep original window width
		newWidth = rect.Right - rect.Left
		// Use half monitor height
		newHeight = (monitorRect.Bottom - monitorRect.Top) / 2
		// Adjust for taskbar
		newHeight += rectDiff.Bottom
	} else {
		log.Println("DEBUG: Invalid direction. -1, 1, -2, 2.")
		return
	}

	log.Printf("DEBUG: New window position: x=%d, y=%d, width=%d, height=%d\n", newX, newY, newWidth, newHeight)

	// Fixes a bug with the window not resizing properly (VScode suffers from this)
	count := 0
	for {
		rect, err = GetWindowRectWrapper(activeWindow)
		if err != nil {
			log.Println("DEBUG: Error getting window rect:", err)
			return
		}
		log.Println(rect, rect.Right-rect.Left, rect.Bottom-rect.Top)

		// if rect.Right-rect.Left != newWidth || rect.Bottom-rect.Top != newHeight {
		if newX != rect.Left || newY != rect.Top || newWidth != rect.Right-rect.Left || newHeight != rect.Bottom-rect.Top {
			ret, _, err := procSetWindowPos.Call(
				uintptr(activeWindow),
				0,
				uintptr(newX),
				uintptr(newY),
				uintptr(newWidth),
				uintptr(newHeight),
				SWP_SHOWWINDOW,
			)
			if ret == 0 {
				log.Println("DEBUG: MoveWindow failed:", err)
				return
			}
		} else {
			break
		}
		count++
		if count >= 20 {
			// Safety break
			break
		}
	}

	log.Println("DEBUG: Window moved successfully to", direction)
}
