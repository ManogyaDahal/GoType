// This file defines the central hub. Global event loop 

package websockets
import(
	"time"
	"log"
)

// Hub manages all central websocket connection with clients. 
// and handles all broadcasting messages between them
type Hub struct{
	// All connected clients	
	clients 	 map[*Clients]bool

	//channel for broadcasting incoming messages
	broadcast 	 chan []byte

	//channel for registering and unregistering clients
	register     chan *Clients
	unregistered chan *Clients

	//Error management
	errors chan ErrorEvent
}


// Initializes a new hub
func NewHub() *Hub {
	return &Hub{
		clients: make(map[*Clients]bool),
		broadcast: make(chan []byte),
		register: make(chan *Clients),
		unregistered: make(chan *Clients),
		errors : make(chan ErrorEvent),
	}
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
