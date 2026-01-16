// main_cli.go
//go:build cli
// +build cli

package main

import (
	"io"
	"log"
	"os"
	"telewindow/lumberjack"
	"telewindow/window"
)

func main() {

	// Create a multi-writer that writes to both file and stdout
	multiWriter := io.MultiWriter(&lumberjack.Logger{
		Filename:   "./telewindow.log",
		MaxSize:    1, // megabytes
		MaxBackups: 5,
		MaxAge:     28,    //days
		Compress:   false, // disabled by default
	})

	// Set the output of the default logger to the multi-writer
	log.SetOutput(multiWriter)

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

	log.Println("Received command:", command)

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
			exit(1)
		}
		if maximized {
			window.RestoreActiveWindow(nil)
		} else {
			window.MaximizeActiveWindow(nil)
		}
	case "-NoOp":
		// Do nothing
		log.Println("No operation performed.")
		exit(0)
	default:
		log.Println("Unknown command:", command)
		exit(1)
	}

	exit(0)
}

func exit(code int) {
	log.Println("")
	log.Println("")
	os.Exit(code)
}
