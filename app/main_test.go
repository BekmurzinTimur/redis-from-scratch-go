package main

import (
	// "io"
	// "net"
	// "strings"

	"net"
	"testing"
)

func TestGet(t *testing.T) {
	store.Set("name", "john")

	server, client := net.Pipe()

	go handleGet(server, []string{})

	buf := make([]byte, 1024)
	n, _ := client.Read(buf)
	reply := string(buf[:n])

	want := "$5\r\njohn\r\n"
	if reply != want {
		t.Errorf("got %q, want %q", reply, want)
	}
}
