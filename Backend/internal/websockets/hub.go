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
		roomId: GenerateRoomId(),
		clients: make(map[*Clients]bool),
		broadcast: make(chan Message, 100), //buffered channel to prevent deadlock
		register: make(chan *Clients, 5),
		unregistered: make(chan *Clients, 10),
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
		logger.Logger.Info("[Hub Manager]: Found the requested hub", "roomId", roomID)
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

	logger.Logger.Info("[Hub Manager]: Created new hub", "roomId", newHub.roomId)
	return newHub
}

//Deleting the empty hub
func (m *HubManager) DeleteHub(roomId string){
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.hubs, roomId)
	logger.Logger.Info("[Hub Manager]: successuflly deleated hub", "roomId", roomId)
}

//Run is the main event loop 
func (h *Hub)Run(){
	for{ 
		//it might result in deadlock (empty select)
		select { 
		case client := <-h.register:
			h.clients[client] = true
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
				logger.Logger.Info("[Hub Manager]: Deleting hub due to no people in hub",
														 "roomId", h.roomId)
				h.hubManager.DeleteHub(h.roomId)
				return 
			}

		case msg := <-h.broadcast: 
			logger.Logger.Info("Message received",
        "room_id", h.roomId,
        "type", string(msg.Type),
        "sender", msg.Sender,
        "content_preview", string(msg.Content)[:min(50, len(msg.Content))],
    	)
			if err := ValidateMessage(&msg); err != nil {
				logger.Logger.Warn("Message validation failed",
          "room_id", h.roomId,
          "sender", msg.Sender,
          "error", err,
        )
				continue
			}
			messageHandeling(msg, h)	
			logger.Logger.Info("[Hub]: Message handeled successfully")
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

func (h *Hub) BroadcastPlayerList() {
    players := make([]map[string]interface{}, 0, len(h.clients))
    for client := range h.clients {
        players = append(players, map[string]interface{}{
            "name":  client.name,
            "ready": client.ready,
        })
    }

    // Step 1: Marshal players to []byte
    data, err := json.Marshal(players)
    if err != nil {
				logger.Logger.Warn("[Hub]: Failed to marshal player list")
        return
    }

    // Step 2: Convert []byte to string
    contentStr := string(data)  // ← CORRECT: string(data)

    // Step 3: Marshal string to JSON (adds quotes + escapes)
    contentJSON, err := json.Marshal(contentStr)
    if err != nil {
				logger.Logger.Warn("[Hub]: Failed to marshal content string")
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
				logger.Logger.Warn("[Hub]: Broadcast channel full")
    }
}
