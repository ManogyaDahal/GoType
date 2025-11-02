// This file defines the central hub. Global event loop

package websockets

import (
	"crypto/rand"
	"encoding/hex"
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
	broadcast 	 chan []byte

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
		broadcast: make(chan []byte),
		register: make(chan *Clients),
		unregistered: make(chan *Clients),
		errors : make(chan ErrorEvent),
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

		case client := <-h.unregistered:
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(h.unregistered)
			}
			if len(h.clients) == 0 && h.hubManager != nil{
				h.hubManager.DeleteHub(hub.roomId)
			}

		case message := <-h.broadcast: 
			for client := range h.clients {
				select{
				case client.send <- message:
				default: 
				// client channel full assume conn is broken
				close(client.send)
				delete(h.clients, client)
				}	
			}

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
    select {
    case h.errors <- ErrorEvent{
        Time:      time.Now(),
        Client: c.name,
        Source:    src,
        Severity:  sev,
        Message:   msg,
        Error:     err,
    }:
    default:
        log.Println("[Hub] Dropped error event (channel full)")
    }
}
