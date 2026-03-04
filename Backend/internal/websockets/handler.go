package websockets

import (
	"net/http"
	"os"

	"github.com/ManogyaDahal/GoType/internal/logger"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

func getAllowedOrigins() []string {
	frontendURL := os.Getenv("FRONTEND_URL")
	backendURL := os.Getenv("BACKEND_URL")

	origins := []string{}
	if frontendURL != "" {
		origins = append(origins, frontendURL)
	} else {
		origins = append(origins, "http://localhost:5173")
	}
	if backendURL != "" {
		origins = append(origins, backendURL)
	} else {
		origins = append(origins, "http://localhost:8080")
	}
	return origins
}

var upgrader = &websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		for _, allowed := range getAllowedOrigins() {
			if origin == allowed {
				return true
			}
		}
		logger.Logger.Error("[WS] Blocked connection from unauthorized",
			"origin", origin)
		return false
	},
}

// Websocket handler which can be used by authenticated users.
func AuthenticatedWSHandler(m *HubManager) gin.HandlerFunc {
	//checking for valid websocket upgrade
	return func(c *gin.Context) {
		if !websocket.IsWebSocketUpgrade(c.Request) {
			c.JSON(http.StatusBadRequest,
				gin.H{"error": "Expected websocket upgrade"})
			return
		}

		// fetching user email to check if user is logged in
		session := sessions.Default(c)
		name := session.Get("Name")
		if name == nil || name == "" {
			c.JSON(http.StatusUnauthorized,
				gin.H{"error": "The user is not currently logged in"})
			return
		}

		//Retrieving the room Id and action from the url
		roomId := c.Query("room_id")
		action := Action(c.Query("action")) //converting into Action type

		//Ensure correct action is selected and actions are performed
		if !IsValidAction(action) {
			c.JSON(http.StatusBadRequest,
				gin.H{"error": "Invalid Action in the url"})
			return
		}
		var currentHub *Hub
		switch action {
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
			logger.Logger.Error("[WS] Websocket upgrade failed", "error", err)
			c.AbortWithStatus(http.StatusBadRequest)
			// Important: DO NOT call c.JSON() here.
			// WebSocket handshake already writes headers, so just return
			return
		}

		client := &Clients{
			hub:        currentHub,
			connection: conn,
			send:       make(chan []byte, 256),
			name:       name.(string),
			status:     StatusIdle,
		}

		// registering the client
		client.hub.register <- client

		go client.ReadPump()  //client -> connection
		go client.WritePump() //connection -> client
	}
}

func CreateNewRoom(m *HubManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		hub := m.CreateNewHub()
		c.JSON(http.StatusOK,
			gin.H{"room_id": hub.roomId})
	}
}
