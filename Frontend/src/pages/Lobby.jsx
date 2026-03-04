import { useEffect, useState, useRef, useCallback } from "react";
import { useNavigate } from "react-router-dom";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { useRoomSocket } from "../context/RoomSocketContext";
import { generateTextSeeded } from "../lib/gameLogic";

export default function Lobby() {
  const { roomId, send, subscribe, connectionStatus, isConnected } =
    useRoomSocket();
  const navigate = useNavigate();

  const chatEndRef = useRef(null);

  const [players, setPlayers] = useState([]);
  const [messages, setMessages] = useState([]);
  const [messageInput, setMessageInput] = useState("");
  const [isReady, setIsReady] = useState(false);
  const [readyInFlight, setReadyInFlight] = useState(false);
  const [countdown, setCountdown] = useState(null);
  const countdownRef = useRef(null);
  const hasSentInit = useRef(false);

  // On mount (or reconnect): reset our ready state on the server and request
  // a fresh player list so we don't show "Waiting for players..." when
  // players are already in the room (e.g. returning from a game).
  useEffect(() => {
    if (!isConnected || hasSentInit.current) return;
    hasSentInit.current = true;

    // Reset our own ready state on the server
    send({
      type: "reset_ready",
      room_id: roomId,
      content: "reset",
    });

    // Ask the server to broadcast the current player list
    send({
      type: "request_player_list",
      room_id: roomId,
      content: "request",
    });

    // Make sure local state matches
    setIsReady(false);
  }, [isConnected, send, roomId]);

  // Reset the init flag when the component unmounts so it fires again
  // if we navigate back to the lobby later.
  useEffect(() => {
    return () => {
      hasSentInit.current = false;
    };
  }, []);

  // Subscribe to incoming WebSocket messages
  useEffect(() => {
    const unsubscribe = subscribe((data) => {
      if (data.type === "player_list") {
        try {
          const playerList = JSON.parse(data.content);
          setPlayers(playerList);

          // When we receive the authoritative player list, sync our local
          // ready state with what the server says about us. This prevents
          // the UI from getting out of sync after rapid toggling.
          const me = playerList.find((p) => p.name === data._myName || false);
          // We can't reliably know our own name from the list alone
          // (the server doesn't tag "you"), so we just unlock the button
          // whenever a fresh list arrives — the server is the source of truth.
          setReadyInFlight(false);
        } catch (e) {
          console.error("Invalid player_list JSON:", data.content);
        }
      } else if (data.type === "broadcast") {
        setMessages((prev) => [
          ...prev,
          {
            sender: data.sender,
            content: data.content,
            timestamp: data.timestamp,
          },
        ]);
      } else if (data.type === "string") {
        try {
          const content = JSON.parse(data.content);
          setMessages((prev) => [
            ...prev,
            {
              sender: "System",
              content,
              timestamp: data.timestamp,
            },
          ]);
        } catch (e) {
          console.error("Failed to parse system message");
        }
      }
    });

    return unsubscribe;
  }, [subscribe]);

  // Auto-scroll chat
  useEffect(() => {
    chatEndRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [messages]);

  const sendMessage = useCallback(() => {
    if (!isConnected) {
      console.warn("Cannot send — WebSocket not open yet!");
      return;
    }
    const content = messageInput.trim();
    if (!content) {
      console.warn("⚠️ Cannot send message - empty content");
      return;
    }

    send({
      type: "broadcast",
      content: content,
      room_id: roomId,
    });

    // Optimistic UI update
    setMessages((prev) => [
      ...prev,
      {
        sender: "You",
        content: content,
        timestamp: new Date().toISOString(),
      },
    ]);

    setMessageInput("");
  }, [isConnected, messageInput, send, roomId]);

  const toggleReady = useCallback(() => {
    if (!isConnected || readyInFlight) {
      return;
    }

    const newState = !isReady;

    // Lock the button until we get a player_list response back
    setReadyInFlight(true);
    setIsReady(newState);

    // Send the DESIRED state — the server will SET it, not toggle
    send({
      type: "ready_toggle",
      room_id: roomId,
      content: newState ? "ready" : "not ready",
    });

    // Safety timeout: unlock the button after 2s even if we never get
    // a player_list back (e.g. network issues)
    setTimeout(() => setReadyInFlight(false), 2000);
  }, [isConnected, readyInFlight, isReady, send, roomId]);

  const handleLeaveRoom = useCallback(() => {
    console.log("👋 Leaving room");
    // Navigation away from /room/:roomId/* will unmount RoomLayout,
    // which triggers the provider cleanup and closes the WebSocket.
    navigate("/");
  }, [navigate]);

  const allReady =
    players.length > 0 && players.every((p) => p.status === "ready");
  const someInGame = players.some((p) => p.status === "in_game");

  // Auto-start: countdown when all players are ready
  useEffect(() => {
    if (allReady && isConnected) {
      setCountdown(3);
      countdownRef.current = setInterval(() => {
        setCountdown((prev) => {
          if (prev === null) return null;
          if (prev <= 1) {
            clearInterval(countdownRef.current);
            countdownRef.current = null;
            const gameText = generateTextSeeded(roomId, 25);
            // Navigate within the same /room/:roomId layout — WebSocket stays alive
            navigate(`/room/${roomId}/game`, { state: { text: gameText } });
            return 0;
          }
          return prev - 1;
        });
      }, 1000);
    } else {
      // Someone un-readied or disconnected — cancel countdown
      if (countdownRef.current) {
        clearInterval(countdownRef.current);
        countdownRef.current = null;
      }
      setCountdown(null);
    }

    return () => {
      if (countdownRef.current) {
        clearInterval(countdownRef.current);
        countdownRef.current = null;
      }
    };
  }, [allReady, isConnected, navigate, roomId]);

  return (
    <div className="flex flex-col h-screen bg-gray-50">
      <header className="p-4 flex justify-between items-center bg-white shadow">
        <div>
          <h1 className="text-2xl font-bold">Room: {roomId}</h1>
          <span
            className={`text-sm ${isConnected ? "text-green-600" : "text-red-600"}`}
          >
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
                  {p.status === "ready" && (
                    <span className="text-green-500 font-medium">Ready</span>
                  )}
                  {p.status === "in_game" && (
                    <span className="text-yellow-500 font-medium">In Game</span>
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
            disabled={!isConnected || readyInFlight}
          >
            {readyInFlight ? "..." : isReady ? "Unready" : "Ready"}
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
      <footer className="p-4 bg-white shadow flex justify-center items-center gap-3">
        {countdown !== null ? (
          <span className="text-lg font-bold text-green-600 animate-pulse">
            Starting in {countdown}...
          </span>
        ) : allReady ? (
          <span className="text-sm text-green-600">All players ready!</span>
        ) : someInGame ? (
          <span className="text-sm text-yellow-600">
            Waiting for players to finish their game...
          </span>
        ) : (
          <span className="text-sm text-gray-500">
            Waiting for all players to ready up...
          </span>
        )}
      </footer>
    </div>
  );
}
