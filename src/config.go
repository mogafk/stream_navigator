package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync/atomic"
	"time"
)

type RootConfig struct {
	TwitchLink   string `json:"twitchLink"`
	ActiveConfig string `json:"activeConfig"`
}

type ObsLayoutConfig struct {
	Duration string `json:"duration"`
}

type FeatureConfig struct {
	Feature        string          `json:"feature"`
	DebugKey       string          `json:"debugKey"`
	Cooldown       string          `json:"cooldown"`
	ChatCommand    []string        `json:"chatCommand"`
	Time           string          `json:"time"`
	Distance       int32           `json:"distance"`
	ModificatorKey string          `json:"modificatorKey"`
	ObsLayout      ObsLayoutConfig `json:"obsLayout"`
}

type Config struct {
	Mute       bool            `json:"mute"`
	ServerPort int             `json:"serverPort"`
	Features   []FeatureConfig `json:"features"`
}

type turnEntry struct {
	key           uint32
	debugKey      string
	dist          int32
	cooldown      time.Duration
	chatCmds      []string
	modDown       uint32
	modUp         uint32
	modVK         uint32
	modOtherVK    uint32
	modOtherUp    uint32
	obsLayoutDur  time.Duration
	lastNano      atomic.Int64
}

type stunEntry struct {
	key          uint32
	debugKey     string
	duration     time.Duration
	cooldown     time.Duration
	chatCmds     []string
	obsLayoutDur time.Duration
	lastNano     atomic.Int64
}

var (
	rootCfg       RootConfig
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
	rootData, err := os.ReadFile("config.json")
	if err != nil {
		return fmt.Errorf("config.json: %w", err)
	}
	var root RootConfig
	if err = json.Unmarshal(rootData, &root); err != nil {
		return fmt.Errorf("config.json: %w", err)
	}
	if root.ActiveConfig == "" {
		return fmt.Errorf("config.json: activeConfig is empty")
	}

	activeData, err := os.ReadFile(root.ActiveConfig)
	if err != nil {
		return fmt.Errorf("%s: %w", root.ActiveConfig, err)
	}
	var c Config
	if err = json.Unmarshal(activeData, &c); err != nil {
		return fmt.Errorf("%s: %w", root.ActiveConfig, err)
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
			if f.ObsLayout.Duration != "" {
				if t.obsLayoutDur, err = time.ParseDuration(f.ObsLayout.Duration); err != nil {
					return fmt.Errorf("turn: invalid obsLayout.duration: %w", err)
				}
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
			if f.ObsLayout.Duration != "" {
				if s.obsLayoutDur, err = time.ParseDuration(f.ObsLayout.Duration); err != nil {
					return fmt.Errorf("stun: invalid obsLayout.duration: %w", err)
				}
			}
			stuns = append(stuns, s)

		default:
			return fmt.Errorf("unknown feature %q", f.Feature)
		}
	}

	// All values valid — apply atomically
	rootCfg = root
	cfg = c
	cfgTurns = turns
	cfgStuns = stuns
	cfgTwitchChan = channelFromURL(root.TwitchLink)
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

// watchConfig polls config.json and the active config file every second,
// reloading when either changes.
func watchConfig() {
	modTime := func(path string) time.Time {
		info, err := os.Stat(path)
		if err != nil {
			return time.Time{}
		}
		return info.ModTime()
	}

	lastRoot   := modTime("config.json")
	lastActive := modTime(rootCfg.ActiveConfig)

	for range time.Tick(time.Second) {
		rootChanged   := modTime("config.json").After(lastRoot)
		activeChanged := modTime(rootCfg.ActiveConfig).After(lastActive)

		if rootChanged || activeChanged {
			lastRoot   = modTime("config.json")
			lastActive = modTime(rootCfg.ActiveConfig)
			reloadConfig()
			// update active path in case activeConfig field changed
			lastActive = modTime(rootCfg.ActiveConfig)
		}
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
