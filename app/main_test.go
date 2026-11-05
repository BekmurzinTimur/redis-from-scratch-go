package main

import (
	// "io"
	// "net"
	// "strings"
	"fmt"
	"net"
	"testing"
)

func TestGet(t *testing.T) {
	server, _ := net.Pipe()
	err := handleGet(server, []string{})
	if err != nil {
		fmt.Println(err)
	}
}
