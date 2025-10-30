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
	CheckOrigin: func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		//add frontend or prod domain here
		if origin == "http://localhost:5173" || origin == "http://localhost:8080" {
		return true 
		}
		log.Printf("[WS] Blocked connection from unauthorized origin: %s\n", origin)
		return false
	},
}

var hub = NewHub()
func Init(){
	go hub.Run()
}

func AuthenticatedWSHandler(c *gin.Context) {
	//checking for valid websocket upgrade
	if !websocket.IsWebSocketUpgrade(c.Request) {
		c.JSON(http.StatusBadRequest, 
		gin.H{"error":"Expected websocket upgrade"})
		return
	}

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
    	c.AbortWithStatus(http.StatusBadRequest)
        // Important: DO NOT call c.JSON() here.
        // WebSocket handshake already writes headers, so just return
		return
	}
	log.Println("[WS] Upgraded connection for:", name)
	defer func() { log.Println("[WS] Closed connection for:", name) }()

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
