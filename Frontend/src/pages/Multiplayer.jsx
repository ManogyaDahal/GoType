import { useState, useEffect } from "react";
import { useNavigate } from "react-router-dom";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { fetchUser } from "../lib/api";

export default function Multiplayer() {
  const [roomCode, setRoomCode] = useState("");
  const [user, setUser] = useState(null);
  const [loading, setLoading] = useState(true);
  const navigate = useNavigate();

  useEffect(() => {
    fetchUser()
      .then((u) => {
        if (!u) {
          navigate("/", { replace: true });
        } else {
          setUser(u);
          setLoading(false);
        }
      })
      .catch(() => {
        navigate("/", { replace: true });
      });
  }, [navigate]);

  const handleJoinRoom = () => {
    if (!roomCode.trim()) {
      alert("Please enter a valid room code.");
      return;
    }
    navigate(`/room/${roomCode}/lobby`);
  };

  const handleCreateRoom = async () => {
    try {
      const res = await fetch("http://localhost:8080/api/create-room", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        credentials: "include",
      });

      if (!res.ok) {
        if (res.status === 401) {
          alert("Session expired. Please login again.");
          navigate("/");
          return;
        }
        throw new Error("Failed to create room");
      }

      const data = await res.json();
      console.log("Room created:", data);
      navigate(`/room/${data.room_id}/lobby`);
    } catch (err) {
      console.error("Error creating room:", err);
      alert("Failed to create room.");
    }
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center h-screen">
        <p className="text-xl text-muted-foreground">Loading...</p>
      </div>
    );
  }

  return (
    <div className="flex flex-col items-center justify-center h-screen gap-6">
      <h1 className="text-4xl font-bold">Multiplayer Mode</h1>
      <p className="text-muted-foreground">Playing as {user?.name}</p>

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
