package websockets

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"
)

//Defines the attributes of messages to be sent
type Message struct{ 
	Type string `json:"type"`			   //Type of message private, broadcast
	RoomId string `json:"room_id"`		   //roomId: id of hub
	Sender string `json:"sender"`		   //Client's name
	Reciever string `json:"reciever"`   //Reciever's name {if private message}
	Content string `json:"content"` 	   //content which the message hols
	TimeStamp time.Time`json:"timestamp"`  //Time of message arrival
}

//Type of the messages 
const (
	PrivateMessage string = "private"	   //one to one chat with clients
	BroadcastMessage string = "broadcast"  // 
	SystemMessage	 string = "string"     //Message sent by system(user joined.)
)

// type of system messages
const (
	UserJoinedSysMessage string = "userJoined"
	UserLeftSysMessage 	 string = "userLeft"
	NewHubCreated        string = "newHubCreated"
)

func encodeMessage(msg Message) []byte{
	data, err := json.Marshal(msg)
    if err != nil {
        log.Printf("[EncodeError] failed to encode message: %v", err)
        return []byte(`{"type":"error","content":"Internal server error"}`)
    }
	return data
}

//Based on the recieved message type (message.Type) recieved it preforms 
//specific operations
func messageHandeling(message Message, h *Hub){
	switch message.Type{
		case BroadcastMessage:
			//message broadcasting to all messages
			for client := range h.clients{ 
				client.send <- encodeMessage(message)
			}
		case PrivateMessage:
			//message for specific client
			for client := range h.clients{
				if client.name == message.Reciever{
					client.send <- encodeMessage(message)
					break
				}
			}
		case SystemMessage:
			//message for all clients
			for client := range h.clients{
				client.send <- encodeMessage(message)
			}
	}
}

//Validates the recieved message based on it's type
func ValidateMessage (msg *Message) error {
	switch msg.Type {
	case BroadcastMessage, SystemMessage, PrivateMessage:
		if strings.TrimSpace(msg.Content) == ""{
			return fmt.Errorf("Empty message content")
		}
	default:
		return fmt.Errorf("Invalid message Type")
	}
	return nil
}


// This is the function which sends the system messages i.e
//if new user has joind or an existing user disconnected
/* Sending system message could've done by frontend */
func SendSystemMessages(systemMessageType string, client *Clients, hub *Hub)  {
	var m Message 
	m.Sender = "System"
    m.TimeStamp = time.Now()
	switch systemMessageType{
	case UserJoinedSysMessage:
		m.Content = fmt.Sprintf("%s joined the room", client.name)
	case UserLeftSysMessage:
		m.Content = fmt.Sprintf("%s left the room", client.name)
	case NewHubCreated:
		m.Content = "New Hub Created"
	}

	hub.broadcast <- m
}
