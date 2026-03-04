import {
  createContext,
  useContext,
  useEffect,
  useRef,
  useState,
  useCallback,
  useMemo,
} from "react";
import { useParams } from "react-router-dom";

const RoomSocketContext = createContext(null);

export function useRoomSocket() {
  const ctx = useContext(RoomSocketContext);
  if (!ctx) {
    throw new Error("useRoomSocket must be used within a RoomSocketProvider");
  }
  return ctx;
}

/**
 * Returns the room socket context or null if not inside a RoomSocketProvider.
 * Safe to call unconditionally — use this in components that render in both
 * singleplayer (no provider) and multiplayer (has provider) modes.
 */
export function useOptionalRoomSocket() {
  return useContext(RoomSocketContext);
}

/**
 * Provides a single WebSocket connection for the entire room.
 * Wraps both Lobby and Game routes so the connection persists
 * across navigation between them.
 */
export function RoomSocketProvider({ children }) {
  const { roomId } = useParams();
  const wsRef = useRef(null);
  const listenersRef = useRef(new Set());
  const [connectionStatus, setConnectionStatus] = useState("connecting");

  // Subscribe to incoming messages. Returns an unsubscribe function.
  const subscribe = useCallback((callback) => {
    listenersRef.current.add(callback);
    return () => {
      listenersRef.current.delete(callback);
    };
  }, []);

  // Send a JSON payload over the socket
  const send = useCallback((payload) => {
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      wsRef.current.send(JSON.stringify(payload));
    }
  }, []);

  // Establish the WebSocket connection once per roomId
  useEffect(() => {
    if (!roomId) return;

    // If there's already an open connection for this room, don't re-create
    if (
      wsRef.current &&
      (wsRef.current.readyState === WebSocket.OPEN ||
        wsRef.current.readyState === WebSocket.CONNECTING)
    ) {
      return;
    }

    setConnectionStatus("connecting");

    const socket = new WebSocket(
      `ws://localhost:8080/ws?action=join&room_id=${roomId}`,
    );
    wsRef.current = socket;

    socket.onopen = () => {
      console.log("[RoomSocket] Connected to room:", roomId);
      setConnectionStatus("connected");
    };

    socket.onmessage = (event) => {
      let data;
      try {
        data = JSON.parse(event.data);
      } catch {
        return;
      }
      // Fan out to all subscribers
      for (const listener of listenersRef.current) {
        try {
          listener(data);
        } catch (err) {
          console.error("[RoomSocket] Listener error:", err);
        }
      }
    };

    socket.onerror = (err) => {
      console.error("[RoomSocket] Error:", err);
      setConnectionStatus("error");
    };

    socket.onclose = (e) => {
      console.log("[RoomSocket] Closed:", e.code, e.reason);
      setConnectionStatus("disconnected");
    };

    return () => {
      console.log("[RoomSocket] Cleanup: closing WebSocket for room", roomId);
      if (
        socket.readyState === WebSocket.OPEN ||
        socket.readyState === WebSocket.CONNECTING
      ) {
        socket.close(1000, "Room layout unmount");
      }
      wsRef.current = null;
    };
  }, [roomId]);

  const value = useMemo(
    () => ({
      roomId,
      send,
      subscribe,
      connectionStatus,
      isConnected: connectionStatus === "connected",
    }),
    [roomId, send, subscribe, connectionStatus],
  );

  return (
    <RoomSocketContext.Provider value={value}>
      {children}
    </RoomSocketContext.Provider>
  );
}
