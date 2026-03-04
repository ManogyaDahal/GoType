package websockets

import (
	"encoding/json"
	"net"
	"time"

	"github.com/ManogyaDahal/GoType/internal/logger"
	"github.com/gorilla/websocket"
)

// Player status constants
const (
	StatusIdle   = "idle"    // default state in lobby
	StatusReady  = "ready"   // player has readied up
	StatusInGame = "in_game" // player is currently in a game
)

//Client represents one connected websocket user
type Clients struct{
	hub 	   *Hub				// refrence to hub
	connection *websocket.Conn //actual websocket connection
	send       chan []byte// channel for outgoing messages
	name 	   string 		   //name of the client
	status     string          // player status: "idle", "ready", "in_game"
}

const (
	writeWait  = 10*time.Second    //how long to wait before timing out on writes
	pongWait   = 60*time.Second    //how long to wait for next pong message
	pingPeriod = (pongWait * 9)/10 //how often to send ping messages
)

//Reads the message from client and
//Broadcasts it to the hub's broadcast channel (input ie. client->server)
func (c *Clients) ReadPump() {
	logger.Logger.Info("[ReadPump] Connection started", "user", c.name)
	defer func() {
		logger.Logger.Warn("[ReadPump] Connection closing", "user", c.name)
		c.hub.unregistered <- c
		c.connection.Close()
	}()

	c.connection.SetReadLimit(4096) //4kb
	_ = c.connection.SetReadDeadline(time.Now().Add(pongWait))
	c.connection.SetPongHandler(func(string) error {
		logger.Logger.Debug("[ReadPump] Pong received", "user", c.name)
		_ = c.connection.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, data, err := c.connection.ReadMessage()

		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				c.hub.EventReport(c, "read", Error, "unexpected close", err)
			} else if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				c.hub.EventReport(c, "read", Error, "normal closure", err)
			} else if ne, ok := err.(net.Error); ok && ne.Timeout() {
				c.hub.EventReport(c, "read", Error, "client timeout", err)
			} else {
				c.hub.EventReport(c, "read", Error, "read error", err)
			}
			logger.Logger.Warn("[ReadPump] Read loop breaking", "user", c.name, "error", err)
			break
		}

		var message Message
		if err := json.Unmarshal(data, &message); err != nil {
			logger.Logger.Error("[ReadPump] JSON unmarshal failed", "error", err)
			c.hub.EventReport(c, "read", "error", "Error in json Unmarshal", err)
			continue
		}
		message.Sender = c.name
		message.RoomId = c.hub.roomId
		message.TimeStamp = time.Now()
		logger.Logger.Debug("[ReadPump] Message forwarded to broadcast", "type", message.Type, "user", c.name)
		c.hub.broadcast <- message
	}
}

// continously sends the message from the `send` channel to websocket.
// (ie. output: server ->client )
func (c *Clients) WritePump() {
	logger.Logger.Info("[WritePump] Started", "user", c.name)
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.connection.Close()
		logger.Logger.Warn("[WritePump] Connection closing", "user", c.name)
	}()

	for {
		select {
		case message, ok := <-c.send:
			_ = c.connection.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				_ = c.connection.WriteMessage(websocket.CloseMessage, []byte{})
				c.hub.EventReport(c, "write", Warning, "send channel closed by hub", nil)
				logger.Logger.Warn("[WritePump] Send channel closed", "user", c.name)
				return
			}

			logger.Logger.Debug("[WritePump] Sending message", "user", c.name, "size", len(message))
			// Send each message as its own WebSocket frame.
			// The old approach used NextWriter + batching loop which concatenated
			// multiple JSON objects into a single frame without delimiters,
			// causing JSON.parse failures on the client (messages silently lost).
			if err := c.connection.WriteMessage(websocket.TextMessage, message); err != nil {
				c.hub.EventReport(c, "write", Error, "write error", err)
				logger.Logger.Error("[WritePump] Write error", "user", c.name, "error", err)
				return
			}

		case <-ticker.C:
			_ = c.connection.SetWriteDeadline(time.Now().Add(writeWait))
			logger.Logger.Debug("[WritePump] Sending ping", "user", c.name)
			if err := c.connection.WriteMessage(websocket.PingMessage, []byte{}); err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					c.hub.EventReport(c, "write", Error, "ping failed, connection lost", err)
				} else if ne, ok := err.(net.Error); ok && ne.Timeout() {
					c.hub.EventReport(c, "write", Error, "Ping timeout", err)
				} else {
					c.hub.EventReport(c, "write", Error, "write pump error", err)
				}
				logger.Logger.Error("[WritePump] Ping failed", "user", c.name, "error", err)
				return
			}
		}
	}
}
