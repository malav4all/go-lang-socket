package main

import (
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func handler(w http.ResponseWriter, r *http.Request) {
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

		// Simulate sending logs to the client
		for i := 0; i < 5; i++ {
			logMessage := fmt.Sprintf("%s: Attempted to connect to input device.", time.Now().Format("15:04:05.000"))
			if err := conn.WriteMessage(messageType, []byte(logMessage)); err != nil {
				fmt.Println("Error sending log:", err)
				break
			}
			time.Sleep(1 * time.Second)
		}

		ack := []byte("ACK received")
		if err := conn.WriteMessage(messageType, ack); err != nil {
			fmt.Println("Error sending ACK:", err)
			break
		}
	}
}

func tcpServer() {
	listener, err := net.Listen("tcp", ":9990")
	if err != nil {
		fmt.Println("Error starting TCP server:", err)
		return
	}
	defer listener.Close()
	fmt.Println("TCP Server listening on :9990")

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Error accepting connection:", err)
			continue
		}
		go handleTCPConnection(conn)
	}
}

func handleTCPConnection(conn net.Conn) {
	defer conn.Close()

	buf := make([]byte, 1024)
	for {
		n, err := conn.Read(buf)
		if err != nil {
			fmt.Println("TCP Client disconnected")
			break
		}
		message := string(buf[:n])
		fmt.Printf("TCP Received: %s\n", message)

		ack := "ACK received\n"
		conn.Write([]byte(ack))
	}
}

func main() {
	go tcpServer()

	http.HandleFunc("/ws", handler)
	fmt.Println("WebSocket server started at :8080")
	http.ListenAndServe(":8080", nil)
}
