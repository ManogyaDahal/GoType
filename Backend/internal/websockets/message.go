package websockets

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/ManogyaDahal/GoType/internal/logger"
)

// Defines the attributes of messages to be sent
type Message struct {
	Type      string          `json:"type"`      // Type of message private, broadcast
	RoomId    string          `json:"room_id"`   // roomId: id of hub
	Sender    string          `json:"sender"`    // Client's name
	Reciever  string          `json:"reciever"`  // Reciever's name {if private message}
	Content   json.RawMessage `json:"content"`   // content which the message holds
	TimeStamp time.Time       `json:"timestamp"` // Time of message arrival
}

// Type of the messages
const (
	PrivateMessage    string = "private"      // one to one chat with clients
	BroadcastMessage  string = "broadcast"    // broadcast to all clients
	SystemMessage     string = "string"       // Message sent by system (user joined.)
	PlayerListMessage string = "player_list"  // informs about the list of player
	PlayerReadyToggle string = "ready_toggle" // informs about the ready state

	// Game message types
	PlayerProgress    string = "player_progress"     // relay cursor position + WPM to other players
	GameFinished      string = "game_finished"       // relay when a player finishes the game
	GameStart         string = "game_start"          // broadcast game text to all players when game begins
	RequestPlayerList string = "request_player_list" // client requests a fresh player list
	ResetReady        string = "reset_ready"         // client asks server to set their ready state to false
	PlayerJoinedGame  string = "player_joined_game"  // client signals it has arrived on the game page
	GameCountdown     string = "game_countdown"      // server → clients: countdown tick (3, 2, 1)
	GameGo            string = "game_go"             // server → clients: game starts now (includes start_time)
)

// type of system messages
const (
	UserJoinedSysMessage string = "userJoined"
	UserLeftSysMessage   string = "userLeft"
	NewHubCreated        string = "newHubCreated"
)

func encodeMessage(msg Message) []byte {
	data, err := json.Marshal(msg)
	if err != nil {
		logger.Logger.Error("[EncodeError] Failed to encode message",
			"type", msg.Type,
			"error", err)
		return []byte(`{"type":"error","content":"Internal server error"}`)
	}
	return data
}

