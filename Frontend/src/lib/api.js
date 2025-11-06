export async function fetchUser() {
  try {
    const res = await fetch("http://localhost:8080/api/whoamI", {
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
