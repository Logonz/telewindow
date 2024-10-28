package main

import (
	_ "embed"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"telewindow/lumberjack"
	"telewindow/window"
	"time"

	"github.com/getlantern/systray"
	"github.com/moutend/go-hook/pkg/keyboard"
	"github.com/moutend/go-hook/pkg/types"
)

//go:embed assets/dock-window-light.ico
var iconData []byte

var signalChan chan os.Signal = make(chan os.Signal, 1)

// Direction constants
const (
	LeftDirection  = -1
	RightDirection = 1
	UpDirection    = -2
	DownDirection  = 2
)

// Constants for Windows API
const (
	WM_KEYDOWN    = "WM_KEYDOWN"
	WM_KEYUP      = "WM_KEYUP"
	WM_SYSKEYDOWN = "WM_SYSKEYDOWN"
)

func main() {
	// Create a multi-writer that writes to both file and stdout
	multiWriter := io.MultiWriter(os.Stdout, &lumberjack.Logger{
		Filename:   "./telewindow.log",
		MaxSize:    1, // megabytes
		MaxBackups: 5,
		MaxAge:     28,    //days
		Compress:   false, // disabled by default
	})

	// Set the output of the default logger to the multi-writer
	log.SetOutput(multiWriter)

	config, err := window.LoadConfig()
	if err != nil {
		log.Println("Error loading config:", err)
		os.Exit(1)
	}

	// Set the global SizeByPixel variable
	window.SizeByPixel = config.SizeByPixel

	// Detect if we are running as admininstrator
	if !window.IsRunningAsAdmin() {
		if !config.AllowNonAdmin {
			fmt.Println("This application needs to run as administrator.")
			err := window.RelaunchAsAdmin()
			if err != nil {
				fmt.Println("Failed to restart as administrator:", err)
				os.Exit(1)
			}
			os.Exit(0) // Exit current instance
		} else {
			log.Println("WARNING: Not running as administrator. Some features may not work correctly. Such as keyboard hooking in administrative windows.")
		}
	}

	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)

	systray.Run(func() {
		// Pass in the config to the onReady function
		onReady(config)
	}, onExit)
}

func onReady(config *window.Config) {
	// Set the icon (optional)
	systray.SetIcon(iconData)
	systray.SetTooltip("TeleWindow Service")
	mQuit := systray.AddMenuItem("Quit", "Quit the application")
	go func() {
		select {
		case <-mQuit.ClickedCh:
			log.Println("Quit menu item clicked.")
			signalChan <- syscall.SIGTERM
		case <-signalChan:
			log.Println("Interrupt signal received.")
		}
		log.Println("Quitting TeleWindow.")
		systray.Quit()
		os.Exit(0)
	}()

	log.Println("Window manager is running. Press Ctrl+C to exit.")
	go keyboardHook(signalChan, config)
}

func onExit() {
	// Cleanup tasks
	log.Println("TeleWindow exited.")
}

func keyboardHook(signalChan chan os.Signal, config *window.Config) error {
	// Buffer size is depends on your need. The 100 is placeholder value.
	keyboardChan := make(chan types.KeyboardEvent, 100)

	if err := keyboard.Install(nil, keyboardChan); err != nil {
		return err
	}

	defer keyboard.Uninstall()

	keyDownMap := make(map[string]bool)
	var keyDownMapMutex sync.Mutex

	var lastMove time.Time = time.Now()

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

			keyDownMapMutex.Lock()
			if down && !keyDownMap[key] {
				// log.Printf("Down %v\n", k.VKCode)
				keyDownMap[key] = true

				// If control and right arrow are pressed
				if time.Since(lastMove) > 50*time.Millisecond {
					if config.KeyBindings.MoveRight.Down(keyDownMap) {
						log.Println("Hotkey Move Right Pressed")
						window.MoveActiveWindow(RightDirection)
						lastMove = time.Now()
					} else if config.KeyBindings.MoveLeft.Down(keyDownMap) {
						log.Println("Hotkey Move Left Pressed")
						window.MoveActiveWindow(LeftDirection)
						lastMove = time.Now()
					} else if config.KeyBindings.MoveUp.Down(keyDownMap) {
						log.Println("Hotkey Move Up Pressed")
						window.MoveActiveWindow(UpDirection)
						lastMove = time.Now()
					} else if config.KeyBindings.MoveDown.Down(keyDownMap) {
						log.Println("Hotkey Move Down Pressed")
						window.MoveActiveWindow(DownDirection)
						lastMove = time.Now()
					} else if config.KeyBindings.ToggleMaximize.Down(keyDownMap) {
						log.Println("Hotkey Toggle Maximize Pressed")
						maximized, err := window.IsActiveWindowMaximized(nil)
						if err != nil {
							log.Println("Error checking if window is maximized:", err)
							continue
						}
						if maximized {
							log.Println("Window is maximized, restoring window.")
							window.RestoreActiveWindow(nil)
						} else {
							log.Println("Window is not maximized, maximizing window.")
							window.MaximizeActiveWindow(nil)
						}

						lastMove = time.Now()
					}
				}
			} else if up && keyDownMap[key] {
				// log.Printf("Up %v\n", k.VKCode)
				keyDownMap[key] = false
			}
			keyDownMapMutex.Unlock()
			continue

		}
	}
}
