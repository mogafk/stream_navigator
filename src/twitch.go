package main

import (
	"bufio"
	"fmt"
	"math/rand"
	"net"
	"strings"
	"time"
)

const twitchIRC = "irc.chat.twitch.tv:6667"

// connectTwitch joins a Twitch IRC channel anonymously and calls onMessage for every PRIVMSG.
// Reconnects automatically on disconnect.
func connectTwitch(channel string, onMessage func(username, msg string)) {
	for {
		err := ircSession(channel, onMessage)
		if err != nil {
			fmt.Printf("[twitch] error: %v — reconnecting in 5s\n", err)
		} else {
			fmt.Println("[twitch] disconnected — reconnecting in 5s")
		}
		time.Sleep(5 * time.Second)
	}
}

func ircSession(channel string, onMessage func(username, msg string)) error {
	conn, err := net.DialTimeout("tcp", twitchIRC, 10*time.Second)
	if err != nil {
		return err
	}
	defer conn.Close()

	nick := fmt.Sprintf("justinfan%d", rand.Intn(90000)+10000)
	fmt.Fprintf(conn, "NICK %s\r\n", nick)
	fmt.Fprintf(conn, "JOIN #%s\r\n", strings.ToLower(channel))
	fmt.Printf("[twitch] connected as %s, joined #%s\n", nick, channel)

	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		line := scanner.Text()

		// Keep-alive
		if strings.HasPrefix(line, "PING") {
			fmt.Fprintf(conn, "PONG %s\r\n", strings.TrimPrefix(line, "PING "))
			continue
		}

		// Only care about chat messages
		// Format: :nick!nick@nick.tmi.twitch.tv PRIVMSG #channel :message
		if !strings.Contains(line, " PRIVMSG ") {
			continue
		}

		username := parseNick(line)
		msg := parseMessage(line)
		if username == "" || msg == "" {
			continue
		}

		fmt.Printf("[chat] get message %s: %s\n", username, msg)
		onMessage(username, msg)
	}

	return scanner.Err()
}

// parseNick extracts the username from the IRC prefix ":nick!nick@..."
func parseNick(line string) string {
	if len(line) == 0 || line[0] != ':' {
		return ""
	}
	end := strings.IndexByte(line, '!')
	if end < 2 {
		return ""
	}
	return line[1:end]
}

// parseMessage extracts the chat text after the last " :" in the line.
func parseMessage(line string) string {
	idx := strings.LastIndex(line, " :")
	if idx < 0 {
		return ""
	}
	return line[idx+2:]
}
