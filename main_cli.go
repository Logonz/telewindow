// main_cli.go
//go:build cli
// +build cli

package main

import (
	"log"
	"os"
	"telewindow/window"
)

func main() {
	if len(os.Args) < 2 {
		log.Println("Usage: telewindow [command]")
		log.Println("Commands:")
		log.Println("  -Right         Move window right")
		log.Println("  -Left          Move window left")
		log.Println("  -Up            Move window up")
		log.Println("  -Down          Move window down")
		log.Println("  -Maximize      Maximize active window")
		log.Println("  -Restore       Restore active window")
		log.Println("  -SplitRight    Split window right")
		log.Println("  -SplitLeft     Split window left")
		log.Println("  -SplitUp       Split window up")
		log.Println("  -SplitDown     Split window down")
		log.Println("  -ToggleMaximize Toggle maximize/restore")
		log.Println("  -NoOp 					No Operation (Used to bind over existing shortcuts)")
		os.Exit(0)
	}

	command := os.Args[1]

	switch command {
	case "-Right":
		window.MoveActiveWindow(RightDirection)
	case "-Left":
		window.MoveActiveWindow(LeftDirection)
	case "-Up":
		window.MoveActiveWindow(UpDirection)
	case "-Down":
		window.MoveActiveWindow(DownDirection)
	case "-Maximize":
		window.MaximizeActiveWindow(nil)
	case "-Restore":
		window.RestoreActiveWindow(nil)
	case "-SplitRight":
		window.SplitActiveWindow(RightDirection)
	case "-SplitLeft":
		window.SplitActiveWindow(LeftDirection)
	case "-SplitUp":
		window.SplitActiveWindow(UpDirection)
	case "-SplitDown":
		window.SplitActiveWindow(DownDirection)
	case "-ToggleMaximize":
		maximized, err := window.IsActiveWindowMaximized(nil)
		if err != nil {
			log.Println("Error checking if window is maximized:", err)
			os.Exit(1)
		}
		if maximized {
			window.RestoreActiveWindow(nil)
		} else {
			window.MaximizeActiveWindow(nil)
		}
	case "-NoOp":
		// Do nothing
		os.Exit(0)
	default:
		log.Println("Unknown command:", command)
		os.Exit(1)
	}
}
