package window

import (
	"encoding/json"
	"os"
)

type KeyBinding struct {
	Ctrl  bool   `json:"ctrl"`
	Alt   bool   `json:"alt"`
	Shift bool   `json:"shift"`
	Key   string `json:"key"`
}

const (
	VK_CONTROL  = "VK_CONTROL"
	VK_LCONTROL = "VK_LCONTROL"
	VK_RCONTROL = "VK_RCONTROL"
	VK_MENU     = "VK_MENU"
	VK_LMENU    = "VK_LMENU"
	VK_RMENU    = "VK_RMENU"
	VK_SHIFT    = "VK_SHIFT"
	VK_LSHIFT   = "VK_LSHIFT"
	VK_RSHIFT   = "VK_RSHIFT"
)

func (kb KeyBinding) Down(keyDownMap map[string]bool) bool {
	ctrlDown := keyDownMap[VK_LCONTROL] || keyDownMap[VK_RCONTROL] || keyDownMap[VK_CONTROL]
	altDown := keyDownMap[VK_LMENU] || keyDownMap[VK_RMENU] || keyDownMap[VK_MENU]
	shiftDown := keyDownMap[VK_LSHIFT] || keyDownMap[VK_RSHIFT] || keyDownMap[VK_SHIFT]
	if kb.Ctrl == !ctrlDown {
		return false
	}
	if kb.Alt == !altDown {
		return false
	}
	if kb.Shift == !shiftDown {
		return false
	}
	if !keyDownMap[kb.Key] {
		return false
	}
	return true
}

type Config struct {
	AllowNonAdmin bool `json:"allowNonAdmin"`
	SizeByPixel   bool `json:"sizeByPixel"`
	KeyBindings   struct {
		MoveRight      KeyBinding `json:"moveRight"`
		MoveLeft       KeyBinding `json:"moveLeft"`
		MoveUp         KeyBinding `json:"moveUp"`
		MoveDown       KeyBinding `json:"moveDown"`
		ToggleMaximize KeyBinding `json:"toggleMaximize"`
		SplitLeft      KeyBinding `json:"splitLeft"`
		SplitRight     KeyBinding `json:"splitRight"`
		SplitUp        KeyBinding `json:"splitUp"`
		SplitDown      KeyBinding `json:"splitDown"`
		RestoreWindow  KeyBinding `json:"restoreWindow"`
	} `json:"keyBindings"`
}

func LoadConfig() (*Config, error) {
	data, err := os.ReadFile("config.json")
	if err != nil {
		return nil, err
	}

	var config Config
	err = json.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}
