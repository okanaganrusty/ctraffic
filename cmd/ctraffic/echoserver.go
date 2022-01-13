package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
)

const bufSize = 8192

// EchoServer is a helper to handle one to many chat
type EchoServer struct {
	conns map[string]net.Conn
	lock  sync.RWMutex
}

// NewEchoServer builds a new EchoServer
func NewEchoServer() *EchoServer {
	return &EchoServer{conns: make(map[string]net.Conn)}
}

// Register adds a new conn to the EchoServer
func (h *EchoServer) Register(conn net.Conn) {
	fmt.Printf("Connected to %s\n", conn.RemoteAddr())
	h.lock.Lock()
	defer h.lock.Unlock()

	h.conns[conn.RemoteAddr().String()] = conn

	go h.readLoop(conn)
}

func (h *EchoServer) readLoop(conn net.Conn) {
	b := make([]byte, bufSize)
	for {
		n, err := conn.Read(b)
		if err != nil {
			h.unregister(conn)
			return
		}

		fmt.Printf("Got message: %s\n", string(b[:n]))
		conn.Write(b[:n])
	}
}

func (h *EchoServer) unregister(conn net.Conn) {
	h.lock.Lock()
	defer h.lock.Unlock()
	delete(h.conns, conn.RemoteAddr().String())
	err := conn.Close()
	if err != nil {
		fmt.Println("Failed to disconnect", conn.RemoteAddr(), err)
	} else {
		fmt.Println("Disconnected ", conn.RemoteAddr())
	}
}

func (h *EchoServer) broadcast(msg []byte) {
	h.lock.RLock()
	defer h.lock.RUnlock()
	for _, conn := range h.conns {
		_, err := conn.Write(msg)
		if err != nil {
			fmt.Printf("Failed to write message to %s: %v\n", conn.RemoteAddr(), err)
		}
	}
}

// Check is a helper to throw errors in the examples
func Check(err error) {
	switch e := err.(type) {
	case nil:
	case (net.Error):
		if e.Temporary() {
			fmt.Printf("Warning: %v\n", err)
			return
		}

		fmt.Printf("net.Error: %v\n", err)
		panic(err)
	default:
		fmt.Printf("error: %v\n", err)
		panic(err)
	}
}

// Chat starts the stdin readloop to dispatch messages to the EchoServer
func (h *EchoServer) Chat() {
	reader := bufio.NewReader(os.Stdin)
	for {
		msg, err := reader.ReadString('\n')
		Check(err)
		if strings.TrimSpace(msg) == "exit" {
			return
		}
		h.broadcast([]byte(msg))
	}
}
