const API_URL = import.meta.env.VITE_API_URL;

// WebSocket must connect directly to Render — Vercel cannot proxy WebSocket
// connections (serverless platform, no persistent connections).
// VITE_WS_URL should be set to wss://gotype-595o.onrender.com in production.
// Falls back to deriving from API_URL for local development.
const WS_URL = import.meta.env.VITE_WS_URL || API_URL.replace(/^http/, "ws");

export { API_URL, WS_URL };
