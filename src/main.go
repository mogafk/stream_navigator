package main

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
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

func cooldownLeft(nano int64, cooldown time.Duration) time.Duration {
	left := cooldown - time.Duration(time.Now().UnixNano()-nano)
	if left < 0 {
		return 0
	}
	return left.Round(time.Second)
}

func main() {
	loadConfig()

	fmt.Println("Stream Navigator")
	for _, t := range cfgTurns {
		fmt.Printf("  turn  : key=%s | chat=%s | distance=%d px\n", keyLabel(t.debugKey), strings.Join(t.chatCmds, " | "), t.dist)
	}
	for _, s := range cfgStuns {
		fmt.Printf("  стан  : key=%s | chat=%s | duration=%s\n", keyLabel(s.debugKey), strings.Join(s.chatCmds, " | "), s.duration)
	}
	fmt.Printf("  twitch: #%s\n", cfgTwitchChan)
	fmt.Println("Ctrl+C to quit")

	go watchConfig()

	if cfgTwitchChan != "" {
		go connectTwitch(cfgTwitchChan, func(username, msg string) {
			normalized := strings.ToLower(strings.TrimSpace(msg))
			for _, t := range cfgTurns {
				if matchesCmd(normalized, t.chatCmds) {
					prev := t.lastNano.Load()
					if t.cooldown > 0 && time.Duration(time.Now().UnixNano()-prev) < t.cooldown {
						fmt.Printf("[main] skip поворот by %s (cooldown %s left)\n", username, cooldownLeft(prev, t.cooldown))
						return
					}
					if !t.lastNano.CompareAndSwap(prev, time.Now().UnixNano()) {
						return
					}
					fmt.Printf("[main] do поворот %d px by %s\n", t.dist, username)
					moveMouse180(t)
					return
				}
			}
			for _, s := range cfgStuns {
				if matchesCmd(normalized, s.chatCmds) {
					prev := s.lastNano.Load()
					if s.cooldown > 0 && time.Duration(time.Now().UnixNano()-prev) < s.cooldown {
						fmt.Printf("[main] skip стан by %s (cooldown %s left)\n", username, cooldownLeft(prev, s.cooldown))
						return
					}
					if !s.lastNano.CompareAndSwap(prev, time.Now().UnixNano()) {
						return
					}
					fmt.Printf("[main] do стан by %s\n", username)
					activateStun(s)
					return
				}
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