// Based on the received message type (message.Type) received it performs
// specific operations
func messageHandeling(message Message, h *Hub) {
	switch message.Type {
	case BroadcastMessage:
		// message broadcasting to all clients
		for client := range h.clients {
			if client.name == message.Sender {
				continue
			} // skip the broadcast to sender
			client.send <- encodeMessage(message)
		}

	case PrivateMessage:
		// message for specific client
		for client := range h.clients {
			if client.name == message.Reciever {
				client.send <- encodeMessage(message)
				break
			}
		}

	case SystemMessage:
		// message for all clients
		for client := range h.clients {
			client.send <- encodeMessage(message)
		}

	case PlayerListMessage:
		for client := range h.clients {
			client.send <- encodeMessage(message)
		}

	case PlayerReadyToggle:
		logger.Logger.Debug("[Game] ready_toggle received", "sender", message.Sender)
		// Read the desired state from the content instead of toggling
		var contentStr string
		if err := json.Unmarshal(message.Content, &contentStr); err == nil {
			for client := range h.clients {
				if client.name == message.Sender {
					if contentStr == "ready" {
						client.status = StatusReady
					} else {
						client.status = StatusIdle
					}
					h.BroadcastPlayerList()
					break
				}
			}
		}

	case RequestPlayerList:
		logger.Logger.Debug("[Game] request_player_list received", "sender", message.Sender)
		h.BroadcastPlayerList()

	case ResetReady:
		logger.Logger.Debug("[Game] reset_ready received", "sender", message.Sender)
		for client := range h.clients {
			if client.name == message.Sender {
				client.status = StatusIdle
				break
			}
		}
		h.BroadcastPlayerList()
		// Also reset the game state so a new round can start
		h.ResetGameState()

	case PlayerJoinedGame:
		// A client has arrived on the game page. Track it and start
		// the countdown once every connected player has checked in.
		// NOTE: Do NOT set status to "in_game" here — doing so would
		// broadcast a player_list that breaks allReady for players still
		// in the lobby countdown (race condition). Status is set to
		// "in_game" later when game_go fires (the actual start signal).
		h.PlayerJoinedGame(message.Sender)

	case PlayerProgress:
		// Relay player progress (cursor position, WPM) to all OTHER clients in the room
		logger.Logger.Debug("[Game] player_progress received",
			"sender", message.Sender,
			"room_id", h.roomId,
		)
		for client := range h.clients {
			if client.name == message.Sender {
				continue
			}
			client.send <- encodeMessage(message)
		}

	case GameFinished:
		// Relay game finished notification to all OTHER clients in the room
		logger.Logger.Info("[Game] game_finished received",
			"sender", message.Sender,
			"room_id", h.roomId,
		)
		for client := range h.clients {
			if client.name == message.Sender {
				continue
			}
			client.send <- encodeMessage(message)
		}

	case GameStart:
		// Broadcast game start (with text) to ALL clients including sender
		logger.Logger.Info("[Game] game_start received",
			"sender", message.Sender,
			"room_id", h.roomId,
		)
		for client := range h.clients {
			client.send <- encodeMessage(message)
		}

	case GameCountdown:
		// Server-generated countdown tick (3, 2, 1) — send to ALL clients
		logger.Logger.Debug("[Game] game_countdown",
			"room_id", h.roomId,
		)
		for client := range h.clients {
			client.send <- encodeMessage(message)
		}

	case GameGo:
		// Server-generated GO signal with start_time — send to ALL clients
		logger.Logger.Info("[Game] game_go",
			"room_id", h.roomId,
		)
		for client := range h.clients {
			client.send <- encodeMessage(message)
		}
		// NOW mark all players as "in_game". At this point every player
		// has arrived on the game page and the race is truly starting.
		// This prevents a returning-to-lobby player from triggering a
		// new game while others are still racing.
		for client := range h.clients {
			client.status = StatusInGame
		}
		h.BroadcastPlayerList()
	}
}

// Validates the received message based on its type
func ValidateMessage(msg *Message) error {
	switch msg.Type {

	// Chat / system messages — content must be a JSON string
	case BroadcastMessage, SystemMessage, PrivateMessage,
		PlayerListMessage, PlayerReadyToggle:
		var content string
		if err := json.Unmarshal(msg.Content, &content); err != nil {
			return fmt.Errorf("invalid message content format: %v", err)
		}
		if strings.TrimSpace(content) == "" {
			return fmt.Errorf("empty message content")
		}

	// Request / signal messages — content can be empty or minimal
	case RequestPlayerList, ResetReady, PlayerJoinedGame:
		// These are simple signal messages; no strict content validation needed
		// beyond being valid JSON (which is already guaranteed by unmarshal in ReadPump)

	// Server-originated messages (countdown / go) — should never arrive FROM a client,
	// but if they do just let them through validation so the hub can ignore them.
	case GameCountdown, GameGo:
		// no-op: these are server-generated

	// Game messages — content is a JSON object (or JSON-encoded string of an object)
	// We just verify it's non-empty valid JSON
	case PlayerProgress, GameFinished, GameStart:
		if len(msg.Content) == 0 {
			return fmt.Errorf("empty game message content")
		}
		// Make sure the content is valid JSON (object, string, whatever)
		if !json.Valid(msg.Content) {
			return fmt.Errorf("invalid JSON in game message content")
		}

	default:
		return fmt.Errorf("invalid message type: %s", msg.Type)
	}
	return nil
}

// This is the function which sends the system messages i.e
// if new user has joined or an existing user disconnected
func SendSystemMessages(systemMessageType string, client *Clients, hub *Hub) {
	var m Message
	m.Sender = "System"
	m.TimeStamp = time.Now()

	var content string
	switch systemMessageType {
	case UserJoinedSysMessage:
		content = fmt.Sprintf("%s joined the room", client.name)
	case UserLeftSysMessage:
		content = fmt.Sprintf("%s left the room", client.name)
	case NewHubCreated:
		content = "New Hub Created"
	}

	data, _ := json.Marshal(content)
	m.Content = data

	hub.broadcast <- m
}
