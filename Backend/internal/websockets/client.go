package websockets

import (
	"encoding/json"
	"net"
	"time"

	"github.com/ManogyaDahal/GoType/internal/logger"
	"github.com/gorilla/websocket"
)

//Client represents one connected websocket user
type Clients struct{
	hub 	   *Hub				// refrence to hub
	connection *websocket.Conn //actual websocket connection
	send       chan []byte// channel for outgoing messages
	name 	   string 		   //name of the client
	ready      bool// ready and not-ready states (true is ready false is not-ready)
}

const (
	writeWait  = 10*time.Second    //how long to wait before timing out on writes
	pongWait   = 60*time.Second    //how long to wait for next pong message
	pingPeriod = (pongWait * 9)/10 //how often to send ping messages
)

//Reads the message from client and
//Broadcasts it to the hub's broadcast channel (input ie. client->server)
func (c *Clients) ReadPump() {
	logger.Logger.Info("[ReadPump]: Starting connection",
											"user", c.name)
	defer func() {
	logger.Logger.Warn("[ReadPump]: Closing connection",
											"user", c.name)
		c.hub.unregistered <- c
		c.connection.Close()
	}()

	c.connection.SetReadLimit(4096) //4kb
	_ = c.connection.SetReadDeadline(time.Now().Add(pongWait))
	c.connection.SetPongHandler(func(string) error {
		logger.Logger.Info("[ReadPump] Pong received from %s", c.name)
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
			logger.Logger.Warn("[ReadPump]  Read loop breaking for user: %s | err: %v", c.name, err)
			break
		}

		var message Message
		if err := json.Unmarshal(data, &message); err != nil {
			logger.Logger.Error("[ReadPump]Got error in unmarshal", "error", err)
			c.hub.EventReport(c, "read", "error", "Error in json Unmarshal", err)
			continue
		}
		message.Sender = c.name
		message.RoomId = c.hub.roomId
		message.TimeStamp = time.Now()
    logger.Logger.Info("[ReadPump] Sending to broadcast channel - Type: %s", message.Type)
		c.hub.broadcast <- message
	}
}

// continously sends the message from the `send` channel to websocket.
// (ie. output: server ->client )
func (c *Clients) WritePump() {
	logger.Logger.Info("[WritePump] ▶ Started for user: %s", c.name)
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.connection.Close()
		logger.Logger.Warn("[WritePump] Closing connection for user: %s", c.name)
	}()

	for {
		select {
		case message, ok := <-c.send:
			_ = c.connection.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				_ = c.connection.WriteMessage(websocket.CloseMessage, []byte{})
				c.hub.EventReport(c, "write", Warning, "send channel closed by hub", nil)
				logger.Logger.Warn("[WritePump] Send channel closed for user: %s", c.name)
				return
			}

			w, err := c.connection.NextWriter(websocket.TextMessage)
			if err != nil {
				c.hub.EventReport(c, "write", Error, "write setup error", err)
				logger.Logger.Error("[WritePump] Write setup error for user: %s | err: %v", c.name, err)
				return
			}

			logger.Logger.Info("[WritePump]: Sending message to %s: %s", c.name, string(message))
			if _, err := w.Write(message); err != nil {
				c.hub.EventReport(c, "write", Error, "write error", err)
				logger.Logger.Error("[WritePump] Write error for user: %s | err: %v", c.name, err)
				_ = w.Close()
				return
			}

			n := len(c.send)
			for i := 0; i < n; i++ {
				nextMsg := <-c.send
				if _, err := w.Write(nextMsg); err != nil {
					c.hub.EventReport(c, "write", Error, "message queue write failed", err)
					logger.Logger.Error("[WritePump] Message queue write failed for user: %s | err: %v", c.name, err)
					break
				}
			}

			if err := w.Close(); err != nil {
				c.hub.EventReport(c, "write", Error, "writer close error", err)
				logger.Logger.Error("[WritePump] Writer close error for user: %s | err: %v", c.name, err)
				return
			}

		case <-ticker.C:
			_ = c.connection.SetWriteDeadline(time.Now().Add(writeWait))
			logger.Logger.Info("[WritePump] Sending ping to user: %s", c.name)
			if err := c.connection.WriteMessage(websocket.PingMessage, []byte{}); err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					c.hub.EventReport(c, "write", Error, "ping failed, connection lost", err)
				} else if ne, ok := err.(net.Error); ok && ne.Timeout() {
					c.hub.EventReport(c, "write", Error, "Ping timeout", err)
				} else {
					c.hub.EventReport(c, "write", Error, "write pump error", err)
				}
				logger.Logger.Error("[WritePump] Ping failed for user: %s | err: %v", c.name, err)
				return
			}
		}
	}
}
