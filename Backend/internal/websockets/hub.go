// This file defines the central hub. Global event loop

package websockets

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"sync"
	"time"

	"github.com/ManogyaDahal/GoType/internal/logger"
)

// Grace period before deleting an empty hub.
// This prevents race conditions when all players navigate from lobby → game
// (all WS connections close briefly, then reconnect on the game page).
const hubDeletionGracePeriod = 10 * time.Second

//Manages all hubs
type HubManager struct{
	hubs map [string]*Hub //Stores the key for a hub
	mu   sync.RWMutex	  //for concurrency safety
}

// Hub manages all central websocket connection with clients.
// and handles all broadcasting messages between them
type Hub struct{
	roomId 		 string
	// All connected clients
	clients 	 map[*Clients]bool

	//channel for broadcasting incoming messages
	broadcast 	 chan Message

	//channel for registering and unregistering clients
	register     chan *Clients
	unregistered chan *Clients

	//Hub manager
	hubManager  *HubManager

	// Timer for delayed hub deletion (grace period)
	deleteTimer  *time.Timer

	// Game countdown state
	gameJoinedPlayers  map[string]bool // players who sent player_joined_game
	expectedPlayers    int             // snapshot of client count when game navigation started
	gameCountdownActive bool           // true while countdown goroutine is running
}

//new hub manager
func NewHubManager() *HubManager {
	return &HubManager{
		hubs:make(map[string]*Hub) ,
	}
}

// Initializes a new hub
func NewHub() *Hub {
	return &Hub{
		roomId:            GenerateRoomId(),
		clients:           make(map[*Clients]bool),
		broadcast:         make(chan Message, 100), //buffered channel to prevent deadlock
		register:          make(chan *Clients, 5),
		unregistered:      make(chan *Clients, 10),
		gameJoinedPlayers: make(map[string]bool),
	}
}

//Generating a random roomId
func GenerateRoomId() string{
	data := make([]byte, 16)
	rand.Read(data)
	return hex.EncodeToString(data)
}

// If generated roomId matches to any of the existing ones it returns
//the new generated roomId, else it returns the roomId that is passed as
//parameter
func (h *HubManager)CheckIfRoomAlreadyExists(roomId string) string {
	for {
		if _, ok := h.hubs[roomId]; !ok{
			return roomId
		}
		roomId = GenerateRoomId()
	}
}

// Get hub returns the hub to access specific hub
func (m *HubManager) GetExistringHub(roomID string) *Hub{
	m.mu.RLock() // for read lock
	defer m.mu.RUnlock()
	if hub, exists := m.hubs[roomID]; exists{
		logger.Logger.Debug("[HubManager] Found requested hub", "roomId", roomID)
		return hub
	}
	return nil
}


func (m *HubManager) CreateNewHub() *Hub {
	//If hub is not found we create new hub by doing rw lock
	m.mu.Lock()
	defer m.mu.Unlock()

	newHub := NewHub()
	newHub.roomId = m.CheckIfRoomAlreadyExists(newHub.roomId)
	// CHANGED: Set hubManager reference so hub can delete itself when empty
	newHub.hubManager = m
	m.hubs[newHub.roomId] = newHub
	go newHub.Run()

	logger.Logger.Info("[HubManager] Created new hub", "roomId", newHub.roomId)
	return newHub
}

//Deleting the empty hub
func (m *HubManager) DeleteHub(roomId string){
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.hubs, roomId)
	logger.Logger.Info("[HubManager] Deleted hub", "roomId", roomId)
}

//Run is the main event loop
func (h *Hub)Run(){
	for{
		//it might result in deadlock (empty select)
		select {
		case client := <-h.register:
			h.clients[client] = true
			// Cancel any pending deletion timer — a player reconnected
			if h.deleteTimer != nil {
				h.deleteTimer.Stop()
				h.deleteTimer = nil
				logger.Logger.Info("[Hub] Cancelled deletion timer, player reconnected",
					"roomId", h.roomId,
					"client", client.name,
				)
			}
			h.BroadcastPlayerList()
			h.EventReport(client, "[hub]", Info, "NewClient registered", nil)
			// SendSystemMessages(UserJoinedSysMessage, client, h)

		case client := <-h.unregistered:
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send) //edit
				h.BroadcastPlayerList()
				h.EventReport(client, "[hub]", Warning, "Client Unregistered", nil)
				// SendSystemMessages(UserLeftSysMessage, client, h)
			}
			if len(h.clients) == 0 && h.hubManager != nil{
				// Don't delete immediately — start a grace period timer.
				// Players navigating from lobby to game will reconnect within seconds.
				logger.Logger.Info("[Hub] All clients left, starting deletion grace period",
					"roomId", h.roomId,
					"grace", hubDeletionGracePeriod,
				)
				if h.deleteTimer != nil {
					h.deleteTimer.Stop()
				}
				h.deleteTimer = time.AfterFunc(hubDeletionGracePeriod, func() {
					// Re-check if still empty before deleting
					// (can't access h.clients directly from goroutine, so send a signal)
					// We use a dummy unregister with nil to trigger the final check
					select {
					case h.unregistered <- nil:
					default:
					}
				})
			}
			// Handle the deferred deletion check (nil client means timer fired)
			if client == nil && len(h.clients) == 0 && h.hubManager != nil {
				logger.Logger.Info("[HubManager] Deleting hub after grace period (still empty)",
					"roomId", h.roomId)
				h.hubManager.DeleteHub(h.roomId)
				return
			}

		case msg := <-h.broadcast:
			logger.Logger.Debug("[Hub] Message received",
				"room_id", h.roomId,
				"type", msg.Type,
				"sender", msg.Sender,
			)
			if err := ValidateMessage(&msg); err != nil {
				logger.Logger.Warn("[Hub] Message validation failed",
					"room_id", h.roomId,
					"sender", msg.Sender,
					"error", err,
				)
				continue
			}
			messageHandeling(msg, h)
		}
	}
}

