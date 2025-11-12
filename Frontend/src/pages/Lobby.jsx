import { useEffect, useState, useRef } from "react";
import { useParams, useNavigate } from "react-router-dom";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";

export default function Lobby() {
  const { roomId } = useParams();
  const [ws, setWs] = useState(null);
  const [players, setPlayers] = useState([]);
  const [messages, setMessages] = useState([]);
  const [messageInput, setMessageInput] = useState("");
  const [isReady, setIsReady] = useState(false);
  const [connectionStatus, setConnectionStatus] = useState("Connecting..."); // ADDED: Show connection status
  const chatEndRef = useRef(null);
  const navigate = useNavigate();

  // ✅ Scroll chat to bottom when new message appears
  useEffect(() => {
    chatEndRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [messages]);

  // ✅ Connect to WebSocket on mount
  useEffect(() => {
    const socket = new WebSocket(
      "ws://localhost:8080/ws?action=join&room_id=" + roomId
    );
    setWs(socket);

    // CHANGED: Properly handle WebSocket connection
    socket.onopen = () => {
      console.log("✅ WS Connected");
      setConnectionStatus("Connected"); // ADDED: Update status
    };

    // CHANGED: Parse and handle incoming messages properly
    socket.onmessage = (event) => {
      console.log("📩 WS Message:", event.data);
      
      try {
        const data = JSON.parse(event.data);
        console.log("Parsed message:", data);

        // CHANGED: Handle different message types
        if (data.type === "player_list") {
          // CHANGED: Parse the player list content
          const playerList = JSON.parse(data.content);
          console.log("Player list updated:", playerList);
          setPlayers(playerList);
        } else if (data.type === "broadcast") {
          // CHANGED: Parse broadcast content and add to messages
          const content = JSON.parse(data.content);
          setMessages((prev) => [
            ...prev,
            {
              sender: data.sender,
              content: content,
              timestamp: data.timestamp,
            },
          ]);
        } else if (data.type === "string") {
          // CHANGED: Handle system messages (note: your backend has "string" instead of "system")
          const content = JSON.parse(data.content);
          setMessages((prev) => [
            ...prev,
            {
              sender: "System",
              content: content,
              timestamp: data.timestamp,
            },
          ]);
        }
      } catch (err) {
        console.error("Error parsing message:", err);
      }
    };

    socket.onerror = (err) => {
      console.error("⚠️ WS Error:", err);
      setConnectionStatus("Error"); // ADDED: Update status
    };

    socket.onclose = (e) => {
      console.log("❌ WS Closed:", e.code, e.reason);
      setConnectionStatus("Disconnected"); // ADDED: Update status
      // ADDED: Optional - try to reconnect or show error to user
      if (e.code !== 1000) {
        alert("Connection lost. Please try rejoining the room.");
      }
    };

    // CHANGED: Cleanup function
    return () => {
      if (socket.readyState === WebSocket.OPEN) {
        socket.close();
      }
    };
  }, [roomId]);

  // ✅ Send chat message
  const sendMessage = () => {
    if (!messageInput.trim() || !ws) return;
    
    // CHANGED: Check if WebSocket is open before sending
    if (ws.readyState !== WebSocket.OPEN) {
      console.error("WebSocket is not open. ReadyState:", ws.readyState);
      alert("Connection not ready. Please wait or refresh.");
      return;
    }

    const msg = {
      type: "broadcast",
      room_id: roomId,
      content: JSON.stringify(messageInput), // CHANGED: Stringify the content
    };
    
    console.log("Sending message:", msg);
    ws.send(JSON.stringify(msg));
    setMessageInput("");
  };

  // CHANGED: Handle Enter key press in chat input
  const handleKeyPress = (e) => {
    if (e.key === "Enter") {
      sendMessage();
    }
  };

  // ✅ Toggle Ready state
  const toggleReady = () => {
    const newReadyState = !isReady;
    setIsReady(newReadyState);

    if (ws && ws.readyState === WebSocket.OPEN) {
      // CHANGED: Use the correct message type for ready toggle
      const msg = {
        type: "ready_toggle", // CHANGED: Match your backend constant PlayerReadyToggle
        room_id: roomId,
        content: JSON.stringify(newReadyState ? "ready" : "not ready"), // CHANGED: Stringify content
      };
      console.log("Toggling ready state:", msg);
      ws.send(JSON.stringify(msg));
    }
  };

  // ✅ Start Game (for now, just navigate)
  const startGame = () => {
    navigate(`/game/${roomId}`);
  };

  return (
    <div className="flex flex-col h-screen bg-gray-50">
      <header className="p-4 flex justify-between items-center bg-white shadow">
        <div>
          <h1 className="text-2xl font-bold">Room: {roomId}</h1>
          {/* ADDED: Show connection status */}
          <span className={`text-sm ${connectionStatus === 'Connected' ? 'text-green-600' : 'text-red-600'}`}>
            {connectionStatus}
          </span>
        </div>
        <Button onClick={() => navigate("/")}>Leave Room</Button>
      </header>

      <main className="flex flex-1 gap-6 p-6">
        {/* Player List */}
        <aside className="w-1/4 bg-white rounded-xl shadow p-4">
          <h2 className="text-lg font-semibold mb-3">Players</h2>
          <ul className="space-y-2">
            {players.length > 0 ? (
              players.map((p, i) => (
                <li
                  key={i}
                  className="px-3 py-2 bg-gray-100 rounded-md text-gray-700 flex justify-between"
                >
                  <span>{p.name}</span>
                  {p.ready && (
                    <span className="text-green-500 font-medium">Ready</span>
                  )}
                </li>
              ))
            ) : (
              <li className="text-gray-500">Waiting for players...</li>
            )}
          </ul>

          <Button
            onClick={toggleReady}
            variant={isReady ? "secondary" : "default"}
            className="mt-4 w-full"
            disabled={connectionStatus !== "Connected"} // ADDED: Disable if not connected
          >
            {isReady ? "Unready" : "Ready"}
          </Button>
        </aside>

        {/* Chat Section */}
        <section className="flex flex-col flex-1 bg-white rounded-xl shadow p-4">
          <div className="flex-1 overflow-y-auto space-y-2">
            {messages.map((m, i) => (
              <div
                key={i}
                className={`p-2 rounded-md ${
                  m.sender === "System"
                    ? "text-gray-500 text-center"
                    : "bg-gray-100"
                }`}
              >
                {m.sender !== "System" && (
                  <span className="font-semibold mr-2">{m.sender}:</span>
                )}
                {m.content}
              </div>
            ))}
            <div ref={chatEndRef} />
          </div>

          <div className="flex mt-4">
            <Input
              value={messageInput}
              onChange={(e) => setMessageInput(e.target.value)}
              onKeyPress={handleKeyPress} // ADDED: Handle Enter key
              placeholder="Type a message..."
              className="flex-1"
              disabled={connectionStatus !== "Connected"} // ADDED: Disable if not connected
            />
            <Button 
              onClick={sendMessage} 
              className="ml-2"
              disabled={connectionStatus !== "Connected"} // ADDED: Disable if not connected
            >
              Send
            </Button>
          </div>
        </section>
      </main>

      {/* Footer */}
      <footer className="p-4 bg-white shadow flex justify-center">
        <Button
          disabled={!isReady || players.some((p) => !p.ready) || connectionStatus !== "Connected"}
          onClick={startGame}
        >
          Start Game
        </Button>
      </footer>
    </div>
  );
}
