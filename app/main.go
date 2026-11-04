package main

import (
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

var mu sync.RWMutex
var store = map[string]string{}

func main() {
	// You can use print statements as follows for debugging, they'll be visible when running tests.
	fmt.Println("Logs from your program will appear here!")

	// Uncomment the code below to pass the first stage
	//
	listener, err := net.Listen("tcp", "0.0.0.0:6379")
	if err != nil {
		fmt.Println("Failed to bind to port 6379")
		os.Exit(1)
	}
	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Error accepting connection: ", err)
			continue
		}
		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {
	defer conn.Close()
	buf := make([]byte, 1024)
	for {
		n, err := conn.Read(buf)
		if errors.Is(err, io.EOF) {
			return
		}
		if err != nil {
			fmt.Println("Error while reading the response", err)
			break
		}
		cmd, args, err := parseRESPCommand(string(buf[:n]))
		fmt.Println(cmd, args, len(args))
		if err != nil {
			return
		}
		switch strings.ToUpper(cmd) {
		case "SET":
			if len(args) < 2 {
				if _, err := conn.Write([]byte("-ERR wrong number of arguments\r\n")); err != nil {
					return
				}
			}
			key, value := args[0], args[1]
			LockedWrite(key, value)
			for i := 2; i < len(args); i++ {
				switch strings.ToUpper(args[i]) {
				case "PX":
					expiryTime, err := strconv.Atoi(args[i+1])
					if err != nil {
						return
					}
					time.AfterFunc(time.Duration(expiryTime)*time.Millisecond, func() {
						LockedDelete(key)
					})
				case "EX":
					expiryTime, err := strconv.Atoi(args[i+1])
					if err != nil {
						return
					}
					time.AfterFunc(time.Duration(expiryTime)*time.Second, func() {
						LockedDelete(key)
					})
				}
			}
			fmt.Println("Store:", store)
			if _, err := conn.Write([]byte("+OK\r\n")); err != nil {
				return
			}
		case "GET":
			if len(args) < 1 {
				if _, err := conn.Write([]byte("-ERR wrong number of arguments\r\n")); err != nil {
					return
				}
			}
			key := args[0]
			mu.RLock()
			value, ok := store[key]
			mu.RUnlock()
			if !ok {
				if _, err := conn.Write([]byte("$-1\r\n")); err != nil {
					return
				}
				continue
			}
			response := fmt.Sprintf("$%d\r\n%s\r\n", len(value), value)
			if _, err := conn.Write([]byte(response)); err != nil {
				return
			}
		case "ECHO":
			var text string
			for _, w := range args {
				text = text + w
			}
			response := "$" + strconv.Itoa(len(text)) + "\r\n" + text + "\r\n"
			fmt.Println(response)
			if _, err := conn.Write([]byte(response)); err != nil {
				return
			}
		case "PING":
			if _, err := conn.Write([]byte("+PONG\r\n")); err != nil {
				return
			}
		default:
			if _, err := conn.Write([]byte("-ERR Unkown command\r\n")); err != nil {
				return
			}
		}
	}
}

func parseRESPCommand(cmd string) (string, []string, error) {
	cmdParts := strings.SplitN(cmd, "\r\n", 2)
	if len(cmdParts) < 2 {
		fmt.Println("Invalid RESP command")
		return "", nil, errors.New("Invalid RESP command")
	}
	_, rest := cmdParts[0], cmdParts[1]

	words := strings.Split(rest, "\r\n")
	var parsedCommand []string
	for i, w := range words {
		if i%2 == 1 {
			parsedCommand = append(parsedCommand, w)
		}
	}
	if len(parsedCommand) == 0 {
		fmt.Println("Invalid RESP command")
		return "", nil, errors.New("Invalid RESP command")
	} else if len(parsedCommand) == 1 {
		return parsedCommand[0], nil, nil
	}
	return parsedCommand[0], parsedCommand[1:], nil
}

func LockedWrite(key string, value string) {
	mu.Lock()
	store[key] = value
	mu.Unlock()
}

func LockedDelete(key string) {
	mu.Lock()
	delete(store, key)
	mu.Unlock()
}
