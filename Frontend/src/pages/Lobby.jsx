import { useEffect, useState, useRef } from "react";
import { useParams, useNavigate } from "react-router-dom";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";

export default function Lobby() {
  const { roomId } = useParams();
  const navigate = useNavigate();
  
  const wsRef = useRef(null);
  const chatEndRef = useRef(null);
  
  const [players, setPlayers] = useState([]);
  const [messages, setMessages] = useState([]);
  const [messageInput, setMessageInput] = useState("");
  const [isReady, setIsReady] = useState(false);
  const [connectionStatus, setConnectionStatus] = useState("connecting");

  // Connect to WebSocket once on mount
  useEffect(() => {
    const socket = new WebSocket(
      `ws://localhost:8080/ws?action=join&room_id=${roomId}`
    );
    wsRef.current = socket;

    socket.onopen = () => {
      console.log("✅ Connected to room:", roomId);
      setConnectionStatus("connected");
    };

    socket.onmessage = (event) => {
      console.log("📩 Raw message:", event.data);
      
      try {
        const data = JSON.parse(event.data);
        console.log("📦 Parsed message:", data);
        
        if (data.type === "player_list") {
          const playerList = JSON.parse(data.content);
          console.log("👥 Player list:", playerList);
          setPlayers(playerList);
        } else if (data.type === "broadcast") {
          const content = JSON.parse(data.content);
          console.log("💬 Chat message:", content, "from:", data.sender);
          setMessages((prev) => [...prev, {
            sender: data.sender,
            content: content,
            timestamp: data.timestamp,
          }]);
        } else if (data.type === "string") {
          const content = JSON.parse(data.content);
          console.log("📢 System message:", content);
          setMessages((prev) => [...prev, {
            sender: "System",
            content: content,
            timestamp: data.timestamp,
          }]);
        }
      } catch (err) {
        console.error("❌ Parse error:", err, "Raw data:", event.data);
      }
    };

    socket.onerror = (err) => {
      console.error("⚠️ WebSocket error:", err);
      setConnectionStatus("error");
    };

    socket.onclose = (e) => {
      console.log("🔌 WebSocket closed:", e.code, e.reason);
      setConnectionStatus("disconnected");
      if (e.code !== 1000) {
        alert("Connection lost. Please rejoin.");
      }
    };

    return () => {
      console.log("🧹 Cleanup: Closing WebSocket");
      if (socket.readyState === WebSocket.OPEN) {
        socket.close(1000, "Component unmount");
      }
    };
  }, [roomId]);

  // Auto-scroll chat
  useEffect(() => {
    chatEndRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [messages]);

  const sendMessage = () => {
		if (wsRef.current?.readyState !== WebSocket.OPEN) {
  		console.warn("Cannot send — WebSocket not open yet!", wsRef.current?.readyState);
  		return;
		}
    const content = messageInput.trim();
    if (!content || !wsRef.current || wsRef.current.readyState !== WebSocket.OPEN) {
      console.warn("⚠️ Cannot send message - invalid state");
      return;
    }

    const payload = {
      type: "broadcast",
      content: content,
      room_id: roomId,
    };

    console.log("📤 Sending message:", payload);
    wsRef.current.send(JSON.stringify(payload));
    
    // Optimistic UI update
    setMessages((prev) => [...prev, {
      sender: "You",
      content: content,
      timestamp: new Date().toISOString(),
    }]);
    
    setMessageInput("");
  };

  const toggleReady = () => {
    if (!wsRef.current || wsRef.current.readyState !== WebSocket.OPEN) {
      console.warn("⚠️ Cannot toggle ready - not connected");
      return;
    }
    
    const newState = !isReady;
    setIsReady(newState);

    const payload = {
      type: "ready_toggle",
      room_id: roomId,
      content: newState ? "ready" : "not ready",
    };

    console.log("🎮 Toggling ready:", payload);
    wsRef.current.send(JSON.stringify(payload));
  };

  const handleLeaveRoom = () => {
    console.log("👋 Leaving room");
    if (wsRef.current && wsRef.current.readyState === WebSocket.OPEN) {
      wsRef.current.close(1000, "User left");
    }
    navigate("/");
  };

  const startGame = () => {
    console.log("🎮 Starting game");
    navigate(`/game/${roomId}`);
  };

  const isConnected = connectionStatus === "connected";
  const allReady = players.length > 0 && players.every(p => p.ready);

  return (
    <div className="flex flex-col h-screen bg-gray-50">
      <header className="p-4 flex justify-between items-center bg-white shadow">
        <div>
          <h1 className="text-2xl font-bold">Room: {roomId}</h1>
          <span className={`text-sm ${isConnected ? 'text-green-600' : 'text-red-600'}`}>
            {connectionStatus}
          </span>
        </div>
        <Button onClick={handleLeaveRoom}>Leave Room</Button>
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
            disabled={!isConnected}
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
                className={`p-2 rounded-md max-w-xs ${
                  m.sender === "System"
                    ? "text-gray-500 text-center mx-auto"
                    : m.sender === "You"
                    ? "bg-blue-100 ml-auto"
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
              onKeyPress={(e) => e.key === "Enter" && sendMessage()}
              placeholder="Type a message..."
              className="flex-1"
              disabled={!isConnected}
            />
            <Button 
              onClick={sendMessage} 
              className="ml-2"
              disabled={!isConnected}
            >
              Send
            </Button>
          </div>
        </section>
      </main>

      {/* Footer */}
      <footer className="p-4 bg-white shadow flex justify-center">
        <Button
          disabled={!isReady || !allReady || !isConnected}
          onClick={startGame}
        >
          Start Game
        </Button>
      </footer>
    </div>
  );
}
