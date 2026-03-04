import { useState, useEffect, useCallback, useRef } from "react";

// ---------------------------------------------------------------------------
// Word Bank  (~200 common English words + numbers)
// ---------------------------------------------------------------------------
export const WORD_BANK = [
  "the",
  "be",
  "to",
  "of",
  "and",
  "a",
  "in",
  "that",
  "have",
  "it",
  "for",
  "not",
  "on",
  "with",
  "as",
  "you",
  "do",
  "at",
  "this",
  "but",
  "his",
  "by",
  "from",
  "they",
  "we",
  "say",
  "her",
  "she",
  "or",
  "an",
  "will",
  "my",
  "one",
  "all",
  "would",
  "there",
  "their",
  "what",
  "so",
  "up",
  "out",
  "if",
  "about",
  "who",
  "get",
  "which",
  "go",
  "me",
  "when",
  "make",
  "can",
  "like",
  "time",
  "no",
  "just",
  "him",
  "know",
  "take",
  "people",
  "into",
  "year",
  "your",
  "good",
  "some",
  "could",
  "them",
  "see",
  "other",
  "than",
  "then",
  "now",
  "look",
  "only",
  "come",
  "its",
  "over",
  "think",
  "also",
  "back",
  "after",
  "use",
  "two",
  "how",
  "our",
  "work",
  "first",
  "well",
  "way",
  "even",
  "new",
  "want",
  "because",
  "any",
  "these",
  "give",
  "day",
  "most",
  "us",
  "great",
  "between",
  "need",
  "large",
  "often",
  "hand",
  "high",
  "place",
  "hold",
  "turn",
  "such",
  "here",
  "why",
  "move",
  "play",
  "small",
  "number",
  "off",
  "always",
  "next",
  "open",
  "seem",
  "together",
  "white",
  "children",
  "begin",
  "got",
  "walk",
  "example",
  "ease",
  "paper",
  "group",
  "music",
  "those",
  "both",
  "mark",
  "book",
  "letter",
  "until",
  "mile",
  "river",
  "car",
  "feet",
  "care",
  "second",
  "enough",
  "plain",
  "girl",
  "usual",
  "young",
  "ready",
  "above",
  "ever",
  "red",
  "list",
  "though",
  "feel",
  "talk",
  "bird",
  "soon",
  "body",
  "dog",
  "family",
  "direct",
  "leave",
  "song",
  "door",
  "black",
  "short",
  "class",
  "wind",
  "question",
  "happen",
  "complete",
  "ship",
  "area",
  "half",
  "rock",
  "order",
  "fire",
  "south",
  "problem",
  "piece",
  "told",
  "knew",
  "pass",
  "since",
  "top",
  "whole",
  "king",
  "space",
  "heard",
  "best",
  "hour",
  "better",
  "true",
  "during",
  "hundred",
  "five",
  "remember",
  "step",
  "early",
  "west",
  "ground",
  "interest",
  "reach",
  "fast",
  "sing",
  "listen",
  "six",
  "table",
  "travel",
  "less",
  "morning",
  "ten",
  "simple",
  "several",
  "toward",
  "night",
  "storm",
  "bright",
  "stand",
  "change",
  "follow",
  "point",
  "write",
  "read",
  "earth",
  "light",
  "hard",
  "start",
  "run",
  "ask",
  "home",
  "own",
  "call",
  "he",
  "must",
  "world",
  "person",
  "never",
  "present",
  "many",
];

// ---------------------------------------------------------------------------
// Utilities
// ---------------------------------------------------------------------------

/** Pick wordCount random words and join them with spaces. */
export function generateText(wordCount = 25) {
  const words = [];
  for (let i = 0; i < wordCount; i++) {
    words.push(WORD_BANK[Math.floor(Math.random() * WORD_BANK.length)]);
  }
  return words.join(" ");
}

// ---------------------------------------------------------------------------
// Seeded PRNG — ensures all clients with the same seed get the same text
// ---------------------------------------------------------------------------
function createSeededRng(seed) {
  let h = 0;
  for (let i = 0; i < seed.length; i++) {
    h = ((h << 5) - h + seed.charCodeAt(i)) | 0;
  }
  // Simple LCG (linear congruential generator)
  return function next() {
    h = (h * 1664525 + 1013904223) | 0;
    return (h >>> 0) / 0xffffffff;
  };
}

/**
 * Deterministic version of generateText.
 * All clients that call this with the same `seed` string will get
 * the exact same sequence of words.
 */
