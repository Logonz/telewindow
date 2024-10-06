package window

import (
	"log"

	"golang.org/x/sys/windows"
)

// SizeByPixel controls if the window should be resized by pixel or percentage if the resolution of the monitors is different
var SizeByPixel bool = false

// Globals
var (
	user32         = windows.NewLazySystemDLL("user32.dll")
	procMoveWindow = user32.NewProc("MoveWindow")
)

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

	log.Println("DEBUG: Window moved successfully.")
}
