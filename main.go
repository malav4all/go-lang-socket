package main

import (
	"bufio"
	"fmt"
	"net"
	"net/http"
	"os"

	"github.com/gorilla/websocket"
)

// WebSocket Upgrader
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// WebSocket Handler
func websocketHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println("Error upgrading to websocket:", err)
		return
	}
	defer conn.Close()

	for {
		messageType, message, err := conn.ReadMessage()
		if err != nil {
			fmt.Println("Error reading message:", err)
			break
		}
		fmt.Printf("WebSocket Received: %s\n", message)

		// Send message to TCP server and get acknowledgment
		ack, err := sendToTCPServer(string(message))
		if err != nil {
			fmt.Println("Error sending to TCP server:", err)
			break
		}

		// Send acknowledgment back to WebSocket client
		if err := conn.WriteMessage(messageType, []byte(ack)); err != nil {
			fmt.Println("Error sending ACK to WebSocket client:", err)
			break
		}
	}
}

// Function to send data to TCP server and receive acknowledgment
func sendToTCPServer(data string) (string, error) {
	tcpAddr := "localhost:9090" // Address of the TCP server
	conn, err := net.Dial("tcp", tcpAddr)
	if err != nil {
		return "", fmt.Errorf("Error connecting to TCP server: %w", err)
	}
	defer conn.Close()

	// Send data to the TCP server
	_, err = conn.Write([]byte(data + "\n"))
	if err != nil {
		return "", fmt.Errorf("Error sending data to TCP server: %w", err)
	}

	// Read acknowledgment from the TCP server
	reader := bufio.NewReader(conn)
	ack, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("Error reading ACK from TCP server: %w", err)
	}

	return ack, nil
}

// Start TCP server
func startTCPServer() {
	port := ":9090"

	listener, err := net.Listen("tcp", port)
	if err != nil {
		fmt.Println("Error starting TCP server:", err)
		os.Exit(1)
	}
	defer listener.Close()
	fmt.Println("TCP Server listening on", port)

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Error accepting TCP connection:", err)
			continue
		}

		go handleTCPConnection(conn)
	}
}

// TCP/IP Handler
func handleTCPConnection(conn net.Conn) {
	defer conn.Close()

	reader := bufio.NewReader(conn)

	for {
		data, err := reader.ReadString('\n')
		if err != nil {
			if err.Error() == "EOF" {
				fmt.Println("TCP Client disconnected")
			} else {
				fmt.Println("Error reading from TCP connection:", err)
			}
			return
		}

		fmt.Print("TCP Received: ", data)

		ack := "ACK received\n"
		_, err = conn.Write([]byte(ack))
		if err != nil {
			fmt.Println("Error sending TCP ACK:", err)
			return
		}
	}
}

func main() {
	// Start TCP server in a new Goroutine
	go startTCPServer()

	// Start WebSocket server
	http.HandleFunc("/ws", websocketHandler)
	fmt.Println("WebSocket server started at :8080")
	http.ListenAndServe(":8080", nil)
}