export function generateTextSeeded(seed, wordCount = 25) {
  const rng = createSeededRng(seed);
  const words = [];
  for (let i = 0; i < wordCount; i++) {
    words.push(WORD_BANK[Math.floor(rng() * WORD_BANK.length)]);
  }
  return words.join(" ");
}

/** Standard WPM: (correct chars / 5) / minutes. */
export function calculateWPM(correctChars, elapsedMs) {
  if (elapsedMs < 500) return 0;
  const minutes = elapsedMs / 60000;
  return Math.round(correctChars / 5 / minutes);
}

// ---------------------------------------------------------------------------
// useGameLogic hook  —  WORD-BY-WORD approach
// ---------------------------------------------------------------------------
/**
 * The user types one word at a time.
 *  - Can only advance to the next word (via Space) if the current word matches.
 *  - Backspace only works within the current word (cannot revisit previous words).
 *  - The last word finishes the game on the final correct character (no trailing space).
 *
 * Returns everything the Game component needs to render characters and cursors.
 */
export function useGameLogic({ text, onProgress, onFinish, forcedStartTime }) {
  // "idle" | "playing" | "finished"
  const [gameState, setGameState] = useState("idle");
  const [currentWordIndex, setCurrentWordIndex] = useState(0);
  const [currentInput, setCurrentInput] = useState("");
  const [wpm, setWpm] = useState(0);
  const [timeElapsed, setTimeElapsed] = useState(0);

  // Track total correct characters for WPM
  const correctCharsRef = useRef(0);
  const startTimeRef = useRef(null);
  const intervalRef = useRef(null);
  const onProgressRef = useRef(onProgress);
  const onFinishRef = useRef(onFinish);

  // Keep callback refs fresh
  useEffect(() => {
    onProgressRef.current = onProgress;
  });
  useEffect(() => {
    onFinishRef.current = onFinish;
  });

  // Split text into words (memoish via ref to avoid re-splits)
  const wordsRef = useRef([]);
  useEffect(() => {
    wordsRef.current = text ? text.split(" ") : [];
  }, [text]);

  const words = text ? text.split(" ") : [];

  // Full reset when the prompt text changes
  useEffect(() => {
    setCurrentWordIndex(0);
    setCurrentInput("");
    setGameState("idle");
    setWpm(0);
    setTimeElapsed(0);
    correctCharsRef.current = 0;
    startTimeRef.current = null;
    clearInterval(intervalRef.current);
  }, [text]);

  // Server-driven start: when forcedStartTime is provided, immediately
  // set the start time and enter "playing" state so the timer runs from
  // the shared clock without waiting for a keypress.
  useEffect(() => {
    if (forcedStartTime && gameState === "idle") {
      startTimeRef.current = forcedStartTime;
      setGameState("playing");
    }
  }, [forcedStartTime, gameState]);

  // Timer: tick every 100 ms while playing
  useEffect(() => {
    if (gameState === "playing") {
      intervalRef.current = setInterval(() => {
        if (startTimeRef.current) {
          setTimeElapsed(Date.now() - startTimeRef.current);
        }
      }, 100);
    } else {
      clearInterval(intervalRef.current);
    }
    return () => clearInterval(intervalRef.current);
  }, [gameState]);

  // --- Derived state ---
  const isStarted = gameState !== "idle";
  const isFinished = gameState === "finished";

  // Compute the cursor position in the full text string
  // = sum of completed word lengths + spaces + currentInput.length
  let cursorPos = 0;
  for (let w = 0; w < currentWordIndex; w++) {
    cursorPos += words[w].length + 1; // word + space
  }
  cursorPos += currentInput.length;

  // Build per-character state array for the entire text
  const charStates = [];
  if (text) {
    let idx = 0;
    for (let w = 0; w < words.length; w++) {
      const word = words[w];
      for (let c = 0; c < word.length; c++) {
        if (w < currentWordIndex) {
          // Completed word — always correct (we only allow advancing correct words)
          charStates[idx] = "correct";
        } else if (w === currentWordIndex) {
          // Current word
          if (c < currentInput.length) {
            charStates[idx] =
              currentInput[c] === word[c] ? "correct" : "incorrect";
          } else {
            charStates[idx] = "untyped";
          }
        } else {
          charStates[idx] = "untyped";
        }
        idx++;
      }
      // Space after word (except the last)
      if (w < words.length - 1) {
        if (w < currentWordIndex) {
          charStates[idx] = "correct";
        } else {
          charStates[idx] = "untyped";
        }
        idx++;
      }
    }
  }

  // --- Key handler ---
  const handleKeyDown = useCallback(
    (e) => {
      if (!text || gameState === "finished") return;
      if (e.ctrlKey || e.altKey || e.metaKey) return;
      if (e.key === "Tab") {
        e.preventDefault();
        return;
      }

      const currentWords = wordsRef.current;
      if (currentWords.length === 0) return;

      // Start timer on first keypress (only if not already started by forcedStartTime)
      if (gameState === "idle" && e.key.length === 1) {
        startTimeRef.current = Date.now();
        setGameState("playing");
      }

      if (e.key === "Backspace") {
        e.preventDefault();
        // Only delete within current word, can't go to previous word
        setCurrentInput((prev) => prev.slice(0, -1));
        return;
      }

      // Space key — attempt to advance to next word
      if (e.key === " ") {
        e.preventDefault();
        setCurrentInput((prev) => {
          const targetWord = currentWords[currentWordIndex];
          if (prev !== targetWord) {
            // Word doesn't match — don't advance
            return prev;
          }
          // Word is correct! Advance.
          const correctSoFar = correctCharsRef.current + targetWord.length + 1; // +1 for the space
          correctCharsRef.current = correctSoFar;

          const elapsed = startTimeRef.current
            ? Date.now() - startTimeRef.current
            : 0;
          const curWpm = calculateWPM(correctSoFar, elapsed);
          setWpm(curWpm);

          const nextIndex = currentWordIndex + 1;
          setCurrentWordIndex(nextIndex);

          // Compute position in full text
          let pos = 0;
          for (let w = 0; w <= currentWordIndex; w++) {
            pos += currentWords[w].length + 1;
          }
          onProgressRef.current?.({ pos, wpm: curWpm });

          return ""; // clear input for next word
        });
        return;
      }

      // Regular character
      if (e.key.length !== 1) return;
      e.preventDefault();

      setCurrentInput((prev) => {
        const targetWord = currentWords[currentWordIndex];
        // Don't allow typing beyond word length + a few extra for visual feedback
        if (prev.length >= targetWord.length + 8) return prev;

        const newInput = prev + e.key;

        // Check if this is the LAST word and it's now fully & correctly typed
        if (
          currentWordIndex === currentWords.length - 1 &&
          newInput === targetWord
        ) {
          const correctTotal = correctCharsRef.current + targetWord.length;
          const elapsed = startTimeRef.current
            ? Date.now() - startTimeRef.current
            : 0;
          const curWpm = calculateWPM(correctTotal, elapsed);
          setWpm(curWpm);
          setGameState("finished");
          clearInterval(intervalRef.current);
          setTimeElapsed(elapsed);
          // Send a final progress update so other players' ghost cursors
          // move all the way to the end (avoids the "2 chars short" bug
          // caused by throttling dropping the last update).
          onProgressRef.current?.({ pos: text.length, wpm: curWpm });
          onFinishRef.current?.({ wpm: curWpm, elapsed });
        } else {
          // Update WPM while typing
          const elapsed = startTimeRef.current
            ? Date.now() - startTimeRef.current
            : 0;
          if (elapsed > 500) {
            // Count correct chars: all completed words + correct chars in current input
            let cc = correctCharsRef.current;
            for (let c = 0; c < newInput.length; c++) {
              if (c < targetWord.length && newInput[c] === targetWord[c]) cc++;
            }
            const curWpm = calculateWPM(cc, elapsed);
            setWpm(curWpm);
          }

          // Send progress
          let pos = 0;
          for (let w = 0; w < currentWordIndex; w++) {
            pos += currentWords[w].length + 1;
          }
          pos += newInput.length;
          const elapsed2 = startTimeRef.current
            ? Date.now() - startTimeRef.current
            : 0;
          onProgressRef.current?.({
            pos,
            wpm: calculateWPM(correctCharsRef.current, elapsed2),
          });
        }

        return newInput;
      });
    },
    [text, gameState, currentWordIndex],
  );

  const reset = useCallback(() => {
    setCurrentWordIndex(0);
    setCurrentInput("");
    setGameState("idle");
    setWpm(0);
    setTimeElapsed(0);
    correctCharsRef.current = 0;
    startTimeRef.current = null;
    clearInterval(intervalRef.current);
  }, []);

  return {
    charStates,
    cursorPos,
    currentWordIndex,
    currentInput,
    words,
    wpm,
    timeElapsed,
    isStarted,
    isFinished,
    handleKeyDown,
    reset,
  };
}
