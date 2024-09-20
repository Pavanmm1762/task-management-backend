package controllers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/go/task_management/backend/utils"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// Allow all connections by default
		return true
	},
}

var clients = make(map[string]*websocket.Conn)
var clientsMu sync.Mutex

func HandleWebSocket(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		fmt.Println(err)
		return
	}

	senderUUID := c.Query("sender")
	recipientUUID := c.Query("recipient")

	clientsMu.Lock()
	clients[senderUUID] = conn
	clientsMu.Unlock()

	for {
		messageType, p, err := conn.ReadMessage()
		if err != nil {
			fmt.Println(err)
			clientsMu.Lock()
			delete(clients, senderUUID)
			clientsMu.Unlock()
			return
		}

		var message utils.Message
		if err := json.Unmarshal(p, &message); err != nil {
			fmt.Println(err)
			return
		}

		// Broadcast the message to the recipient
		clientsMu.Lock()
		recipientConn, ok := clients[recipientUUID]
		clientsMu.Unlock()

		if ok {
			responseBytes, err := json.Marshal(message)
			if err != nil {
				fmt.Println(err)
				return
			}

			if err := recipientConn.WriteMessage(messageType, responseBytes); err != nil {
				fmt.Println(err)
				return
			}
		}
	}
}
