package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
)

type Config struct {
	Debug180Key  string `json:"debug180Key"`
	DebugStunKey string `json:"debugStunKey"`
	DebugTime    string `json:"debugTime"`
	TurnDistance int32  `json:"turnDistance"`
	TwitchLink   string `json:"twitchLink"`
}

var (
	cfg           Config
	cfgTurnKey    uint32
	cfgStunKey    uint32
	cfgStunTime   time.Duration
	cfgTwitchChan string
	cfgTurnDist   int32
)

// keyNames maps human-readable key names to Windows virtual key codes.
var keyNames = map[string]uint32{
	"F1": 0x70, "F2": 0x71, "F3": 0x72, "F4": 0x73,
	"F5": 0x74, "F6": 0x75, "F7": 0x76, "F8": 0x77,
	"F9": 0x78, "F10": 0x79, "F11": 0x7A, "F12": 0x7B,
}

func loadConfig() {
	data, err := os.ReadFile("config.json")
	if err != nil {
		fmt.Fprintln(os.Stderr, "cannot read config.json:", err)
		os.Exit(1)
	}
	if err = json.Unmarshal(data, &cfg); err != nil {
		fmt.Fprintln(os.Stderr, "invalid config.json:", err)
		os.Exit(1)
	}

	cfgTurnKey = parseKeyName(cfg.Debug180Key)
	cfgStunKey = parseKeyName(cfg.DebugStunKey)

	if cfg.DebugTime == "" {
		cfgStunTime = 60 * time.Second
	} else {
		cfgStunTime, err = time.ParseDuration(cfg.DebugTime)
		if err != nil {
			fmt.Fprintln(os.Stderr, `invalid debugTime — use e.g. "60s" or "1m30s":`, err)
			os.Exit(1)
		}
	}

	cfgTurnDist = cfg.TurnDistance
	cfgTwitchChan = channelFromURL(cfg.TwitchLink)
}

func parseKeyName(name string) uint32 {
	if name == "" {
		return 0
	}
	vk, ok := keyNames[strings.ToUpper(name)]
	if !ok {
		fmt.Fprintf(os.Stderr, "unknown key %q — supported: F1–F12\n", name)
		os.Exit(1)
	}
	return vk
}

// channelFromURL extracts "podushkamonster" from "https://www.twitch.tv/podushkamonster".
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
