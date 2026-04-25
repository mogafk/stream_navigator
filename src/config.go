package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
)

type Config struct {
	Debug180Key      string `json:"debug180Key"`
	DebugStunKey     string `json:"debugStunKey"`
	StunTime         string `json:"stunTime"`
	TurnDistance     int32  `json:"turnDistance"`
	TurnModificator  string `json:"turnModificator"`
	Mute             bool   `json:"mute"`
	Cooldown180      string `json:"cooldown180"`
	CooldownStun     string `json:"cooldownStun"`
	TwitchLink       string `json:"twitchLink"`
}

var (
	cfg             Config
	cfgTurnKey      uint32
	cfgStunKey      uint32
	cfgStunTime     time.Duration
	cfgCooldown180  time.Duration
	cfgCooldownStun time.Duration
	cfgTwitchChan   string
	cfgTurnDist     int32
	cfgTurnModDown  uint32
	cfgTurnModUp    uint32
)

var keyNames = map[string]uint32{
	"F1": 0x70, "F2": 0x71, "F3": 0x72, "F4": 0x73,
	"F5": 0x74, "F6": 0x75, "F7": 0x76, "F8": 0x77,
	"F9": 0x78, "F10": 0x79, "F11": 0x7A, "F12": 0x7B,
}

// parseConfig reads and validates config.json, applying values to globals.
// Returns an error without modifying globals if anything is invalid.
func parseConfig() error {
	data, err := os.ReadFile("config.json")
	if err != nil {
		return err
	}
	var c Config
	if err = json.Unmarshal(data, &c); err != nil {
		return err
	}

	turnKey, err := parseKeyName(c.Debug180Key)
	if err != nil {
		return err
	}
	stunKey, err := parseKeyName(c.DebugStunKey)
	if err != nil {
		return err
	}

	stunTime := 60 * time.Second
	if c.StunTime != "" {
		if stunTime, err = time.ParseDuration(c.StunTime); err != nil {
			return fmt.Errorf("invalid stunTime: %w", err)
		}
	}

	var cooldown180, cooldownStun time.Duration
	if c.Cooldown180 != "" {
		if cooldown180, err = time.ParseDuration(c.Cooldown180); err != nil {
			return fmt.Errorf("invalid cooldown180: %w", err)
		}
	}
	if c.CooldownStun != "" {
		if cooldownStun, err = time.ParseDuration(c.CooldownStun); err != nil {
			return fmt.Errorf("invalid cooldownStun: %w", err)
		}
	}

	var turnModDown, turnModUp uint32
	switch strings.ToUpper(c.TurnModificator) {
	case "PMB":
		turnModDown, turnModUp = mouseEventLeftDown, mouseEventLeftUp
	case "SMB":
		turnModDown, turnModUp = mouseEventRightDown, mouseEventRightUp
	case "":
		// no button held during turn
	default:
		return fmt.Errorf("invalid turnModificator %q — use PMB or SMB", c.TurnModificator)
	}

	// All values valid — apply atomically
	cfg = c
	cfgTurnKey = turnKey
	cfgStunKey = stunKey
	cfgStunTime = stunTime
	cfgCooldown180 = cooldown180
	cfgCooldownStun = cooldownStun
	cfgTurnDist = c.TurnDistance
	cfgTurnModDown = turnModDown
	cfgTurnModUp = turnModUp
	cfgTwitchChan = channelFromURL(c.TwitchLink)
	return nil
}

// loadConfig is used at startup — exits on error.
func loadConfig() {
	if err := parseConfig(); err != nil {
		fmt.Fprintln(os.Stderr, "config error:", err)
		os.Exit(1)
	}
}

// reloadConfig is used by the file watcher — logs error and keeps old config.
func reloadConfig() {
	if err := parseConfig(); err != nil {
		fmt.Fprintln(os.Stderr, "[config] reload failed:", err)
		return
	}
	fmt.Println("[config] reloaded")
}

// watchConfig polls config.json every second and reloads on change.
func watchConfig() {
	info, err := os.Stat("config.json")
	if err != nil {
		return
	}
	lastMod := info.ModTime()
	for range time.Tick(time.Second) {
		info, err := os.Stat("config.json")
		if err != nil || !info.ModTime().After(lastMod) {
			continue
		}
		lastMod = info.ModTime()
		reloadConfig()
	}
}

func parseKeyName(name string) (uint32, error) {
	if name == "" {
		return 0, nil
	}
	vk, ok := keyNames[strings.ToUpper(name)]
	if !ok {
		return 0, fmt.Errorf("unknown key %q — supported: F1–F12", name)
	}
	return vk, nil
}

func channelFromURL(url string) string {
	s := strings.TrimRight(url, "/")
	if idx := strings.LastIndex(s, "/"); idx >= 0 {
		return s[idx+1:]
	}
	return s
}

func keyLabel(name string) string {
	if name == "" {
		return "(disabled)"
	}
	return name
}
