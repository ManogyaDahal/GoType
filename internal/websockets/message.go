package websockets

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

//Defines the attributes of messages to be sent
type Message struct{ 
	Type string `json:"type"`			   //Type of message private, broadcast
	RoomId string `json:"room_id"`		   //roomId: id of hub
	Sender string `json:"sender"`		   //Client's name
	Reciepitent string `json:"reciepitent"`//Reciever's name {if private message}
	Content string `json:"content"` 	   //content which the message hols
	TimeStamp time.Time`json:"timestamp"`  //Time of message arrival
}

//Type of the messages 
const (
	PrivateMessage string = "private"	   //one to one chat with clients
	BroadcastMessage string = "broadcast"  // 
	SystemMessage	 string = "string"     //Message sent by system(user joined.)
)

func encodeMessage(msg Message) []byte{
	data, _ := json.Marshal(msg)
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
				if client.name == message.Reciepitent || client.name == message.Sender{
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
