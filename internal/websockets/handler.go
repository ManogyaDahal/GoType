package websockets

import (
	"log"
	"net/http"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)


var upgrader = &websocket.Upgrader{
	// Make origins stronglater
	CheckOrigin: func(r *http.Request) bool { return true },
}

var hub = NewHub()
func Init(){
	go hub.Run()
}

func AuthenticatedWSHandler(c *gin.Context) {
	// fetching user email to check if user is looged in
	session := sessions.Default(c)
	name := session.Get("Name")
	log.Println(name)
	if name == nil || name == ""{
		c.JSON(http.StatusUnauthorized, 
		gin.H {"error":"The user is not currently logged in"})
		return
	}

	//upgrading connection from http to websocket
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
        log.Printf("[WS] Upgrade failed: %v\n", err)
        // Important: DO NOT call c.JSON() here.
        // WebSocket handshake already writes headers, so just return
		return
	}

	client := &Clients{
		hub: hub, 	
		connection: conn,
		send: make(chan []byte, 256),
		name: name.(string),
	}

	// registering the client
	client.hub.register <- client

	go client.WritePump() //connection -> client
	go client.ReadPump()  //client -> connection
}
