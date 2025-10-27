// This file defines the central hub. Global event loop 

package websockets

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
}

// Initializes a new hub
func NewHub() *Hub {
	return &Hub{
		clients: make(map[*Clients]bool),
		broadcast: make(chan []byte),
		register: make(chan *Clients),
		unregistered: make(chan *Clients),
	}
}


//Run is the main event loop 
func (h *Hub)Run(){
	for{ 
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
		}
	}
}
