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

//Websocket handler which can be used by authenticated users.
//api eg: ws://localhost:8080/ws?action=join&room_id=abcd1234
//api eg: ws://localhost:8080/ws?action=create
func AuthenticatedWSHandler(m *HubManager) gin.HandlerFunc {
	//checking for valid websocket upgrade
	return func(c *gin.Context){
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

		//Retrieving the room Id and action from the url
		roomId := c.Query("room_id")
		action := Action(c.Query("action")) //converting into Action type

		//Ensure correct action is selected and actions are performed
		if !IsValidAction(action){
			c.JSON(http.StatusBadRequest,
			gin.H{"error":"Invlaid Action in the url"})
			return
		}
		var currentHub *Hub
		switch action{
		case ActionJoin:
			currentHub = m.GetExistringHub(roomId)
        	if currentHub == nil {
            	c.JSON(http.StatusNotFound, gin.H{"error": "Room not found"})
            	return
        	}
		}

		//upgrading connection from http to websocket
		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			log.Printf("[wshandler]: websocket upgrade failed")
			c.AbortWithStatus(http.StatusBadRequest)
			// Important: DO NOT call c.JSON() here.
			// WebSocket handshake already writes headers, so just return
			return
		}

		client := &Clients{
			hub: currentHub, 	
			connection: conn,
			send: make(chan []byte, 256),
			name: name.(string),
			ready: false,
		}

		// registering the client
		log.Printf("[wsHandler]: Regestering the client")
		client.hub.register <- client

		go client.ReadPump()  //client -> connection
		go client.WritePump() //connection -> client
	}
}

func CreateNewRoom(m *HubManager) gin.HandlerFunc{
	return  func (c *gin.Context){
		hub := m.CreateNewHub()
		c.JSON(http.StatusOK,
		gin.H{ "room_id":hub.roomId})
	}
}
