import { Outlet } from "react-router-dom";
import { RoomSocketProvider } from "../context/RoomSocketContext";

/**
 * Layout wrapper for all room-scoped routes (/lobby/:roomId, /game/:roomId).
 * Provides a single shared WebSocket connection that persists across
 * Lobby ↔ Game navigation within the same room.
 */
export default function RoomLayout() {
  return (
    <RoomSocketProvider>
      <Outlet />
    </RoomSocketProvider>
  );
}
