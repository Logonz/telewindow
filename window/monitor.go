package window

import (
	"fmt"
	"log"
	"math"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	procGetForegroundWindow = user32.NewProc("GetForegroundWindow")
	procGetWindowRect       = user32.NewProc("GetWindowRect")
	procEnumDisplayMonitors = user32.NewProc("EnumDisplayMonitors")
	procGetMonitorInfo      = user32.NewProc("GetMonitorInfoW")
)

// Monitor information
type Monitor struct {
	HMonitor windows.Handle
	Info     MONITORINFO
	Center   Point
}

// Structures
type Point struct {
	X, Y float64
}

type RECT struct {
	Left, Top, Right, Bottom int32
}

type MONITORINFO struct {
	CbSize    uint32
	RCMonitor RECT
	RCWork    RECT
	DwFlags   uint32
}

var directionVectors = map[int]Point{
	-1: {X: -1, Y: 0}, // Left
	1:  {X: 1, Y: 0},  // Right
	-2: {X: 0, Y: -1}, // Up
	2:  {X: 0, Y: 1},  // Down
}

func calculateMonitorCenter(mi MONITORINFO) Point {
	centerX := float64(mi.RCMonitor.Left+mi.RCMonitor.Right) / 2
	centerY := float64(mi.RCMonitor.Top+mi.RCMonitor.Bottom) / 2
	return Point{X: centerX, Y: centerY}
}

func calculateOverlap(windowRect *RECT, monitorRect *RECT) int64 {
	left := max(windowRect.Left, monitorRect.Left)
	top := max(windowRect.Top, monitorRect.Top)
	right := min(windowRect.Right, monitorRect.Right)
	bottom := min(windowRect.Bottom, monitorRect.Bottom)

	if left < right && top < bottom {
		return int64(right-left) * int64(bottom-top)
	}
	return 0
}

func GetMonitors() ([]Monitor, error) {
	// log.Println("DEBUG: Entering GetMonitors()")
	var monitors []Monitor

	enumProc := syscall.NewCallback(func(hMonitor windows.Handle, hdcMonitor windows.Handle, lprcMonitor *RECT, lParam uintptr) uintptr {
		// log.Printf("DEBUG: Enumerating monitor: %v\n", hMonitor)
		var mi MONITORINFO
		mi.CbSize = uint32(unsafe.Sizeof(mi))
		ret, _, _ := procGetMonitorInfo.Call(
			uintptr(hMonitor),
			uintptr(unsafe.Pointer(&mi)),
		)
		if ret == 0 {
			log.Println("DEBUG: GetMonitorInfo failed, continuing enumeration")
			return 1 // Continue enumeration
		}
		monitors = append(monitors, Monitor{
			HMonitor: hMonitor,
			Info:     mi,
			Center:   calculateMonitorCenter(mi),
		})
		// log.Printf("DEBUG: Added monitor: %+v\n", mi)
		return 1 // Continue enumeration
	})

	ret, _, err := procEnumDisplayMonitors.Call(
		0,
		0,
		enumProc,
		0,
	)
	if ret == 0 {
		log.Printf("DEBUG: EnumDisplayMonitors failed: %v\n", err)
		return nil, fmt.Errorf("EnumDisplayMonitors failed: %v", err)
	}

	log.Printf("DEBUG: GetMonitors() found %d monitors\n", len(monitors))
	return monitors, nil
}

func GetActiveWindow() (windows.Handle, error) {
	// log.Println("DEBUG: Entering GetActiveWindow()")
	ret, _, err := procGetForegroundWindow.Call()
	if ret == 0 {
		log.Printf("DEBUG: GetForegroundWindow failed: %v\n", err)
		return 0, fmt.Errorf("GetForegroundWindow failed: %v", err)
	}
	log.Printf("DEBUG: Active window handle: %v\n", windows.Handle(ret))
	return windows.Handle(ret), nil
}

func GetWindowRectWrapper(hwnd windows.Handle) (*RECT, error) {
	// log.Printf("DEBUG: Entering GetWindowRectWrapper() for handle: %v\n", hwnd)
	var rect RECT
	ret, _, err := procGetWindowRect.Call(
		uintptr(hwnd),
		uintptr(unsafe.Pointer(&rect)),
	)
	if ret == 0 {
		log.Printf("DEBUG: GetWindowRect failed: %v\n", err)
		return nil, fmt.Errorf("GetWindowRect failed: %v", err)
	}
	// log.Printf("DEBUG: Window rect: %+v\n", rect)
	return &rect, nil
}

func findTargetMonitor(monitors []Monitor, currentMonitor *Monitor, direction int) *Monitor {
	dirVec, exists := directionVectors[direction]
	if !exists {
		return nil
	}

	var candidates []Monitor

	for _, monitor := range monitors {
		if monitor.HMonitor == currentMonitor.HMonitor {
			continue // Skip the current monitor
		}

		// Vector from current monitor to the other monitor
		vecToMonitor := Point{
			X: monitor.Center.X - currentMonitor.Center.X,
			Y: monitor.Center.Y - currentMonitor.Center.Y,
		}

		// Normalize the vector to monitor
		mag := math.Sqrt(vecToMonitor.X*vecToMonitor.X + vecToMonitor.Y*vecToMonitor.Y)
		if mag == 0 {
			continue // Skip if magnitude is zero
		}
		normVecToMonitor := Point{
			X: vecToMonitor.X / mag,
			Y: vecToMonitor.Y / mag,
		}

		// Calculate dot product between direction vector and vector to monitor
		dot := dirVec.X*normVecToMonitor.X + dirVec.Y*normVecToMonitor.Y

		// Consider monitors that are in roughly the same direction (e.g., within 45 degrees)
		if dot >= math.Cos(math.Pi/4) { // Cosine of 45 degrees
			candidates = append(candidates, monitor)
		}
	}

	// If no candidates, return nil
	if len(candidates) == 0 {
		return nil
	}

	// Find the nearest monitor among candidates
	var targetMonitor *Monitor
	minDistance := math.MaxFloat64

	for _, monitor := range candidates {
		distance := math.Hypot(
			monitor.Center.X-currentMonitor.Center.X,
			monitor.Center.Y-currentMonitor.Center.Y,
		)
		if distance < minDistance {
			minDistance = distance
			targetMonitor = &monitor
		}
	}

	return targetMonitor
}
