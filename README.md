# TeleWindow - Windows Multi-Monitor Window Manager

This project provides a lightweight window management tool for Windows, allowing users to easily move windows between multiple monitors while maintaining their size and relative position.

## Features

- Move active windows between monitors using keyboard shortcuts
- Preserves window size and relative position when moving
- Supports multiple monitor configurations
- Works with Windows OS

## How It Works

The application runs in the background and listens for specific keyboard combinations:

- Ctrl + Right Arrow: Move window to the monitor on the right
- Ctrl + Left Arrow: Move window to the monitor on the left
- Ctrl + Up Arrow: Move window to the monitor above
- Ctrl + Down Arrow: Move window to the monitor below

When a hotkey is pressed, the active window is "teleported" to the corresponding monitor, maintaining its size and relative position.

## Installation

1. Clone this repository
2. Ensure you have Go installed on your system
3. Build the project:
```bash
go build -o telewindow.exe
```
4. Run the executable (as an administrator if you want the keyboard shortcuts to work across all applications)

## Usage

After starting the application, it will run in the background. Use the keyboard shortcuts mentioned above to move windows between monitors.

To exit the application, press Ctrl+C in the terminal where it's running.

## Requirements

- Windows operating system
- Multiple monitors

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

[Add your chosen license here]

## Acknowledgements

This project uses the following libraries:
- github.com/moutend/go-hook
- golang.org/x/sys/windows

## Disclaimer

This software interacts with Windows system calls and hooks into keyboard events. Use at your own risk.
