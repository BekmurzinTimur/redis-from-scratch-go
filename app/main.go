package main

import (
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

func main() {

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
		if err != nil {
			return
		}
		fmt.Println(cmd, args, len(args))
		switch strings.ToUpper(cmd) {
		case "LRANGE":
			err = handleLRange(conn, args)
		case "RPUSH":
			err = handleRPush(conn, args)
		case "SET":
			err = handleSet(conn, args)
		case "GET":
			err = handleGet(conn, args)
		case "ECHO":
			err = handleEcho(conn, args)
		case "PING":
			err = handlePing(conn, args)
		default:
			err = handleUnknown(conn, args)
		}
		if err != nil {
			return
		}
	}
}

func handleLRange(conn net.Conn, args []string) error {
	if len(args) < 3 {
		if _, err := conn.Write([]byte("-ERR wrong number of arguments\r\n")); err != nil {
			return err
		}
		return nil
	}
	key, startString, stopString := args[0], args[1], args[2]
	start, err := strconv.Atoi(startString)
	if err != nil {
		return err
	}
	stop, err := strconv.Atoi(stopString)
	if err != nil {
		return err
	}
	list, ok := store.GetList(key, start, stop)
	if !ok {
		if _, err := conn.Write([]byte("*0\r\n")); err != nil {
			return err
		}
		return nil
	}
	var response string
	length := len(list)
	if length == 0 {
		response = "*0\r\n"
	} else {
		response = "*" + strconv.Itoa(length) + "\r\n"
		for _, w := range list {
			response += "$" + strconv.Itoa(len(w)) + "\r\n" + w + "\r\n"
		}
	}
	if _, err := conn.Write([]byte(response)); err != nil {
		return err
	}
	return nil
}

func handleRPush(conn net.Conn, args []string) error {
	if len(args) < 2 {
		if _, err := conn.Write([]byte("-ERR wrong number of arguments\r\n")); err != nil {
			return err
		}
		return nil
	}
	listKey, values := args[0], args[1:]
	length := store.SetList(listKey, values)
	response := fmt.Sprintf(":%d\r\n", length)
	if _, err := conn.Write([]byte(response)); err != nil {
		return err
	}
	return nil
}

func handleSet(conn net.Conn, args []string) error {
	if len(args) < 2 {
		if _, err := conn.Write([]byte("-ERR wrong number of arguments\r\n")); err != nil {
			return err
		}
		return nil
	}
	key, value := args[0], args[1]
	store.Set(key, value)
	if len(args) == 2 {
		store.DeleteExpiry(key)
	}
outer:
	for i := 2; i < len(args); i++ {
		switch strings.ToUpper(args[i]) {
		case "PX":
			expiryTime, err := strconv.Atoi(args[i+1])
			if err != nil {
				return err
			}
			dateOfExpiry := time.Now().Add(time.Duration(expiryTime) * time.Millisecond)

			fmt.Println("dateOfExpiry", dateOfExpiry)
			store.SetExpiry(key, dateOfExpiry)
			break outer
		case "EX":
			expiryTime, err := strconv.Atoi(args[i+1])
			if err != nil {
				return err
			}
			dateOfExpiry := time.Now().Add(time.Duration(expiryTime) * time.Second)
			store.SetExpiry(key, dateOfExpiry)
			break outer
		default:
			store.DeleteExpiry(key)
		}

	}
	if _, err := conn.Write([]byte("+OK\r\n")); err != nil {
		return err
	}
	return nil
}

func handleGet(conn net.Conn, args []string) error {
	if len(args) < 1 {
		if _, err := conn.Write([]byte("-ERR wrong number of arguments\r\n")); err != nil {
			return err
		}
		return nil
	}
	key := args[0]
	value, ok := store.Get(key)
	if !ok {
		if _, err := conn.Write([]byte("$-1\r\n")); err != nil {
			return err
		}
		return nil
	}
	dateOfExpiry, ok := store.GetExpiry(key)
	fmt.Println("Now", time.Now())
	fmt.Println("Expiry", dateOfExpiry)
	if ok && dateOfExpiry.Before(time.Now()) {
		if _, err := conn.Write([]byte("$-1\r\n")); err != nil {
			return err
		}
		return nil
	}
	response := fmt.Sprintf("$%d\r\n%s\r\n", len(value), value)
	if _, err := conn.Write([]byte(response)); err != nil {
		return err
	}
	return nil
}

func handleEcho(conn net.Conn, args []string) error {
	var text string
	for _, w := range args {
		text = text + w
	}
	response := "$" + strconv.Itoa(len(text)) + "\r\n" + text + "\r\n"
	fmt.Println(response)
	if _, err := conn.Write([]byte(response)); err != nil {
		return err
	}
	return nil
}

func handlePing(conn net.Conn, args []string) error {
	if _, err := conn.Write([]byte("+PONG\r\n")); err != nil {
		return err
	}
	return nil
}

func handleUnknown(conn net.Conn, args []string) error {
	if _, err := conn.Write([]byte("-ERR Unkown command\r\n")); err != nil {
		return err
	}
	return nil
}

func parseRESPCommand(cmd string) (string, []string, error) {
	cmdParts := strings.SplitN(cmd, "\r\n", 2)
	if len(cmdParts) < 2 {
		return "", nil, errors.New("invalid RESP command")
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
		return "", nil, errors.New("invalid RESP command")
	} else if len(parsedCommand) == 1 {
		return parsedCommand[0], nil, nil
	}
	return parsedCommand[0], parsedCommand[1:], nil
}
