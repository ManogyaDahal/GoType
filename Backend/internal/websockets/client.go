package websockets

import (
	"encoding/json"
	"net"
	"time"
	"log"

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
	log.Printf("[ReadPump] ▶ Started for user: %s", c.name)
	defer func() {
		log.Printf("[ReadPump] Closing connection for user: %s", c.name)
		c.hub.unregistered <- c
		c.connection.Close()
	}()

	c.connection.SetReadLimit(4096) //4kb
	_ = c.connection.SetReadDeadline(time.Now().Add(pongWait))
	c.connection.SetPongHandler(func(string) error {
		log.Printf("[ReadPump] Pong received from %s", c.name)
		_ = c.connection.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {

		log.Printf("[ReadPump]:Before")
		_, data, err := c.connection.ReadMessage()
		log.Printf("[ReadPump]:After")

		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				c.hub.ErrorReport(c, "read", Error, "unexpected close", err)
			} else if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				c.hub.ErrorReport(c, "read", Error, "normal closure", err)
			} else if ne, ok := err.(net.Error); ok && ne.Timeout() {
				c.hub.ErrorReport(c, "read", Error, "client timeout", err)
			} else {
				c.hub.ErrorReport(c, "read", Error, "read error", err)
			}
			log.Printf("[ReadPump]  Read loop breaking for user: %s | err: %v", c.name, err)
			break
		}

		log.Printf("[ReadPump] Message received from %s: %s", c.name, string(data))

		var message Message
		if err := json.Unmarshal(data, &message); err != nil {
			log.Printf("[ReadPump]Got error in unmarshal")
			c.hub.ErrorReport(c, "read", "error", "Error in json Unmarshal", err)
			continue
		}
		message.Sender = c.name
		message.RoomId = c.hub.roomId
		message.TimeStamp = time.Now()
    log.Printf("[ReadPump] 📤 Sending to broadcast channel - Type: %s", message.Type)
		c.hub.broadcast <- message
	}
}

// continously sends the message from the `send` channel to websocket.
// (ie. output: server ->client )
func (c *Clients) WritePump() {
	log.Printf("[WritePump] ▶ Started for user: %s", c.name)
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.connection.Close()
		log.Printf("[WritePump] Closing connection for user: %s", c.name)
	}()

	for {
		select {
		case message, ok := <-c.send:
			_ = c.connection.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				_ = c.connection.WriteMessage(websocket.CloseMessage, []byte{})
				c.hub.ErrorReport(c, "write", Warning, "send channel closed by hub", nil)
				log.Printf("[WritePump] Send channel closed for user: %s", c.name)
				return
			}

			w, err := c.connection.NextWriter(websocket.TextMessage)
			if err != nil {
				c.hub.ErrorReport(c, "write", Error, "write setup error", err)
				log.Printf("[WritePump] Write setup error for user: %s | err: %v", c.name, err)
				return
			}

			log.Printf("[WritePump] 📨 Sending message to %s: %s", c.name, string(message))
			if _, err := w.Write(message); err != nil {
				c.hub.ErrorReport(c, "write", Error, "write error", err)
				log.Printf("[WritePump] Write error for user: %s | err: %v", c.name, err)
				_ = w.Close()
				return
			}

			n := len(c.send)
			for i := 0; i < n; i++ {
				nextMsg := <-c.send
				if _, err := w.Write(nextMsg); err != nil {
					c.hub.ErrorReport(c, "write", Error, "message queue write failed", err)
					log.Printf("[WritePump] Message queue write failed for user: %s | err: %v", c.name, err)
					break
				}
			}

			if err := w.Close(); err != nil {
				c.hub.ErrorReport(c, "write", Error, "writer close error", err)
				log.Printf("[WritePump] Writer close error for user: %s | err: %v", c.name, err)
				return
			}

		case <-ticker.C:
			_ = c.connection.SetWriteDeadline(time.Now().Add(writeWait))
			log.Printf("[WritePump] Sending ping to user: %s", c.name)
			if err := c.connection.WriteMessage(websocket.PingMessage, []byte{}); err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					c.hub.ErrorReport(c, "write", Error, "ping failed, connection lost", err)
				} else if ne, ok := err.(net.Error); ok && ne.Timeout() {
					c.hub.ErrorReport(c, "write", Error, "Ping timeout", err)
				} else {
					c.hub.ErrorReport(c, "write", Error, "write pump error", err)
				}
				log.Printf("[WritePump] Ping failed for user: %s | err: %v", c.name, err)
				return
			}
		}
	}
}
