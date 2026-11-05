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
var listStore = map[string][]string{}
var expiryStore = map[string]time.Time{}

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
		case "RPUSH":
			if len(args) < 2 {
				if _, err := conn.Write([]byte("-ERR wrong number of arguments\r\n")); err != nil {
					return
				}
				continue
			}
			lisKey, values := args[0], args[1:]
			length := LockedWriteList(lisKey, values)
			response := fmt.Sprintf(":%d\r\n", length)
			if _, err := conn.Write([]byte(response)); err != nil {
				return
			}
		case "SET":
			if len(args) < 2 {
				if _, err := conn.Write([]byte("-ERR wrong number of arguments\r\n")); err != nil {
					return
				}
				continue
			}
			key, value := args[0], args[1]
			LockedWrite(key, value)
			if len(args) == 2 {
				LockedDeleteExpiry(key)
			}
		outer:
			for i := 2; i < len(args); i++ {
				switch strings.ToUpper(args[i]) {
				case "PX":
					expiryTime, err := strconv.Atoi(args[i+1])
					if err != nil {
						return
					}
					dateOfExpiry := time.Now().Add(time.Duration(expiryTime) * time.Millisecond)

					fmt.Println("dateOfExpiry", dateOfExpiry)
					LockedWriteExpiry(key, dateOfExpiry)
					break outer
				case "EX":
					expiryTime, err := strconv.Atoi(args[i+1])
					if err != nil {
						return
					}
					dateOfExpiry := time.Now().Add(time.Duration(expiryTime) * time.Second)
					LockedWriteExpiry(key, dateOfExpiry)
					break outer
				default:
					LockedDeleteExpiry(key)
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
			value, ok := LockedRead(key)
			if !ok {
				if _, err := conn.Write([]byte("$-1\r\n")); err != nil {
					return
				}
				continue
			}
			dateOfExpiry, ok := LockedReadExpiry(key)
			fmt.Println("Now", time.Now())
			fmt.Println("Expiry", dateOfExpiry)
			if ok && dateOfExpiry.Before(time.Now()) {
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

func LockedRead(key string) (string, bool) {
	mu.RLock()
	value, ok := store[key]
	mu.RUnlock()
	return value, ok
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

func LockedReadExpiry(key string) (time.Time, bool) {
	mu.RLock()
	value, ok := expiryStore[key]
	mu.RUnlock()
	return value, ok
}

func LockedWriteExpiry(key string, _time time.Time) {
	mu.Lock()
	expiryStore[key] = _time
	mu.Unlock()
}

func LockedDeleteExpiry(key string) {
	mu.Lock()
	delete(expiryStore, key)
	mu.Unlock()
}

func LockedWriteList(key string, items []string) int {
	mu.Lock()
	listStore[key] = append(listStore[key], items...)
	mu.Unlock()
	length := len(listStore[key])
	return length
}
