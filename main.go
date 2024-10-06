package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"syscall"
	"telewindow/lumberjack"

	"github.com/moutend/go-hook/pkg/keyboard"
	"github.com/moutend/go-hook/pkg/types"
	"golang.org/x/sys/windows"
)

var PixelBased bool = false

// Constants for Windows API
const (
	WH_KEYBOARD_LL = "WH_KEYBOARD_LL"
	WM_KEYDOWN     = "WM_KEYDOWN"
	WM_KEYUP       = "WM_KEYUP"
	WM_SYSKEYDOWN  = "WM_SYSKEYDOWN"
	VK_LEFT        = "VK_LEFT"
	VK_RIGHT       = "VK_RIGHT"
	VK_UP          = "VK_UP"
	VK_DOWN        = "VK_DOWN"
	VK_CONTROL     = "VK_CONTROL"
	VK_LCONTROL    = "VK_LCONTROL"
	VK_RCONTROL    = "VK_RCONTROL"
	VK_MENU        = "VK_MENU"
	HC_ACTION      = "HC_ACTION"
	WM_QUIT        = "WM_QUIT"
)

// Direction constants
const (
	LeftDirection  = -1
	RightDirection = 1
	UpDirection    = -2
	DownDirection  = 2
)

// Globals
var (
	user32                  = windows.NewLazySystemDLL("user32.dll")
	procGetForegroundWindow = user32.NewProc("GetForegroundWindow")
	procGetWindowRect       = user32.NewProc("GetWindowRect")
	procMoveWindow          = user32.NewProc("MoveWindow")
	procEnumDisplayMonitors = user32.NewProc("EnumDisplayMonitors")
	procGetMonitorInfo      = user32.NewProc("GetMonitorInfoW")
)

func isRunningAsAdmin() bool {
	var sid *windows.SID
	err := windows.AllocateAndInitializeSid(
		&windows.SECURITY_NT_AUTHORITY,
		2,
		windows.SECURITY_BUILTIN_DOMAIN_RID,
		windows.DOMAIN_ALIAS_RID_ADMINS,
		0, 0, 0, 0, 0, 0,
		&sid)
	if err != nil {
		return false
	}
	defer windows.FreeSid(sid)

	token := windows.Token(0)
	member, err := token.IsMember(sid)
	if err != nil {
		return false
	}
	return member
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

	if PixelBased {
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

func main() {
	// Create a multi-writer that writes to both file and stdout
	multiWriter := io.MultiWriter(os.Stdout, &lumberjack.Logger{
		Filename:   "./telewindow.log",
		MaxSize:    1, // megabytes
		MaxBackups: 3,
		MaxAge:     28,    //days
		Compress:   false, // disabled by default
	})

	// Set the output of the default logger to the multi-writer
	log.SetOutput(multiWriter)

	// Detect if we are running as admininstrator
	if !isRunningAsAdmin() {
		log.Println("WARNING: Not running as administrator. Some features may not work correctly. Such as keyboard hooking in administrative windows.")
	}

	log.Println("Window manager is running. Press Ctrl+C to exit.")
	keyboardHook()

	log.Println("\nExiting...")
}

func keyboardHook() error {
	// Buffer size is depends on your need. The 100 is placeholder value.
	keyboardChan := make(chan types.KeyboardEvent, 100)

	if err := keyboard.Install(nil, keyboardChan); err != nil {
		return err
	}

	defer keyboard.Uninstall()

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)

	keyDownMap := make(map[string]bool)

	for {
		select {
		case <-signalChan:
			log.Println("Received shutdown signal")
			return nil
		case k := <-keyboardChan:
			// log.Printf("Received %v %v\n", k.Message, k.VKCode)
			msg := fmt.Sprint(k.Message)
			key := fmt.Sprint(k.VKCode)

			down := msg == WM_KEYDOWN || msg == WM_SYSKEYDOWN
			up := msg == WM_KEYUP

			ctrlDown := keyDownMap[VK_LCONTROL] || keyDownMap[VK_RCONTROL]

			if down && !keyDownMap[key] {
				// log.Printf("Down %v\n", k.VKCode)
				keyDownMap[key] = true
				// If control and right arrow are pressed
				if ctrlDown && keyDownMap[VK_RIGHT] {
					log.Println("Hotkey Move Right Pressed")
					MoveActiveWindow(RightDirection)
				} else if ctrlDown && keyDownMap[VK_LEFT] {
					log.Println("Hotkey Move Left Pressed")
					MoveActiveWindow(LeftDirection)
				} else if ctrlDown && keyDownMap[VK_UP] {
					log.Println("Hotkey Move Up Pressed")
					MoveActiveWindow(UpDirection)
				} else if ctrlDown && keyDownMap[VK_DOWN] {
					log.Println("Hotkey Move Down Pressed")
					MoveActiveWindow(DownDirection)
				}
			} else if up && keyDownMap[key] {
				// log.Printf("Up %v\n", k.VKCode)
				keyDownMap[key] = false
			}
			continue

		}
	}
}
