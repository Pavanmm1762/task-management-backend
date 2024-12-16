// server.go

package webSocket

import (
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

// WebSocketServer holds the clients and synchronization mechanism
type WebSocketServer struct {
	Clients map[string]*websocket.Conn // Map of userID to WebSocket connections
	Mutex   sync.Mutex
}

var WSServer = &WebSocketServer{
	Clients: make(map[string]*websocket.Conn),
}

// upgrader is used to upgrade HTTP connections to WebSocket
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// Allow only specific origins (modify for production use)
		allowedOrigins := map[string]bool{
			"http://localhost:3000":               true,
			"https://project-pioneer.netlify.app": true,
		}
		origin := r.Header.Get("Origin")
		return allowedOrigins[origin]
	},
}

// HandleWebSocket is the handler for incoming WebSocket connections
func (wss *WebSocketServer) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Failed to upgrade connection from %s: %v", r.RemoteAddr, err)
		return
	}

	// Read userID from query params
	userID := r.URL.Query().Get("userID")
	if userID == "" {
		http.Error(w, "User ID is required", http.StatusBadRequest)
		conn.Close()
		return
	}

	// Ensure cleanup on disconnect
	defer func() {
		wss.Mutex.Lock()
		delete(wss.Clients, userID)
		wss.Mutex.Unlock()
		conn.Close()
		fmt.Printf("Connection closed for user %s", userID)
	}()

	// Store the connection in the server's Clients map
	wss.Mutex.Lock()
	wss.Clients[userID] = conn
	wss.Mutex.Unlock()
	fmt.Printf("New connection established for user %s", userID)

	// Set pong handler for connection health check
	conn.SetPongHandler(func(appData string) error {
		log.Printf("Pong received from user %s", userID)
		return nil
	})

	// Keep the connection open and handle incoming messages
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			log.Printf("Connection error for user %s: %v", userID, err)
			break
		}
	}
}

// SendMessage sends a real-time WebSocket notification to a specific user
func (wss *WebSocketServer) SendMessage(userID, message string) error {
	wss.Mutex.Lock()
	defer wss.Mutex.Unlock()

	conn, exists := wss.Clients[userID]
	if !exists {
		return fmt.Errorf("WebSocket connection not found for user %s", userID)
	}

	err := conn.WriteMessage(websocket.TextMessage, []byte(message))
	if err != nil {
		log.Printf("Error sending message to %s: %v", userID, err)
		delete(wss.Clients, userID) // Remove if sending message fails
		return fmt.Errorf("failed to send message: %v", err)
	}

	return nil
}

// Shutdown gracefully shuts down the WebSocket server by closing all connections
func (wss *WebSocketServer) Shutdown() {
	wss.Mutex.Lock()
	defer wss.Mutex.Unlock()
	for userID, conn := range wss.Clients {
		conn.Close()
		delete(wss.Clients, userID)
		log.Printf("Closed connection for user %s", userID)
	}
	fmt.Printf("WebSocket server shutdown complete")
}

// StartWebSocketServer starts the WebSocket server on the specified port
func StartWebSocketServer() {
	http.HandleFunc("/ws", WSServer.HandleWebSocket)
	go func() {
		if err := http.ListenAndServe(":8081", nil); err != nil {
			log.Fatal("WebSocket server failed to start:", err)
		}
	}()
	fmt.Printf("websocket server is running on port 8081")

}
