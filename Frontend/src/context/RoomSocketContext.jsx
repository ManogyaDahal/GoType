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
import { WS_URL } from "@/lib/config";
import { fetchUser } from "@/lib/api";

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
 *
 * Auth flow:
 *  1. Fetches /api/whoamI through the Vercel proxy (session cookie is valid there).
 *  2. Extracts the short-lived HMAC ws_token from the response.
 *  3. Passes the token as ?token= in the WebSocket URL, which connects
 *     directly to Render — no session cookie needed on that domain.
 */
export function RoomSocketProvider({ children }) {
  const { roomId } = useParams();
  const wsRef = useRef(null);
  const listenersRef = useRef(new Set());
  const [connectionStatus, setConnectionStatus] = useState("connecting");
  const [wsToken, setWsToken] = useState(null);

  // Step 1: fetch the ws_token from whoamI on mount.
  // This call goes through the Vercel proxy where the session cookie is valid.
  useEffect(() => {
    fetchUser().then((user) => {
      if (user?.ws_token) {
        setWsToken(user.ws_token);
      } else {
        // Not logged in or token missing — mark as error so the UI can react.
        setConnectionStatus("error");
      }
    });
  }, []);

  // Subscribe to incoming messages. Returns an unsubscribe function.
  const subscribe = useCallback((callback) => {
    listenersRef.current.add(callback);
    return () => {
      listenersRef.current.delete(callback);
    };
  }, []);

  // Send a JSON payload over the socket.
  const send = useCallback((payload) => {
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      wsRef.current.send(JSON.stringify(payload));
    }
  }, []);

  // Step 2: open the WebSocket once both roomId and wsToken are available.
  useEffect(() => {
    if (!roomId || !wsToken) return;

    // If there's already an open connection for this room, don't re-create.
    if (
      wsRef.current &&
      (wsRef.current.readyState === WebSocket.OPEN ||
        wsRef.current.readyState === WebSocket.CONNECTING)
    ) {
      return;
    }

    setConnectionStatus("connecting");

    // Connect directly to Render with the HMAC token — Vercel cannot proxy
    // WebSocket connections, so WS_URL must point to the Render backend.
    const socket = new WebSocket(
      `${WS_URL}/ws?action=join&room_id=${roomId}&token=${encodeURIComponent(wsToken)}`,
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
      // Fan out to all subscribers.
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
  }, [roomId, wsToken]);

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
