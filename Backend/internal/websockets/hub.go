// This file defines the central hub. Global event loop

package websockets

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"log"
	"sync"
	"time"
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

	//Error management
	errors chan ErrorEvent

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
		broadcast: make(chan Message, 100),
		register: make(chan *Clients, 5),
		unregistered: make(chan *Clients, 10),
		errors : make(chan ErrorEvent, 100), //buffered channel to prevent blocking 
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
		return hub	
	}  
	return nil
}


func (m *HubManager) CreateNewHub() *Hub {
	//If hub is not found we create new hub by doing rw lock
	m.mu.Lock()
	defer m.mu.Unlock()


	log.Printf("[HubManager]: Is making new Hub")
	newHub := NewHub()
	newHub.roomId = m.CheckIfRoomAlreadyExists(newHub.roomId)
	// CHANGED: Set hubManager reference so hub can delete itself when empty
	newHub.hubManager = m
	m.hubs[newHub.roomId] = newHub
	go newHub.Run()
	return newHub
}

//Deleting the empty hub
func (m *HubManager) DeleteHub(roomId string){
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.hubs, roomId)
	log.Printf("Removed the hub with the ID: %s", roomId)
}

//Run is the main event loop 
func (h *Hub)Run(){
	for{ 
		//it might result in deadlock (empty select)
		select { 
		case client := <-h.register:
			h.clients[client] = true
			log.Println("[Hub Run]: UserRegistered...")
			h.BroadcastPlayerList()
			log.Println("[Hub Run]: UserRegistered")
			SendSystemMessages(UserJoinedSysMessage, client, h)

		case client := <-h.unregistered:
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send) //edit
				h.BroadcastPlayerList()
				log.Println("[Hub Run]: User UnRegistered")
				SendSystemMessages(UserLeftSysMessage, client, h)
			}
			if len(h.clients) == 0 && h.hubManager != nil{
				log.Println("[Hub Run]: Hub Deleated")
				h.hubManager.DeleteHub(h.roomId)
				return //edit
			}

		case msg := <-h.broadcast: 
    	log.Printf("[Hub] 📨 Received message - Type: %s, Sender: %s", msg.Type, msg.Sender)
			if err := ValidateMessage(&msg); err != nil {
				log.Printf("[Hub] Message validation failed: %v", err)
				continue
			}
    	log.Printf("[Hub] ✅ Message valid, routing to handler")
			messageHandeling(msg, h)	
    	log.Printf("[Hub] ✅ Message handled")

		case errorEvent := <-h.errors:
			//centralized logging
			log.Printf("[%s] [%s] [client: %s] %s: %v\n", 
			errorEvent.Time.Format("15:04:05"),
			errorEvent.Source, 
			errorEvent.Client,
			errorEvent.Message,
			errorEvent.Error,
			)

			// can use switch for various severity [info], [warning]
			if errorEvent.Severity == "fatal"{
				//do something
			}
		}
	}
}

//For error reports
func (h *Hub) ErrorReport(c *Clients, src string, sev Severity, msg string, err error) {
		clientName := "unknown"
		if c != nil {
			clientName = c.name
		}

    select {
    case h.errors <- ErrorEvent{
        Time:      time.Now(),
        Client:    clientName,
        Source:    src,
        Severity:  sev,
        Message:   msg,
        Error:     err,
    }:
    default:
        log.Println("[Hub] Dropped error event (channel full)")
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
        log.Printf("Failed to marshal player list")
        return
    }

    // Step 2: Convert []byte to string
    contentStr := string(data)  // ← CORRECT: string(data)

    // Step 3: Marshal string to JSON (adds quotes + escapes)
    contentJSON, err := json.Marshal(contentStr)
    if err != nil {
        log.Printf("Failed to marshal content string")
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
        log.Printf("Broadcast channel full")
    }
}
