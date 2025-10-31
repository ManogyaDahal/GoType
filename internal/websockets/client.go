package websockets

import (
	"time"
	"net"

	"github.com/gorilla/websocket"
)

//Client represents one connected websocket user
type Clients struct{
	hub 	   *Hub				// refrence to hub
	connection *websocket.Conn //actual websocket connection
	send       chan []byte 	   // channel for outgoing messages
	name 	   string 		   //name of the client
}

const (
	writeWait  = 10*time.Second    //how long to wait before timing out on writes
	pongWait   = 60*time.Second    //how long to wait for next pong message
	pingPeriod = (pongWait * 9)/10 //how often to send ping messages
)

//Reads the message from client and
//Broadcasts it to the hub's broadcast channel (input ie. client->server)
func (c *Clients) ReadPump() {
	defer func(){
		c.hub.unregistered <- c
		c.connection.Close()
	}()

	c.connection.SetReadLimit(512) //protects oversized messages
	_ = c.connection.SetReadDeadline(time.Now().Add(pongWait)) // handle error
	//reloads the timeout whenever client responds to pings
	c.connection.SetPongHandler( func(string)error {
		_ = c.connection.SetReadDeadline(time.Now().Add(pongWait))	
		return nil
	})

	for {
		_, message, err := c.connection.ReadMessage()

		// classifying the error to get specific error
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				c.hub.ErrorReport(c, "read", Error , "unexpected close",err)
			} else if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				c.hub.ErrorReport(c, "read", Error, "normal closure",err)
			} else if ne, ok := err.(net.Error); ok && ne.Timeout() {
				c.hub.ErrorReport(c, "read", Error, "client timeout",err)
			} else {
				c.hub.ErrorReport(c, "read", Error, "read error",err)
			}
			break	
		}
		c.hub.broadcast <- message
	}
}

// continously sends the message from the `send` channel to websocket.
// (ie. output: server ->client )
func(c *Clients) WritePump(){ 
	ticker := time.NewTicker(pingPeriod)
	defer func(){
		ticker.Stop()
		c.connection.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			_ = c.connection.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				_ = c.connection.WriteMessage(websocket.CloseMessage, []byte{})
				c.hub.ErrorReport(c, "write", Warning, "send channel closed by hub",nil)
				return
			}

			w, err := c.connection.NextWriter(websocket.TextMessage)
			if err != nil {
				c.hub.ErrorReport(c, "write", Error, "write setup error",err)
				return 
			}

			//write main message
			if _, err := w.Write(message); err != nil {
				c.hub.ErrorReport(c, "write", Error, "write error",err)
				_ = w.Close()
				return
			}

			//add queued message into same websocket frame
			for i := 0; i < len(c.send); i++ {
				nextMsg := <-c.send
				if _, err := w.Write(nextMsg); err != nil {
					c.hub.ErrorReport(c, "write", Error, "message queue write failed", err)
					break
				}
			}

			if err := w.Close(); err != nil{
				c.hub.ErrorReport(c, "write", Error, "writer close error",err)
				return
			}

		case <- ticker.C:
			//send ping periodically to keep the connection alive handeled by pong handler
			c.connection.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.connection.WriteMessage(websocket.PingMessage, []byte{}); err != nil {
				// classify error
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					c.hub.ErrorReport(c, "write", Error, "ping failed, connection lost",err)
				} else if ne, ok := err.(net.Error); ok && ne.Timeout() {
					c.hub.ErrorReport(c, "write", Error, "Ping timeout",err)
				} else {
					c.hub.ErrorReport(c, "write", Error, "write pump error",err)
				}
				return
			}
		}
	}
}
