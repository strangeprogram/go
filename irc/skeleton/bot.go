package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
	"time"
)

// IRCBot represents the bot structure
type IRCBot struct {
	primaryServer string
	backupServer  string
	port          string
	nick          string
	user          string
	channel       string
	conn          net.Conn
}

// NewIRCBot initializes a new IRC bot
func NewIRCBot(primaryServer, backupServer, port, nick, user, channel string) *IRCBot {
	return &IRCBot{
		primaryServer: primaryServer,
		backupServer:  backupServer,
		port:          port,
		nick:          nick,
		user:          user,
		channel:       channel,
	}
}

// Connect tries to connect to the IRC server, with a backup option
func (bot *IRCBot) Connect() error {
	var err error
	for attempt := 1; attempt <= 3; attempt++ {
		fmt.Printf("Attempt %d to connect to %s...\n", attempt, bot.primaryServer)
		bot.conn, err = net.Dial("tcp", bot.primaryServer+":"+bot.port)
		if err != nil {
			fmt.Println("Failed to connect, retrying...")
			time.Sleep(2 * time.Second) // Wait before retrying
			continue
		}

		// Set a read timeout
		bot.conn.SetReadDeadline(time.Now().Add(30 * time.Second))

		// Send NICK and USER immediately after connection
		if err := bot.send(fmt.Sprintf("NICK %s", bot.nick)); err != nil {
			return err
		}
		if err := bot.send(fmt.Sprintf("USER %s 8 * :%s", bot.user, bot.nick)); err != nil {
			return err
		}

		return nil // Successful connection
	}

	// If primary server fails after 3 attempts, try backup server
	fmt.Printf("Connecting to backup server %s...\n", bot.backupServer)
	bot.conn, err = net.Dial("tcp", bot.backupServer+":"+bot.port)
	if err != nil {
		return err
	}

	// Send NICK and USER for backup server
	if err := bot.send(fmt.Sprintf("NICK %s", bot.nick)); err != nil {
		return err
	}
	if err := bot.send(fmt.Sprintf("USER %s 8 * :%s", bot.user, bot.nick)); err != nil {
		return err
	}

	return nil
}

// send sends a message to the IRC server and checks for errors
func (bot *IRCBot) send(message string) error {
	_, err := bot.conn.Write([]byte(message + "\r\n"))
	if err != nil {
		fmt.Printf("Error sending command to server: %s\n", err)
	}
	return err
}

// Run starts the bot's main loop
func (bot *IRCBot) Run() {
	scanner := bufio.NewScanner(bot.conn)
	for {
		// Set a read timeout
		bot.conn.SetReadDeadline(time.Now().Add(5 * time.Minute)) // Adjust as needed

		if !scanner.Scan() {
			if err := scanner.Err(); err != nil {
				fmt.Println("Error reading from connection:", err)
				// Reconnect or exit based on the error
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					fmt.Println("Timeout occurred. Attempting to reconnect...")
					if err := bot.Connect(); err != nil {
						fmt.Println("Reconnection failed:", err)
						return
					}
					continue
				} else {
					return
				}
			}
			break
		}

		line := scanner.Text()
		fmt.Println("Received:", line)

		if strings.HasPrefix(line, "PING") {
			// Respond to PING messages to keep the connection alive
			cookie := strings.TrimPrefix(line, "PING ")
			bot.send("PONG " + cookie)
		} else if strings.Contains(line, "001") {
			bot.send("JOIN " + bot.channel)
		} else {
			bot.handleMessage(line)
		}
	}
}

// handleMessage processes chat messages and responds to commands
func (bot *IRCBot) handleMessage(message string) {
	parts := strings.Split(message, " ")
	if len(parts) < 4 {
		return
	}

	if parts[1] == "PRIVMSG" {
		sender := strings.Split(parts[0], "!")[0][1:]
		channel := parts[2]
		command := strings.Join(parts[3:], " ")[1:]

		switch command {
		case "!ping":
			bot.send(fmt.Sprintf("PRIVMSG %s :%s: Pong!", channel, sender))
		case "!quit":
			bot.send("QUIT: Quit command issued")
			os.Exit(0)
			// Add more commands here
		}
	}
}

// main is the entry point of the program
func main() {
	// Hardcoded configurations
	primaryServer := "irc.supernets.org"
	backupServer := ""
	port := "6667"
	nick := "GOnzo"
	user := "GOnzo"
	channel := "#kushboy"

	bot := NewIRCBot(primaryServer, backupServer, port, nick, user, channel)
	err := bot.Connect()
	if err != nil {
		fmt.Println("Failed to connect to both servers:", err)
		return
	}
	bot.Run()
}
