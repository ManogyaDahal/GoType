import { useEffect, useState } from "react";
import { Button } from "@/components/ui/button";
import { fetchUser } from "../lib/api";
import { useNavigate } from "react-router-dom";
import { API_URL } from "@/lib/config";

export default function Home() {
  const [user, setUser] = useState(null);
  const [showLoginPrompt, setShowLoginPrompt] = useState(false);
  const navigate = useNavigate();

  useEffect(() => {
    fetchUser().then(setUser);
  }, []);

  const handleLogin = () => {
    window.location.href = `${API_URL}/login`;
  };

  const handleLogout = () => {
    window.location.href = `${API_URL}/logout`;
  };

  const handleMultiplayerClick = () => {
    if (user) {
      navigate("/multiplayer");
    } else {
      setShowLoginPrompt(true);
    }
  };

  return (
    <div className="flex flex-col items-center justify-center h-screen gap-6 relative">
      <h1 className="text-4xl font-bold">Welcome to GoType!</h1>

      {user ? (
        <>
          <p className="text-xl">Hello, {user.name}</p>
          <Button onClick={handleLogout}>Logout</Button>
        </>
      ) : (
        <>
          <p className="text-xl">You're not Currently logged in</p>
          <Button onClick={handleLogin}>Sign in with Google</Button>
        </>
      )}

      <div className="flex gap-4 mt-6">
        <Button onClick={() => navigate("/singleplayer")}>Single Player</Button>
        <Button onClick={handleMultiplayerClick}>Multiplayer</Button>
      </div>

      {/* Login prompt modal */}
      {showLoginPrompt && (
        <div
          className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm"
          onClick={() => setShowLoginPrompt(false)}
        >
          <div
            className="bg-card border rounded-xl shadow-lg p-8 max-w-sm w-full mx-4 flex flex-col items-center gap-5"
            onClick={(e) => e.stopPropagation()}
          >
            <div className="text-center">
              <h2 className="text-2xl font-bold mb-2">Login Required</h2>
              <p className="text-muted-foreground">
                You need to be logged in to play multiplayer. Sign in to race
                against other players!
              </p>
            </div>

            <div className="flex flex-col gap-3 w-full">
              <Button onClick={handleLogin} className="w-full">
                Sign in with Google
              </Button>
              <Button
                variant="outline"
                onClick={() => setShowLoginPrompt(false)}
                className="w-full"
              >
                Cancel
              </Button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
