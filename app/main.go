package main

import (
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
)

// Ensures gofmt doesn't remove the "net" and "os" imports in stage 1 (feel free to remove this!)
var _ = net.Listen
var _ = os.Exit

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
		if strings.ToUpper(cmd) == "ECHO" {
			response := "$" + strconv.Itoa(len(args)) + "\r\n" + args + "\r\n"
			fmt.Println(response)
			if _, err := conn.Write([]byte(response)); err != nil {
				return
			}
		} else if strings.ToUpper(cmd) == "PING" {
			if _, err := conn.Write([]byte("+PONG\r\n")); err != nil {
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
