import { useState, useEffect, useRef, useCallback, Fragment } from "react";
import { useParams, useNavigate, useLocation } from "react-router-dom";
import { RotateCcw, Eye, EyeOff } from "lucide-react";
import {
  useGameLogic,
  generateText,
  generateTextSeeded,
} from "../lib/gameLogic";
import { useOptionalRoomSocket } from "../context/RoomSocketContext";

// Colors assigned to other players' ghost cursors
const GHOST_COLORS = [
  "#6366f1", // indigo
  "#ec4899", // pink
  "#10b981", // emerald
  "#f59e0b", // amber
  "#8b5cf6", // violet
  "#06b6d4", // cyan
];

// How often (ms) to push progress to the server
const PROGRESS_THROTTLE_MS = 250;

// ---------------------------------------------------------------------------
// Main Game component
// ---------------------------------------------------------------------------
export default function Game({ mode = "single" }) {
  const { roomId } = useParams();
  const navigate = useNavigate();
  const location = useLocation();

  // ---- state ----
  const [text, setText] = useState("");
  const [gameReady, setGameReady] = useState(false);
  const [isFocused, setIsFocused] = useState(false);
  const [showGhosts, setShowGhosts] = useState(true);

  // Multiplayer pre-game countdown (3/2/1 = counting, 0 = GO)
  const [preGameCountdown, setPreGameCountdown] = useState(
    mode === "multi" ? 3 : 0,
  );
  // Server-authoritative start time (unix ms) — shared across all players
  const [sharedStartTime, setSharedStartTime] = useState(null);

  // { [playerName]: { pos, wpm, color } }
  const [otherPlayers, setOtherPlayers] = useState({});
  // Ordered list of all finishers (self + others) in the order they finished.
  // [{ name, wpm, isMe }]
  const [raceResults, setRaceResults] = useState([]);

  const containerRef = useRef(null);
  const throttleTimer = useRef(null);
  const countdownIntervalRef = useRef(null);
  const joinRetryRef = useRef(null);
  const gameGoReceivedRef = useRef(false);
  const myNameRef = useRef("You");
  const colorIndexRef = useRef(0);

  // ---- shared room socket (only available in multiplayer) ----
  // Returns null when outside RoomSocketProvider (i.e. singleplayer mode).
  const roomSocket = useOptionalRoomSocket();

  // ---- helpers ----
  const sendToRoom = useCallback(
    (payload) => {
      if (mode === "multi" && roomSocket) {
        roomSocket.send(payload);
      }
    },
    [mode, roomSocket],
  );

  // ---- progress callback (throttled) ----
  const handleProgress = useCallback(
    (progress) => {
      if (mode !== "multi") return;
      if (throttleTimer.current) return;
      throttleTimer.current = setTimeout(() => {
        throttleTimer.current = null;
        sendToRoom({
          type: "player_progress",
          room_id: roomId,
          content: JSON.stringify({ pos: progress.pos, wpm: progress.wpm }),
        });
      }, PROGRESS_THROTTLE_MS);
    },
    [mode, roomId, sendToRoom],
  );

  // ---- finish callback ----
  const handleFinish = useCallback(
    (result) => {
      if (mode === "multi") {
        sendToRoom({
          type: "game_finished",
          room_id: roomId,
          content: JSON.stringify({ wpm: result.wpm }),
        });
        // Add ourselves to the results list — everyone already in the
        // list finished before us, so appending preserves finish order.
        setRaceResults((prev) => [
          ...prev,
          { name: myNameRef.current, wpm: result.wpm, isMe: true },
        ]);
      }
    },
    [mode, roomId, sendToRoom],
  );

  // ---- game hook ----
  const {
    charStates,
    cursorPos,
    currentWordIndex,
    currentInput,
    words,
    wpm,
    timeElapsed,
    isStarted,
    isFinished,
    handleKeyDown: rawHandleKeyDown,
    reset,
  } = useGameLogic({
    text,
    onProgress: handleProgress,
    onFinish: handleFinish,
    // In multiplayer, use the server's start time so all players share one clock
    forcedStartTime: mode === "multi" ? sharedStartTime : undefined,
  });

  // Block keyboard input during the pre-game countdown (multiplayer)
  const handleKeyDown = useCallback(
    (e) => {
      if (mode === "multi" && preGameCountdown !== 0) {
        // Countdown hasn't finished yet — swallow the event
        e.preventDefault();
        return;
      }
      rawHandleKeyDown(e);
    },
    [mode, preGameCountdown, rawHandleKeyDown],
  );

  // ---- singleplayer bootstrap ----
  useEffect(() => {
    if (mode === "single") {
      setText(generateText(25));
      setGameReady(true);
    }
  }, [mode]);

  // ---- multiplayer bootstrap (uses shared socket) ----
  useEffect(() => {
    if (mode !== "multi" || !roomId || !roomSocket) return;

    // Set the game text from navigation state or seeded fallback
    if (location.state?.text) {
      setText(location.state.text);
      setGameReady(true);
    } else {
      setText(generateTextSeeded(roomId, 25));
      setGameReady(true);
    }
    if (location.state?.myName) {
      myNameRef.current = location.state.myName;
    }

    // Tell the server we've arrived on the game page.
    // We retry every second until game_go is received, because the
    // initial send can be silently dropped (socket not quite ready,
    // effect re-run, etc.). PlayerJoinedGame is idempotent on the
    // server (keyed by player name), so retries are harmless.
    gameGoReceivedRef.current = false;

    const sendJoin = () => {
      console.log("[Game] Sending player_joined_game", {
        roomId,
        socketReady: roomSocket.isConnected,
      });
      roomSocket.send({
        type: "player_joined_game",
        room_id: roomId,
        content: "joined",
      });
    };

    sendJoin();
    joinRetryRef.current = setInterval(() => {
      if (gameGoReceivedRef.current) {
        clearInterval(joinRetryRef.current);
        joinRetryRef.current = null;
        return;
      }
      console.log("[Game] Retrying player_joined_game (no game_go yet)");
      sendJoin();
    }, 1000);

    // Subscribe to incoming game messages
    const unsubscribe = roomSocket.subscribe((data) => {
      console.log("[Game] WS message received:", data.type);
      switch (data.type) {
        case "game_go": {
          // Server sends a single game_go with start_time set ~3 s in the
          // future.  We run the 3→2→1 countdown locally so there's only ONE
          // critical message to deliver (no more dropped countdown ticks).
          try {
            const payload =
              typeof data.content === "string"
                ? JSON.parse(data.content)
                : data.content;
            const startTime = payload.start_time; // unix ms
            console.log("[Game] game_go received!", {
              startTime,
              now: Date.now(),
              delta: startTime - Date.now(),
            });

            // Stop retrying player_joined_game — server heard us
            gameGoReceivedRef.current = true;
            if (joinRetryRef.current) {
              clearInterval(joinRetryRef.current);
              joinRetryRef.current = null;
            }

            // Immediately show the first countdown number
            const remaining = Math.max(0, startTime - Date.now());
            setPreGameCountdown(Math.ceil(remaining / 1000));

            // Tick every 200 ms for a smooth update until start_time
            if (countdownIntervalRef.current) {
              clearInterval(countdownIntervalRef.current);
            }
            countdownIntervalRef.current = setInterval(() => {
              const left = startTime - Date.now();
              if (left <= 0) {
                console.log("[Game] Countdown finished, starting game");
                clearInterval(countdownIntervalRef.current);
                countdownIntervalRef.current = null;
                setPreGameCountdown(0);
                setSharedStartTime(startTime);
              } else {
                setPreGameCountdown(Math.ceil(left / 1000));
              }
            }, 200);
          } catch {
            /* ignore */
          }
          break;
        }
        case "game_start": {
          try {
            const gameText = JSON.parse(data.content);
            setText(gameText);
            setGameReady(true);
          } catch {
            if (typeof data.content === "string") {
              setText(data.content);
              setGameReady(true);
            }
          }
          break;
        }
        case "player_progress": {
          if (!data.sender || data.sender === myNameRef.current) break;
          try {
            const prog = JSON.parse(data.content);
            setOtherPlayers((prev) => ({
              ...prev,
              [data.sender]: {
                pos: prog.pos ?? 0,
                wpm: prog.wpm ?? 0,
                color:
                  prev[data.sender]?.color ||
                  GHOST_COLORS[Object.keys(prev).length % GHOST_COLORS.length],
              },
            }));
          } catch {
            /* ignore */
          }
          break;
        }
        case "game_finished": {
          if (!data.sender || data.sender === myNameRef.current) break;
          try {
            const result = JSON.parse(data.content);
            // Append to raceResults — order of arrival = finish order
            setRaceResults((prev) => [
              ...prev,
              { name: data.sender, wpm: result.wpm, isMe: false },
            ]);
            // Move the ghost cursor to the very end of the text
            setOtherPlayers((prev) => {
              if (!prev[data.sender]) return prev;
              return {
                ...prev,
                [data.sender]: {
                  ...prev[data.sender],
                  pos: text.length,
                  wpm: result.wpm,
                },
              };
            });
          } catch {
            /* ignore */
          }
          break;
        }
        default:
          break;
      }
    });

    return () => {
      clearTimeout(throttleTimer.current);
      if (joinRetryRef.current) {
        clearInterval(joinRetryRef.current);
        joinRetryRef.current = null;
      }
      if (countdownIntervalRef.current) {
        clearInterval(countdownIntervalRef.current);
        countdownIntervalRef.current = null;
      }
      // NOTE: Do NOT reset preGameCountdown or sharedStartTime here.
      // This cleanup runs both on unmount AND when effect deps change
      // (e.g. roomSocket ref changes when connectionStatus updates).
      // Resetting here would wipe countdown progress mid-game.
      // On actual unmount (navigating away), React destroys all state
      // and useState re-initializes to the correct defaults on remount.
      unsubscribe();
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [mode, roomId, roomSocket]);

  // ---- auto-focus when ready ----
  useEffect(() => {
    if (gameReady) {
      containerRef.current?.focus();
    }
  }, [gameReady]);

  // ---- retry / new game (singleplayer only) ----
  const handleRetry = useCallback(() => {
    setText(generateText(25));
    setRaceResults([]);
    setOtherPlayers({});
    setTimeout(() => containerRef.current?.focus(), 50);
  }, []);

  // ---- time display ----
  const totalSec = Math.floor(timeElapsed / 1000);
  const timeStr = `${totalSec}`;

  // ---------------------------------------------------------------------------
  // Build word-level rendering data
  // ---------------------------------------------------------------------------
  const wordRenderData = [];
  {
    let charOffset = 0;
    for (let w = 0; w < words.length; w++) {
      const word = words[w];
      const wordStart = charOffset;
      const chars = [];

      for (let c = 0; c < word.length; c++) {
        chars.push({
          char: word[c],
          globalIndex: charOffset,
          state: charStates[charOffset] ?? "untyped",
        });
        charOffset++;
      }

      let extraChars = [];
      if (w === currentWordIndex && currentInput.length > word.length) {
        extraChars = currentInput.slice(word.length).split("");
      }

      wordRenderData.push({
        wordIndex: w,
        wordStart,
        wordEnd: charOffset - 1,
        chars,
        extraChars,
      });

      if (w < words.length - 1) {
        charOffset++; // space
      }
    }
  }

  // ---- progress % ----
  const myPct = text ? Math.round((cursorPos / text.length) * 100) : 0;

  // ---------------------------------------------------------------------------
  // Waiting screen (multiplayer only — waiting for host)
  // ---------------------------------------------------------------------------
  if (!gameReady) {
    return (
      <div className="flex flex-col items-center justify-center gap-4 min-h-screen bg-background font-mono">
        <span className="text-muted-foreground text-lg">
          Waiting for host to start the game...
        </span>
        <button
          onClick={() => navigate(`/room/${roomId}/lobby`)}
          className="text-muted-foreground bg-transparent border border-muted-foreground px-5 py-2 rounded-md cursor-pointer text-sm hover:bg-muted transition-colors"
        >
          Back to Lobby
        </button>
      </div>
    );
  }

  // Are we in the pre-game countdown phase? (multiplayer only)
  const inCountdown = mode === "multi" && preGameCountdown > 0;

  // ---------------------------------------------------------------------------
  // Render
  // ---------------------------------------------------------------------------
  return (
    <div
      className="flex flex-col select-none min-h-screen bg-background font-mono"
      onClick={() => containerRef.current?.focus()}
    >
      <div className="flex-1 flex flex-col items-center justify-center px-8 w-full max-w-4xl mx-auto">
        {/* ---- Multiplayer countdown overlay ---- */}
        {inCountdown && (
          <div className="fixed inset-0 z-40 bg-background flex items-center justify-center font-mono">
            <div className="text-center">
              <div className="text-9xl font-bold text-foreground leading-none animate-pulse">
                {preGameCountdown}
              </div>
            </div>
          </div>
        )}

        {/* ---- Top stats row ---- */}
        <div
          className={`self-start mb-6 flex items-end gap-8 transition-opacity duration-300 ${
            isStarted && !isFinished ? "opacity-100" : "opacity-0"
          }`}
        >
          {/* Timer */}
          <div>
            <span className="text-4xl font-bold text-foreground">
              {timeStr}
            </span>
          </div>
          {/* My WPM */}
          <div className="flex flex-col">
            <span className="text-3xl font-bold text-foreground">{wpm}</span>
            <span className="text-xs text-muted-foreground">wpm</span>
          </div>
          {/* Other players' WPM (multiplayer only) */}
          {mode === "multi" &&
            Object.entries(otherPlayers).map(([name, player]) => (
              <div key={name} className="flex flex-col">
                <span
                  className="text-2xl font-bold"
                  style={{ color: player.color }}
                >
                  {player.wpm}
                </span>
                <span
                  className="text-xs truncate max-w-[80px]"
                  style={{ color: player.color, opacity: 0.7 }}
                >
                  {name}
                </span>
              </div>
            ))}
        </div>

        {/* ---- Typing area ---- */}
        <div
          ref={containerRef}
          tabIndex={0}
          onKeyDown={handleKeyDown}
          onFocus={() => setIsFocused(true)}
          onBlur={() => setIsFocused(false)}
          className="w-full outline-none relative cursor-text"
        >
          {/* Blur overlay when not focused */}
          {!isFocused && (
            <div className="absolute inset-0 z-10 flex items-center justify-center backdrop-blur-sm bg-background/50 rounded-lg cursor-pointer">
              <span className="text-muted-foreground text-sm">
                Click here or press any key to focus
              </span>
            </div>
          )}

          {/* Words */}
          <div
            style={{
              fontSize: "1.75rem",
              lineHeight: "2.8",
              letterSpacing: "0.05em",
              wordSpacing: "0.25em",
            }}
          >
            {wordRenderData.map((wd, wi) => {
              const myCursorInWord =
                cursorPos >= wd.wordStart &&
                cursorPos <= wd.wordStart + wd.chars.length;

              const ghostsInWord = showGhosts
                ? Object.entries(otherPlayers).filter(
                    ([, p]) =>
                      p.pos >= wd.wordStart &&
                      p.pos <= wd.wordStart + wd.chars.length,
                  )
                : [];

              return (
                <Fragment key={wi}>
                  <span
                    style={{
                      display: "inline-block",
                      whiteSpace: "nowrap",
                    }}
                  >
                    {wd.chars.map((ch, ci) => {
                      const isMyCursor = ch.globalIndex === cursorPos;
                      const ghostsHere = ghostsInWord.filter(
                        ([, p]) => p.pos === ch.globalIndex,
                      );

                      return (
                        <Fragment key={ci}>
                          {/* Ghost cursors */}
                          {ghostsHere.map(([name, player]) => (
                            <span
                              key={`ghost-${name}`}
                              className="relative"
                              style={{
                                display: "inline-block",
                                width: 2,
                                marginRight: -2,
                              }}
                            >
                              <span
                                style={{
                                  display: "inline-block",
                                  width: 2,
                                  height: "1.2em",
                                  background: player.color,
                                  opacity: 0.3,
                                  verticalAlign: "text-bottom",
                                  borderRadius: 1,
                                }}
                              />
                              <span
                                className="absolute whitespace-nowrap pointer-events-none select-none"
                                style={{
                                  bottom: "100%",
                                  left: 0,
                                  fontSize: "0.5rem",
                                  lineHeight: 1,
                                  color: player.color,
                                  opacity: 0.5,
                                  fontWeight: 500,
                                }}
                              >
                                {name}
                              </span>
                            </span>
                          ))}

                          {/* My cursor */}
                          {isMyCursor && (
                            <span
                              className="game-cursor"
                              style={{
                                display: "inline-block",
                                width: 2,
                                height: "1.2em",
                                background: "currentColor",
                                verticalAlign: "text-bottom",
                                marginRight: -2,
                                borderRadius: 1,
                              }}
                            />
                          )}

                          {/* Character */}
                          <span
                            className={
                              ch.state === "correct"
                                ? "text-foreground"
                                : ch.state === "incorrect"
                                  ? "text-red-500"
                                  : "text-muted-foreground/40"
                            }
                            style={
                              ch.state === "incorrect"
                                ? { background: "rgba(239,68,68,0.1)" }
                                : undefined
                            }
                          >
                            {ch.char}
                          </span>
                        </Fragment>
                      );
                    })}

                    {/* Extra chars beyond word length */}
                    {wd.extraChars.map((ch, ei) => (
                      <span
                        key={`extra-${ei}`}
                        className="text-red-400/70"
                        style={{ background: "rgba(239,68,68,0.1)" }}
                      >
                        {ch}
                      </span>
                    ))}

                    {/* Cursor at end of this word */}
                    {myCursorInWord &&
                      cursorPos === wd.wordStart + wd.chars.length && (
                        <span
                          className="game-cursor"
                          style={{
                            display: "inline-block",
                            width: 2,
                            height: "1.2em",
                            background: "currentColor",
                            verticalAlign: "text-bottom",
                            borderRadius: 1,
                          }}
                        />
                      )}

                    {/* Ghost cursors at end-of-word position */}
                    {showGhosts &&
                      ghostsInWord
                        .filter(
                          ([, p]) => p.pos === wd.wordStart + wd.chars.length,
                        )
                        .map(([name, player]) => (
                          <span
                            key={`ghost-end-${name}`}
                            className="relative"
                            style={{
                              display: "inline-block",
                              width: 2,
                              marginRight: -2,
                            }}
                          >
                            <span
                              style={{
                                display: "inline-block",
                                width: 2,
                                height: "1.2em",
                                background: player.color,
                                opacity: 0.3,
                                verticalAlign: "text-bottom",
                                borderRadius: 1,
                              }}
                            />
                            <span
                              className="absolute whitespace-nowrap pointer-events-none select-none"
                              style={{
                                bottom: "100%",
                                left: 0,
                                fontSize: "0.5rem",
                                lineHeight: 1,
                                color: player.color,
                                opacity: 0.5,
                                fontWeight: 500,
                              }}
                            >
                              {name}
                            </span>
                          </span>
                        ))}
                  </span>

                  {/* Space between words */}
                  {wi < wordRenderData.length - 1 && (
                    <span
                      className={
                        wd.wordIndex < currentWordIndex
                          ? "text-foreground"
                          : "text-muted-foreground/40"
                      }
                    >
                      {" "}
                    </span>
                  )}
                </Fragment>
              );
            })}
          </div>
        </div>

        {/* ---- Bottom controls (singleplayer only) ---- */}
        {mode === "single" && (
          <div className="mt-10 flex items-center gap-6">
            {!isFinished && (
              <button
                onClick={(e) => {
                  e.stopPropagation();
                  handleRetry();
                }}
                className="text-muted-foreground bg-transparent border-none cursor-pointer p-2 hover:opacity-70 transition-opacity"
                title="Restart"
              >
                <RotateCcw size={18} />
              </button>
            )}
          </div>
        )}

        {/* ---- Multiplayer bottom controls (ghost toggle only) ---- */}
        {mode === "multi" && !isFinished && (
          <div className="mt-10 flex items-center gap-6">
            <button
              onClick={(e) => {
                e.stopPropagation();
                setShowGhosts((v) => !v);
              }}
              className="text-muted-foreground bg-transparent border-none cursor-pointer p-2 hover:opacity-70 transition-opacity flex items-center gap-1.5"
              title={showGhosts ? "Hide other cursors" : "Show other cursors"}
            >
              {showGhosts ? <Eye size={16} /> : <EyeOff size={16} />}
              <span className="text-xs">
                {showGhosts ? "hide ghosts" : "show ghosts"}
              </span>
            </button>
          </div>
        )}

        {/* ---- Multiplayer progress bars ---- */}
        {mode === "multi" && !isFinished && (
          <div className="w-full mt-8 space-y-3">
            {/* My progress */}
            <ProgressBar label={myNameRef.current} pct={myPct} wpm={wpm} isMe />
            {/* Other players */}
            {Object.entries(otherPlayers).map(([name, player]) => (
              <ProgressBar
                key={name}
                label={name}
                pct={text ? Math.round((player.pos / text.length) * 100) : 0}
                wpm={player.wpm}
                color={player.color}
              />
            ))}
          </div>
        )}
      </div>

      {/* ---- Results overlay ---- */}
      {isFinished && (
        <div className="fixed inset-0 z-50 bg-background flex items-center justify-center font-mono">
          <div className="text-center">
            <div className="text-7xl font-bold text-foreground leading-none">
              {wpm}
            </div>
            <div className="text-muted-foreground text-sm mt-1">wpm</div>

            <div className="mt-7">
              <div className="text-foreground text-3xl font-bold">
                {timeStr}s
              </div>
              <div className="text-muted-foreground text-sm">time</div>
            </div>

            {mode === "multi" && raceResults.length > 0 && (
              <div className="mt-7">
                <div className="text-muted-foreground text-xs mb-2 uppercase tracking-widest">
                  Results
                </div>
                {raceResults.map((r, i) => (
                  <div
                    key={i}
                    className={`text-sm mb-1 ${r.isMe ? "text-foreground font-semibold" : "text-muted-foreground"}`}
                  >
                    #{i + 1} {r.isMe ? "You" : r.name} — {r.wpm} wpm
                  </div>
                ))}
              </div>
            )}

            <div className="flex gap-4 justify-center mt-9">
              {mode === "single" ? (
                <>
                  <button
                    onClick={handleRetry}
                    className="bg-foreground text-background border-none px-8 py-3 rounded-lg font-bold cursor-pointer text-sm font-mono hover:opacity-90 transition-opacity"
                  >
                    Play Again
                  </button>
                  <button
                    onClick={() => navigate("/")}
                    className="bg-transparent text-muted-foreground border border-muted-foreground px-8 py-3 rounded-lg font-bold cursor-pointer text-sm font-mono hover:bg-muted transition-colors"
                  >
                    Home
                  </button>
                </>
              ) : (
                <button
                  onClick={() => navigate(`/room/${roomId}/lobby`)}
                  className="bg-foreground text-background border-none px-8 py-3 rounded-lg font-bold cursor-pointer text-sm font-mono hover:opacity-90 transition-opacity"
                >
                  Back to Lobby
                </button>
              )}
            </div>
          </div>
        </div>
      )}

      {/* Cursor blink animation */}
      <style>{`
        @keyframes cursorBlink {
          0%, 49% { opacity: 1; }
          50%, 100% { opacity: 0; }
        }
        .game-cursor {
          animation: cursorBlink 1s step-end infinite;
        }
      `}</style>
    </div>
  );
}

// ---------------------------------------------------------------------------
// Progress bar sub-component
// ---------------------------------------------------------------------------
function ProgressBar({ label, pct, wpm, color, isMe = false }) {
  return (
    <div className="flex items-center gap-3">
      <span
        className="text-xs w-20 truncate text-right font-medium"
        style={{ color: isMe ? "hsl(var(--foreground))" : color || "#6b7280" }}
      >
        {label}
      </span>
      <div
        className="flex-1 h-3 rounded-full overflow-hidden"
        style={{ background: "hsl(var(--muted))" }}
      >
        <div
          className="h-full rounded-full transition-all duration-300 ease-out"
          style={{
            width: `${pct}%`,
            background: isMe ? "hsl(var(--foreground))" : color || "#6b7280",
            opacity: isMe ? 1 : 0.7,
          }}
        />
      </div>
      <span
        className="text-xs w-14 font-mono"
        style={{ color: isMe ? "hsl(var(--foreground))" : color || "#6b7280" }}
      >
        {wpm} wpm
      </span>
    </div>
  );
}
