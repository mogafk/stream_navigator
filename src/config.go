package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync/atomic"
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

type turnEntry struct {
	key        uint32
	debugKey   string
	dist       int32
	cooldown   time.Duration
	chatCmds   []string
	modDown    uint32
	modUp      uint32
	modVK      uint32
	modOtherVK uint32
	modOtherUp uint32
	lastNano   atomic.Int64
}

type stunEntry struct {
	key      uint32
	debugKey string
	duration time.Duration
	cooldown time.Duration
	chatCmds []string
	lastNano atomic.Int64
}

var (
	cfg           Config
	cfgTwitchChan string
	cfgTurns      []*turnEntry
	cfgStuns      []*stunEntry
)

var keyNames = map[string]uint32{
	"F1": 0x70, "F2": 0x71, "F3": 0x72, "F4": 0x73,
	"F5": 0x74, "F6": 0x75, "F7": 0x76, "F8": 0x77,
	"F9": 0x78, "F10": 0x79, "F11": 0x7A, "F12": 0x7B,
}

func parseConfig() error {
	data, err := os.ReadFile("config.json")
	if err != nil {
		return err
	}
	var c Config
	if err = json.Unmarshal(data, &c); err != nil {
		return err
	}

	var turns []*turnEntry
	var stuns []*stunEntry

	for _, f := range c.Features {
		switch f.Feature {
		case "turn":
			t := &turnEntry{
				debugKey: f.DebugKey,
				dist:     f.Distance,
				chatCmds: f.ChatCommand,
			}
			if t.key, err = parseKeyName(f.DebugKey); err != nil {
				return err
			}
			if f.Cooldown != "" {
				if t.cooldown, err = time.ParseDuration(f.Cooldown); err != nil {
					return fmt.Errorf("turn: invalid cooldown: %w", err)
				}
			}
			switch strings.ToUpper(f.ModificatorKey) {
			case "PMB":
				t.modDown, t.modUp, t.modVK = mouseEventLeftDown, mouseEventLeftUp, 0x01
				t.modOtherVK, t.modOtherUp = 0x02, mouseEventRightUp
			case "SMB":
				t.modDown, t.modUp, t.modVK = mouseEventRightDown, mouseEventRightUp, 0x02
				t.modOtherVK, t.modOtherUp = 0x01, mouseEventLeftUp
			case "":
			default:
				return fmt.Errorf("turn: invalid modificatorKey %q — use PMB or SMB", f.ModificatorKey)
			}
			turns = append(turns, t)

		case "stun":
			s := &stunEntry{
				debugKey: f.DebugKey,
				duration: 60 * time.Second,
				chatCmds: f.ChatCommand,
			}
			if s.key, err = parseKeyName(f.DebugKey); err != nil {
				return err
			}
			if f.Time != "" {
				if s.duration, err = time.ParseDuration(f.Time); err != nil {
					return fmt.Errorf("stun: invalid time: %w", err)
				}
			}
			if f.Cooldown != "" {
				if s.cooldown, err = time.ParseDuration(f.Cooldown); err != nil {
					return fmt.Errorf("stun: invalid cooldown: %w", err)
				}
			}
			stuns = append(stuns, s)

		default:
			return fmt.Errorf("unknown feature %q", f.Feature)
		}
	}

	// All values valid — apply atomically
	cfg = c
	cfgTurns = turns
	cfgStuns = stuns
	cfgTwitchChan = channelFromURL(c.TwitchLink)
	return nil
}

func loadConfig() {
	if err := parseConfig(); err != nil {
		fmt.Fprintln(os.Stderr, "config error:", err)
		os.Exit(1)
	}
}

func reloadConfig() {
	if err := parseConfig(); err != nil {
		fmt.Fprintln(os.Stderr, "[config] reload failed:", err)
		return
	}
	fmt.Println("[config] reloaded")
}

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
