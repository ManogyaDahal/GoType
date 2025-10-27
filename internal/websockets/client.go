package websockets

import (
	"log"
	"time"

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

	c.connection.SetReadLimit(512)
	c.connection.SetReadDeadline(time.Now().Add(pongWait)) // handle error
	c.connection.SetPongHandler( func(string)error {
		c.connection.SetReadDeadline(time.Now().Add(pongWait))	
		return nil
	})

	for {
		_, message, err := c.connection.ReadMessage()
		if err != nil {
			log.Println("[Client]: read error")
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
			c.connection.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.connection.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.connection.NextWriter(websocket.TextMessage)
			if err != nil {
				return 
			}
			w.Write(message)

			//add queued message into same websocket frame
			for  range len(c.send) {
				w.Write([]byte{})
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil{
				return
			}

		case <- ticker.C:
			//send ping periodically to keep the connection alive 
			c.connection.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.connection.WriteMessage(websocket.PingMessage, []byte{}); err != nil {
				return
			}
		}
	}
}
