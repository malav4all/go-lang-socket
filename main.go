package main

import (
	"fmt"
	"net"
	"net/http"
	"sync"
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

var (
	wsConnections = struct {
		sync.RWMutex
		connections []*websocket.Conn
	}{connections: make([]*websocket.Conn, 0)}
	listener   net.Listener
	tcpRunning = false
	mu         sync.Mutex
	done       chan struct{}
)

func handler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println("Error upgrading to websocket:", err)
		return
	}
	defer conn.Close()

	wsConnections.Lock()
	wsConnections.connections = append(wsConnections.connections, conn)
	wsConnections.Unlock()

	fmt.Printf("WebSocket connected: %s\n", conn.RemoteAddr().String())

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			fmt.Println("Error reading message:", err)
			break
		}
		fmt.Printf("WebSocket Received: %s\n", message)
	}
}

func startWebSocketServer() {
	addr := ":6000"
	http.HandleFunc("/ws", handler)
	fmt.Printf("Starting WebSocket server on %s\n", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		fmt.Printf("Error starting WebSocket server: %s\n", err)
	}
}

func startTCPServer(ip, port string, wg *sync.WaitGroup, messages chan<- string) {
	defer wg.Done()
	addr := fmt.Sprintf("%s:%s", ip, port)
	fmt.Printf("Attempting to start TCP server on %s\n", addr)
	var err error
	listener, err = net.Listen("tcp", addr)
	if err != nil {
		messages <- fmt.Sprintf("Error starting TCP server: %s\n", err)
		fmt.Printf("Error starting TCP server: %s\n", err)
		return
	}
	defer listener.Close()

	mu.Lock()
	tcpRunning = true
	done = make(chan struct{})
	mu.Unlock()

	messages <- fmt.Sprintf("TCP Server listening on %s\n", addr)
	fmt.Println("TCP Server listening on", addr)
	broadcastToWebSocketClients(fmt.Sprintf("TCP Server listening on %s\n", addr))

	go broadcastLoop()

	for {
		conn, err := listener.Accept()
		if err != nil {
			select {
			case <-done:
				fmt.Println("TCP server stopped")
				return
			default:
				fmt.Println("Error accepting connection:", err)
				continue
			}
		}
		go handleTCPConnection(conn, messages)
	}
}

func handleTCPConnection(conn net.Conn, messages chan<- string) {
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
		messages <- fmt.Sprintf("TCP Received: %s\n", message)

		ack := "ACK received\n"
		conn.Write([]byte(ack))

		broadcastToWebSocketClients(message)
	}
}

func broadcastLoop() {
	message := "IN:[<STX>1H|\\^&|||cobas-e411^1|||||host|TSREQ^REAL|P|1<CR>Q|1|^^466716165^13^0^2^^S1^SC||ALL||||||||O<CR>L|1|N<CR><ETX>49<CR><LF>]"
	for {
		select {
		case <-done:
			fmt.Println("Exiting broadcast loop")
			return
		default:
			fmt.Println("Broadcasting message to WebSocket clients")
			broadcastToWebSocketClients(message)
			time.Sleep(1 * time.Second) // Adjust the frequency as needed
		}
	}
}

func broadcastToWebSocketClients(message string) {
	wsConnections.RLock()
	defer wsConnections.RUnlock()
	fmt.Printf("Broadcasting to %d clients\n", len(wsConnections.connections))
	for _, conn := range wsConnections.connections {
		err := conn.WriteMessage(websocket.TextMessage, []byte(message))
		if err != nil {
			fmt.Printf("Error sending message to WebSocket client: %s\n", err)
		}
	}
}

func main() {
	go startWebSocketServer()

	http.HandleFunc("/start", func(w http.ResponseWriter, r *http.Request) {
		ip := r.URL.Query().Get("ip")
		tcpPort := r.URL.Query().Get("tcpPort")
		if ip == "" || tcpPort == "" {
			http.Error(w, "IP and TCP port are required", http.StatusBadRequest)
			return
		}

		messages := make(chan string)
		var wg sync.WaitGroup
		wg.Add(1)

		go startTCPServer(ip, tcpPort, &wg, messages)

		go func() {
			wg.Wait()
			close(messages)
		}()

		w.Header().Set("Content-Type", "text/plain")
		for msg := range messages {
			w.Write([]byte(msg))
		}
	})

	http.HandleFunc("/stop", func(w http.ResponseWriter, r *http.Request) {
		ip := r.URL.Query().Get("ip")
		tcpPort := r.URL.Query().Get("tcpPort")
		addr := fmt.Sprintf("%s:%s", ip, tcpPort)

		mu.Lock()
		if !tcpRunning {
			mu.Unlock()
			w.Write([]byte("TCP server not running"))
			return
		}
		tcpRunning = false
		close(done)
		mu.Unlock()
		if listener != nil {
			listener.Close()
			fmt.Printf("TCP server stopped on %s\n", addr)
			broadcastToWebSocketClients(fmt.Sprintf("TCP server stopped on %s\n", addr))
		}
		w.Write([]byte(fmt.Sprintf("TCP server stopped on %s", addr)))
	})

	fmt.Println("Control server started at :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		fmt.Printf("Error starting control server: %s\n", err)
	}
}
