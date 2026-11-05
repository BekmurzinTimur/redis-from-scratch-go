package main

import (
	// "io"
	// "net"
	// "strings"

	"net"
	"testing"
)

func TestGet(t *testing.T) {
	store.Set("name", "timur")

	server, client := net.Pipe()

	go handleGet(server, []string{"name"})

	buf := make([]byte, 1024)
	n, _ := client.Read(buf)
	reply := string(buf[:n])

	want := "$5\r\ntimur\r\n"
	if reply != want {
		t.Errorf("got %q, want %q", reply, want)
	}
}
