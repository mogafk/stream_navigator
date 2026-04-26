package main

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
	"sync/atomic"
	"syscall"
	"time"
)

func matchesCmd(msg string, cmds []string) bool {
	for _, cmd := range cmds {
		if msg == strings.ToLower(cmd) {
			return true
		}
	}
	return false
}

var (
	last180Nano  atomic.Int64
	lastStunNano atomic.Int64
)

// tryTrigger returns true if the cooldown has passed and records the current time.
// Uses CAS so only one concurrent call wins when multiple arrive at the same time.
func tryTrigger(last *atomic.Int64, cooldown time.Duration) bool {
	if cooldown == 0 {
		return true
	}
	now := time.Now().UnixNano()
	prev := last.Load()
	if time.Duration(now-prev) < cooldown {
		return false
	}
	return last.CompareAndSwap(prev, now)
}

func cooldownLeft(last *atomic.Int64, cooldown time.Duration) time.Duration {
	left := cooldown - time.Duration(time.Now().UnixNano()-last.Load())
	if left < 0 {
		return 0
	}
	return left.Round(time.Second)
}

func main() {
	loadConfig()

	fmt.Println("Stream Navigator")
	fmt.Printf("  turn  : key=%s | chat=%s | distance=%d px\n", keyLabel(cfgDebug180Key), strings.Join(cfgChatCmds180, " | "), cfgTurnDist)
	fmt.Printf("  стан  : key=%s | chat=%s | duration=%s\n", keyLabel(cfgDebugStunKey), strings.Join(cfgChatCmdsStun, " | "), cfgStunTime)
	fmt.Printf("  twitch: #%s\n", cfgTwitchChan)
	fmt.Println("Ctrl+C to quit")

	go watchConfig()

	if cfgTwitchChan != "" {
		fmt.Printf("  cmds 180°: %v\n", cfgChatCmds180)
		fmt.Printf("  cmds стан: %v\n", cfgChatCmdsStun)
		go connectTwitch(cfgTwitchChan, func(username, msg string) {
			normalized := strings.ToLower(strings.TrimSpace(msg))
			switch {
			case matchesCmd(normalized, cfgChatCmds180):
				if !tryTrigger(&last180Nano, cfgCooldown180) {
					fmt.Printf("[main] skip 180° by %s (cooldown %s left)\n", username, cooldownLeft(&last180Nano, cfgCooldown180))
					return
				}
				fmt.Printf("[main] do 180° by %s\n", username)
				moveMouse180()
			case matchesCmd(normalized, cfgChatCmdsStun):
				if !tryTrigger(&lastStunNano, cfgCooldownStun) {
					fmt.Printf("[main] skip стан by %s (cooldown %s left)\n", username, cooldownLeft(&lastStunNano, cfgCooldownStun))
					return
				}
				fmt.Printf("[main] do стан by %s\n", username)
				activateStun()
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
