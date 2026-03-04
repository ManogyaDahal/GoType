import { BrowserRouter as Router, Routes, Route } from "react-router-dom";
import Home from "./pages/Home";
import Multiplayer from "./pages/Multiplayer";
import Lobby from "./pages/Lobby";
import Game from "./pages/Game";
import RoomLayout from "./pages/RoomLayout";

export default function App() {
  return (
    <Router>
      <Routes>
        <Route path="/" element={<Home />} />
        <Route path="/singleplayer" element={<Game mode="single" />} />
        <Route path="/multiplayer" element={<Multiplayer />} />

        {/* Room-scoped routes share a single WebSocket via RoomLayout */}
        <Route path="/room/:roomId" element={<RoomLayout />}>
          <Route path="lobby" element={<Lobby />} />
          <Route path="game" element={<Game mode="multi" />} />
        </Route>
      </Routes>
    </Router>
  );
}
