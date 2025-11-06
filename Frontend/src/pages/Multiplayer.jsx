import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";

export default function Multiplayer() {
  const [roomCode, setRoomCode] = useState("");
  const navigate = useNavigate();

  // When user clicks "Join Room"
  const handleJoinRoom = () => {
    if (!roomCode.trim()) {
      alert("Please enter a valid room code.");
      return;
    }
    navigate(`/lobby/${roomCode}`);
  };

  // When user clicks "Create Room"
  const handleCreateRoom = async () => {
    try {
      // Backend endpoint that creates new hub
      const res = await fetch("http://localhost:8080/api/create-room", {
        credentials: "include",
      });
      const data = await res.json();
      navigate(`/lobby/${data.roomId}`);
    } catch (err) {
      console.error("Error creating room:", err);
      alert("Failed to create room.");
    }
  };

  return (
    <div className="flex flex-col items-center justify-center h-screen gap-6">
      <h1 className="text-4xl font-bold">Multiplayer Mode</h1>

      <div className="flex flex-col items-center gap-3">
        <Input
          value={roomCode}
          onChange={(e) => setRoomCode(e.target.value)}
          placeholder="Enter Room Code"
          className="w-64 text-center"
        />
        <Button onClick={handleJoinRoom} className="w-64">
          Join Room
        </Button>
      </div>

      <p className="text-gray-500 my-2">— OR —</p>

      <Button onClick={handleCreateRoom} className="w-64">
        Create New Room
      </Button>
    </div>
  );
}