//For error reports
func (h *Hub) EventReport(c *Clients, src string, sev Severity, msg string, err error) {
    clientName := "unknown"
    if c != nil && c.name != "" {
        clientName = c.name
    }

    switch sev {
    case Info:
        logger.Logger.Info(msg,
            "room_id", h.roomId,
            "client", clientName,
            "source", src,
            "error", err,
        )
    case Warning:
        logger.Logger.Warn(msg,
            "room_id", h.roomId,
            "client", clientName,
            "source", src,
            "error", err,
        )
    case Error, Fatal:
        logger.Logger.Error(msg,
            "room_id", h.roomId,
            "client", clientName,
            "source", src,
            "error", err,
        )

        if sev == Fatal {
					//do something
        }
    }
}

// ResetGameState clears game countdown state so a new round can start fresh.
func (h *Hub) ResetGameState() {
	h.gameJoinedPlayers = make(map[string]bool)
	h.expectedPlayers = 0
	h.gameCountdownActive = false
}

// PlayerJoinedGame marks a player as having arrived on the game page.
// When all expected players have joined, it kicks off the countdown.
func (h *Hub) PlayerJoinedGame(playerName string) {
	h.gameJoinedPlayers[playerName] = true

	// If we don't have an expected count yet, snapshot the current client count.
	// This handles the first player_joined_game arriving.
	if h.expectedPlayers == 0 {
		h.expectedPlayers = len(h.clients)
	}

	logger.Logger.Info("[Hub] Player joined game",
		"player", playerName,
		"joined", len(h.gameJoinedPlayers),
		"expected", h.expectedPlayers,
		"roomId", h.roomId,
	)

	// If all expected players are in, send game_go (only once)
	if len(h.gameJoinedPlayers) >= h.expectedPlayers && !h.gameCountdownActive {
		h.gameCountdownActive = true
		h.sendGameGo()
	}
}

// sendGameGo sends a single game_go message with start_time set 3 seconds
// in the future. The client handles the 3→2→1 countdown display locally
// based on the shared timestamp. This replaces the old goroutine approach
// that sent 4 separate messages (countdown 3, 2, 1, then go) which was
// prone to dropped messages causing clients to get stuck at countdown "1".
func (h *Hub) sendGameGo() {
	// Start time is 3 seconds from now — gives clients time to show 3-2-1
	const countdownDuration = 3 * time.Second
	startTime := time.Now().Add(countdownDuration).UnixMilli()

	content, _ := json.Marshal(map[string]int64{"start_time": startTime})
	msg := Message{
		Type:      GameGo,
		RoomId:    h.roomId,
		Sender:    "server",
		Content:   json.RawMessage(content),
		TimeStamp: time.Now(),
	}

	select {
	case h.broadcast <- msg:
	default:
		logger.Logger.Warn("[Hub] game_go: broadcast channel full", "roomId", h.roomId)
	}

	logger.Logger.Info("[Hub] Game go sent",
		"roomId", h.roomId,
		"start_time", startTime,
	)
}

func (h *Hub) BroadcastPlayerList() {
    players := make([]map[string]interface{}, 0, len(h.clients))
    for client := range h.clients {
        status := client.status
        if status == "" {
            status = StatusIdle
        }
        players = append(players, map[string]interface{}{
            "name":   client.name,
            "ready":  status == StatusReady, // backward compat for any code still checking .ready
            "status": status,
        })
    }

    // Step 1: Marshal players to []byte
    data, err := json.Marshal(players)
    if err != nil {
				logger.Logger.Warn("[Hub] Failed to marshal player list", "error", err)
        return
    }

    // Step 2: Convert []byte to string
    contentStr := string(data)  // ← CORRECT: string(data)

    // Step 3: Marshal string to JSON (adds quotes + escapes)
    contentJSON, err := json.Marshal(contentStr)
    if err != nil {
				logger.Logger.Warn("[Hub] Failed to marshal content string", "error", err)
        return
    }

    msg := Message{
        Type:      "player_list",
        Content:   json.RawMessage(contentJSON),
        RoomId:    h.roomId,
        TimeStamp: time.Now(),
    }

    select {
    case h.broadcast <- msg:
    default:
				logger.Logger.Warn("[Hub] Broadcast channel full, dropping player_list", "roomId", h.roomId)
    }
}
