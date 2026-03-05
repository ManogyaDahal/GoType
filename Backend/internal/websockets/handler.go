package websockets

import (
	"net/http"
	"os"

	"github.com/ManogyaDahal/GoType/internal/auth"
	"github.com/ManogyaDahal/GoType/internal/logger"
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
		logger.Logger.Error("[WS] Blocked connection from unauthorized origin",
			"origin", origin)
		return false
	},
}

// AuthenticatedWSHandler handles WebSocket connections for authenticated users.
// Authentication is done via a short-lived HMAC-signed token passed as ?token=
// in the URL. The token is issued by /api/whoamI (which runs through the Vercel
// proxy where the session cookie is valid), so the WebSocket can connect
// directly to Render without needing the session cookie at all.
func AuthenticatedWSHandler(m *HubManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !websocket.IsWebSocketUpgrade(c.Request) {
			c.JSON(http.StatusBadRequest,
				gin.H{"error": "Expected websocket upgrade"})
			return
		}

		// Validate the HMAC-signed ws_token from the query string.
		// This token was issued by WhoAmI and is valid for 2 hours.
		wsToken := c.Query("token")
		userName, valid := auth.ValidateWSToken(wsToken)
		if !valid || userName == "" {
			c.JSON(http.StatusUnauthorized,
				gin.H{"error": "Invalid or missing WebSocket token"})
			return
		}

		// Retrieve room ID and action from the URL
		roomId := c.Query("room_id")
		action := Action(c.Query("action"))

		if !IsValidAction(action) {
			c.JSON(http.StatusBadRequest,
				gin.H{"error": "Invalid action in the url"})
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

		// Upgrade HTTP connection to WebSocket
		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			logger.Logger.Error("[WS] WebSocket upgrade failed", "error", err)
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}

		client := &Clients{
			hub:        currentHub,
			connection: conn,
			send:       make(chan []byte, 256),
			name:       userName,
			status:     StatusIdle,
		}

		// Register the client with the hub
		client.hub.register <- client

		go client.ReadPump()
		go client.WritePump()
	}
}

func CreateNewRoom(m *HubManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		hub := m.CreateNewHub()
		c.JSON(http.StatusOK,
			gin.H{"room_id": hub.roomId})
	}
}
