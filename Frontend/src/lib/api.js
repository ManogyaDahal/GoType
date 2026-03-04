import { API_URL } from "./config";

export async function fetchUser() {
  try {
    const res = await fetch(`${API_URL}/api/whoamI`, {
      credentials: "include", // send session cookie
    });

    if (!res.ok) return null;

    const data = await res.json();
    return data; // should be { name: "..." }
  } catch (err) {
    console.error("Error fetching user:", err);
    return null;
  }
}
