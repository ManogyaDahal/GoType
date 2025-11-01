// This file defines the central hub. Global event loop 

package websockets
import(
	"time"
	"log"
	"sync"
	"crypto/rand"
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

//NOTE: It might generate same roomId for two hubs so I have to manage that too
//Generating a random roomId
func GenerateRoomId() string{
	data := make([]byte, 16)
	rand.Read(data)
	return string(data)
}

// Get hub returns the hub to access specific hub
func (m *HubManager) GetHub(roomID string) *Hub{
	m.mu.RLock() // for read lock
	if hub, exists := m.hubs[roomID]; exists{
		m.mu.RUnlock()
		return hub	
	} else { m.mu.RUnlock() }

	//If hub is not found we create new hub by doing rw lock
	m.mu.Lock()
	defer m.mu.Unlock()

	//double checking for the hub  if another go routine made it
	if hub, exists := m.hubs[roomID]; exists{
		return hub	
	}

	log.Printf("[HubManager]: Is making new Hub")
	newHub := NewHub()
	m.hubs[roomID] = newHub
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
