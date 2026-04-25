package main

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

const (
	chatCommand = "!180"
	chatStunCmd = "!стан"
)

func main() {
	loadConfig()

	fmt.Println("Stream Navigator")
	fmt.Printf("  turn  : key=%s | chat=%s | distance=%d px\n", keyLabel(cfg.Debug180Key), chatCommand, cfgTurnDist)
	fmt.Printf("  стан  : key=%s | chat=%s | duration=%s\n", keyLabel(cfg.DebugStunKey), chatStunCmd, cfgStunTime)
	delayLabel := "instant"
	if cfgTriggerInterval > 0 {
		delayLabel = cfgTriggerInterval.String()
	}
	fmt.Printf("  twitch: #%s | delay=%s\n", cfgTwitchChan, delayLabel)
	fmt.Println("Ctrl+C to quit")

	go watchConfig()

	if cfgTwitchChan != "" {
		go connectTwitch(cfgTwitchChan, func(username, msg string) {
			switch strings.ToLower(strings.TrimSpace(msg)) {
			case chatCommand:
				fmt.Printf("[chat] %s → 180° (через %s)\n", username, cfgTriggerInterval)
				time.AfterFunc(cfgTriggerInterval, moveMouse180)
			case chatStunCmd:
				fmt.Printf("[chat] %s → стан (через %s)\n", username, cfgTriggerInterval)
				time.AfterFunc(cfgTriggerInterval, activateStun)
			}
		})
	}

	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt, syscall.SIGTERM)
		<-c
		stopKeyboardHook()
		os.Exit(0)
	}()

	runKeyboardHook()
}
