package main

import (
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/moutend/go-hook/pkg/keyboard"
	"github.com/moutend/go-hook/pkg/types"
	"golang.org/x/sys/windows"
)

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
	fmt.Printf("DEBUG: Entering MoveActiveWindow() with direction: %d\n", direction)
	activeWindow, err := GetActiveWindow()
	if err != nil {
		fmt.Println("DEBUG: Error getting active window:", err)
		return
	}

	rect, err := GetWindowRectWrapper(activeWindow)
	if err != nil {
		fmt.Println("DEBUG: Error getting window rect:", err)
		return
	}

	monitors, err := GetMonitors()
	if err != nil {
		fmt.Println("DEBUG: Error getting monitors:", err)
		return
	}

	if len(monitors) < 2 {
		fmt.Println("DEBUG: Only one monitor detected.")
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
		fmt.Println("DEBUG: Current monitor not found.")
		return
	}
	fmt.Printf("DEBUG: Current monitor: %+v\n", currentMonitor.Info.RCMonitor)

	var targetMonitor *Monitor
	if direction == -1 { // Move left
		for _, m := range monitors {
			if m.Info.RCMonitor.Right == currentMonitor.Info.RCMonitor.Left {
				targetMonitor = &m
				break
			}
		}
	} else if direction == 1 { // Move right
		for _, m := range monitors {
			if m.Info.RCMonitor.Left == currentMonitor.Info.RCMonitor.Right {
				targetMonitor = &m
				break
			}
		}
	} else if direction == -2 { // Move up
		for _, m := range monitors {
			if m.Info.RCMonitor.Bottom == currentMonitor.Info.RCMonitor.Top {
				targetMonitor = &m
				break
			}
		}
	} else if direction == 2 { // Move down
		for _, m := range monitors {
			if m.Info.RCMonitor.Top == currentMonitor.Info.RCMonitor.Bottom {
				targetMonitor = &m
				break
			}
		}
	}

	if targetMonitor == nil {
		fmt.Println("DEBUG: Target monitor not found.")
		return
	}
	fmt.Printf("DEBUG: Target monitor: %+v\n", targetMonitor.Info.RCMonitor)

	width := rect.Right - rect.Left
	height := rect.Bottom - rect.Top

	relativeX := rect.Left - currentMonitor.Info.RCMonitor.Left
	relativeY := rect.Top - currentMonitor.Info.RCMonitor.Top

	newX := targetMonitor.Info.RCMonitor.Left + relativeX
	newY := targetMonitor.Info.RCMonitor.Top + relativeY

	fmt.Printf("DEBUG: New window position: x=%d, y=%d, width=%d, height=%d\n", newX, newY, width, height)

	ret, _, err := procMoveWindow.Call(
		uintptr(activeWindow),
		uintptr(newX),
		uintptr(newY),
		uintptr(width),
		uintptr(height),
		1, // Repaint
	)
	if ret == 0 {
		fmt.Println("DEBUG: MoveWindow failed:", err)
		return
	}

	fmt.Println("DEBUG: Window moved successfully.")
}

func main() {
	// Detect if we are running as admininstrator
	if !isRunningAsAdmin() {
		fmt.Println("WARNING: Not running as administrator. Some features may not work correctly. Such as keyboard hooking in administrative windows.")
	}

	go keyboardHook()

	fmt.Println("Window manager is running. Press Ctrl+C to exit.")

	// Handle graceful shutdown on Ctrl+C
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<-c
	fmt.Println("\nExiting...")
}

func keyboardHook() error {
	// Buffer size is depends on your need. The 100 is placeholder value.
	keyboardChan := make(chan types.KeyboardEvent, 100)

	if err := keyboard.Install(nil, keyboardChan); err != nil {
		return err
	}

	defer keyboard.Uninstall()

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)

	keyDownMap := make(map[string]bool)

	for {
		select {
		case <-time.After(5 * time.Minute):
			fmt.Println("Received timeout signal")
			return nil
		case <-signalChan:
			fmt.Println("Received shutdown signal")
			return nil
		case k := <-keyboardChan:
			// fmt.Printf("Received %v %v\n", k.Message, k.VKCode)

			down := fmt.Sprint(k.Message) == WM_KEYDOWN || fmt.Sprint(k.Message) == WM_SYSKEYDOWN
			up := fmt.Sprint(k.Message) == WM_KEYUP

			ctrlDown := keyDownMap[VK_LCONTROL] || keyDownMap[VK_RCONTROL]

			if down && !keyDownMap[fmt.Sprint(k.VKCode)] {
				// fmt.Printf("Down %v\n", k.VKCode)
				keyDownMap[fmt.Sprint(k.VKCode)] = true
				// If control and right arrow are pressed
				if ctrlDown && keyDownMap[VK_RIGHT] {
					fmt.Println("Hotkey Move Right Pressed")
					MoveActiveWindow(1)
				} else if ctrlDown && keyDownMap[VK_LEFT] {
					fmt.Println("Hotkey Move Left Pressed")
					MoveActiveWindow(-1)
				} else if ctrlDown && keyDownMap[VK_UP] {
					fmt.Println("Hotkey Move Up Pressed")
					MoveActiveWindow(-2)
				} else if ctrlDown && keyDownMap[VK_DOWN] {
					fmt.Println("Hotkey Move Down Pressed")
					MoveActiveWindow(2)
				}
			} else if up && keyDownMap[fmt.Sprint(k.VKCode)] {
				// fmt.Printf("Up %v\n", k.VKCode)
				keyDownMap[fmt.Sprint(k.VKCode)] = false
			}
			continue

		}
	}
}
