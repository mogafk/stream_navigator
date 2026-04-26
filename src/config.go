package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
)

type FeatureConfig struct {
	Feature        string   `json:"feature"`
	DebugKey       string   `json:"debugKey"`
	Cooldown       string   `json:"cooldown"`
	ChatCommand    []string `json:"chatCommand"`
	Time           string   `json:"time"`
	Distance       int32    `json:"distance"`
	ModificatorKey string   `json:"modificatorKey"`
}

type Config struct {
	TwitchLink string          `json:"twitchLink"`
	Mute       bool            `json:"mute"`
	Features   []FeatureConfig `json:"features"`
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
	cfgTurnModVK    uint32
	cfgTurnModOtherVK uint32
	cfgTurnModOtherUp uint32
	cfgDebug180Key  string
	cfgDebugStunKey string
	cfgChatCmds180  []string
	cfgChatCmdsStun []string
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

	var (
		turnKey, stunKey               uint32
		stunTime                       = 60 * time.Second
		cooldown180, cooldownStun      time.Duration
		turnDist                       int32
		turnModDown, turnModUp, turnModVK uint32
		turnModOtherVK, turnModOtherUp uint32
		debug180Key, debugStunKey      string
		chatCmds180, chatCmdsStun      []string
	)

	for _, f := range c.Features {
		switch f.Feature {
		case "stun":
			if stunKey, err = parseKeyName(f.DebugKey); err != nil {
				return err
			}
			debugStunKey = f.DebugKey
			chatCmdsStun = f.ChatCommand
			if f.Time != "" {
				if stunTime, err = time.ParseDuration(f.Time); err != nil {
					return fmt.Errorf("stun: invalid time: %w", err)
				}
			}
			if f.Cooldown != "" {
				if cooldownStun, err = time.ParseDuration(f.Cooldown); err != nil {
					return fmt.Errorf("stun: invalid cooldown: %w", err)
				}
			}
		case "turn":
			if turnKey, err = parseKeyName(f.DebugKey); err != nil {
				return err
			}
			debug180Key = f.DebugKey
			chatCmds180 = f.ChatCommand
			turnDist = f.Distance
			if f.Cooldown != "" {
				if cooldown180, err = time.ParseDuration(f.Cooldown); err != nil {
					return fmt.Errorf("turn: invalid cooldown: %w", err)
				}
			}
			switch strings.ToUpper(f.ModificatorKey) {
			case "PMB":
				turnModDown, turnModUp, turnModVK = mouseEventLeftDown, mouseEventLeftUp, 0x01
				turnModOtherVK, turnModOtherUp = 0x02, mouseEventRightUp
			case "SMB":
				turnModDown, turnModUp, turnModVK = mouseEventRightDown, mouseEventRightUp, 0x02
				turnModOtherVK, turnModOtherUp = 0x01, mouseEventLeftUp
			case "":
				// no button held during turn
			default:
				return fmt.Errorf("turn: invalid modificatorKey %q — use PMB or SMB", f.ModificatorKey)
			}
		default:
			return fmt.Errorf("unknown feature %q", f.Feature)
		}
	}

	// All values valid — apply atomically
	cfg = c
	cfgTurnKey = turnKey
	cfgStunKey = stunKey
	cfgStunTime = stunTime
	cfgCooldown180 = cooldown180
	cfgCooldownStun = cooldownStun
	cfgTurnDist = turnDist
	cfgTurnModDown = turnModDown
	cfgTurnModUp = turnModUp
	cfgTurnModVK = turnModVK
	cfgTurnModOtherVK = turnModOtherVK
	cfgTurnModOtherUp = turnModOtherUp
	cfgDebug180Key = debug180Key
	cfgDebugStunKey = debugStunKey
	cfgChatCmds180 = chatCmds180
	cfgChatCmdsStun = chatCmdsStun
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
