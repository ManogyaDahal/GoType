import { BrowserRouter as Router, Routes, Route } from "react-router-dom";
import Home from "./pages/Home";
// import Multiplayer from "./pages/Multiplayer";
// import Lobby from "./pages/Lobby";
// import Singleplayer from "./pages/Singleplayer";
// import GameRoom from "./pages/GameRoom";

export default function App() {
  return (
    <Router>
      <Routes>
        <Route path="/" element={<Home />} />
      </Routes>
    </Router>
  );
}

        // <Route path="/multiplayer" element={<Multiplayer />} />
        // <Route path="/lobby" element={<Lobby />} />
        // <Route path="/singleplayer" element={<Singleplayer />} />
        // <Route path="/game" element={<GameRoom />} />
