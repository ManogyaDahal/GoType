import { useEffect, useState } from "react";
import { Button } from "@/components/ui/button";
import { fetchUser } from "../lib/api";
import { useNavigate } from "react-router-dom";

export default function Home() {
  const [user, setUser] = useState(null);
  const navigate = useNavigate();

  useEffect(() => {
    fetchUser().then(setUser);
  }, []);

  const handleLogin = () => {
    window.location.href = "http://localhost:8080/login"; // Go OAuth
  };

  const handleLogout = () => {
    window.location.href = "http://localhost:8080/logout";
  };

  return (
    <div className="flex flex-col items-center justify-center h-screen gap-6">
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
        <Button onClick={() => navigate("/multiplayer")}>Multiplayer</Button>
      </div>
    </div>
  );
}
