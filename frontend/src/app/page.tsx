"use client";

import { useEffect, useRef, useCallback, useState, useMemo } from "react";
import { useQuery } from "@tanstack/react-query";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { useStore } from "@/lib/store";
import { fetchClientRules, runAllClientRules } from "@/lib/client-engine";
import { WSConnection, type ServerMessage } from "@/lib/api";
import type { ClientRule } from "@/lib/client-engine";
import type { FlagState } from "@/lib/store";
import CodeMirrorEditor from "@/components/CodeMirrorEditor";
import Codifier from "@/components/Codifier";
import ScorePanel from "@/components/ScorePanel";
import TextStats from "@/components/TextStats";
import FlagCard from "@/components/FlagCard";

/* ------------------------------------------------------------------ */
/*  react-query client                                                 */
/* ------------------------------------------------------------------ */

const queryClient = new QueryClient();

/* ------------------------------------------------------------------ */
/*  Constants                                                          */
/* ------------------------------------------------------------------ */

const DEBOUNCE_MS = 500;

/* ------------------------------------------------------------------ */
/*  Main content                                                       */
/* ------------------------------------------------------------------ */

function HomeContent() {
  const text = useStore((s) => s.text);
  const flags = useStore((s) => s.flags);
  const scores = useStore((s) => s.scores);
  const wordCount = useStore((s) => s.wordCount);
  const textHash = useStore((s) => s.textHash);
  const sentenceCountStore = useStore((s) => s.sentenceCount);
  const charCountStore = useStore((s) => s.charCount);

  const setText = useStore((s) => s.setText);
  const addTentativeFlags = useStore((s) => s.addTentativeFlags);
  const updateTentativeScores = useStore((s) => s.updateTentativeScores);
  const rejectFlag = useStore((s) => s.rejectFlag);
  const applyFlagFix = useStore((s) => s.applyFlagFix);

  const [isApplyingAll, setIsApplyingAll] = useState(false);

  /* ── Floating card state ── */
  const [selectedFlag, setSelectedFlag] = useState<FlagState | null>(null);
  const [cardPosition, setCardPosition] = useState({ x: 0, y: 0 });

  const wsRef = useRef<WSConnection | null>(null);
  const debounceRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const rulesRef = useRef<ClientRule[]>([]);

  /* ---------- Fetch client rules ---------- */
  const rulesQuery = useQuery({
    queryKey: ["client-rules"],
    queryFn: fetchClientRules,
    staleTime: 5 * 60 * 1000,
  });

  useEffect(() => {
    if (rulesQuery.data) {
      rulesRef.current = rulesQuery.data;
    }
  }, [rulesQuery.data]);

  /* ---------- WS connection ---------- */
  useEffect(() => {
    const ws = new WSConnection();
    wsRef.current = ws;

    const onMessage = (msg: ServerMessage) => {
      if (msg.type === "check_result") {
        const state = useStore.getState();
        if (msg.textHash !== state.textHash) return; // stale
        state.mergeFlagsFromServerWire(msg.flags);
        state.setScores(msg.scores);
        if (msg.session_id) state.setSessionId(msg.session_id);
      }
    };

    const onStatusChange = (status: import("@/lib/api").ConnectionStatus) => {
      useStore.getState().setConnectionStatus(status);
    };

    ws.connect({ onMessage, onStatusChange });

    return () => {
      ws.disconnect();
      wsRef.current = null;
    };
  }, []); // mount once — uses getState() for freshness

  /* ---------- Debounced WS send on text change ---------- */
  useEffect(() => {
    if (debounceRef.current) clearTimeout(debounceRef.current);

    if (!text) return;

    debounceRef.current = setTimeout(() => {
      const ws = wsRef.current;
      if (ws?.connected) {
        ws.send({ type: "check", text, textHash });
      }
    }, DEBOUNCE_MS);

    return () => {
      if (debounceRef.current) clearTimeout(debounceRef.current);
    };
  }, [text, textHash]);

  /* ---------- Client rules on text change (instant, before WS) ---------- */
  useEffect(() => {
    if (!text) return;
    const clientFlags = runAllClientRules(rulesRef.current, text);
    addTentativeFlags(clientFlags);
    updateTentativeScores();
  }, [text]); // eslint-disable-line react-hooks/exhaustive-deps

  /* ---------- FlagCard stale detection — close on text delete ---------- */
  useEffect(() => {
    if (selectedFlag && !flags[selectedFlag.id]) {
      setSelectedFlag(null);
    }
  }, [flags, selectedFlag]);

  /* ---------- Action handlers ---------- */
  const handleAccept = useCallback(
    (id: string) => {
      const result = applyFlagFix(id);
      if (result && wsRef.current?.connected) {
        const state = useStore.getState();
        wsRef.current.send({
          type: "check",
          text: state.text,
          textHash: state.textHash,
        });
      }
      setSelectedFlag(null);
    },
    [applyFlagFix],
  );

  const handleReject = useCallback(
    (id: string) => {
      rejectFlag(id);
      wsRef.current?.send({ type: "reject", flagId: id });
      setSelectedFlag(null);
    },
    [rejectFlag],
  );

  const handleApplyAll = useCallback(() => {
    setIsApplyingAll(true);

    const state = useStore.getState();
    const acceptedIds = Object.values(state.flags)
      .filter((f) => f.status === "accepted")
      .map((f) => f.id);

    const result = state.applyAllAccepted();
    if (result && wsRef.current?.connected) {
      wsRef.current.send({ type: "applyAll", flagIds: acceptedIds });
      const updated = useStore.getState();
      wsRef.current.send({
        type: "check",
        text: updated.text,
        textHash: updated.textHash,
      });
    }

    setIsApplyingAll(false);
  }, []);

  const handleTextChange = useCallback(
    (t: string) => {
      setText(t);
    },
    [setText],
  );

  /* ---------- Flag click handler: show floating card ---------- */
  const handleFlagClick = useCallback(
    (flagId: string) => {
      const allFlags = useStore.getState().flags;
      const flag = allFlags[flagId];
      if (!flag || flag.status === "applied") {
        setSelectedFlag(null);
        return;
      }

      // Get position near the flagged text in the editor
      const flagEl = document.querySelector(`[data-flag-id="${flagId}"]`);
      if (flagEl) {
        const rect = flagEl.getBoundingClientRect();
        // Position the card to the right of the flagged element
        setCardPosition({
          x: Math.min(rect.right + 16, window.innerWidth - 400),
          y: Math.max(rect.top - 20, 10),
        });
      } else {
        // Fallback position
        setCardPosition({ x: 600, y: 120 });
      }

      setSelectedFlag(flag);
    },
    [],
  );

  const handleCloseCard = useCallback(() => {
    setSelectedFlag(null);
  }, []);

  /* ---------- Computed ---------- */
  const activeFlags = Object.values(flags).filter(
    (f) => f.status !== "applied",
  );
  const navigableFlags = useMemo(
    () =>
      Object.values(flags)
        .filter((f) => f.status !== "applied" && f.status !== "rejected")
        .sort((a, b) => a.span.from - b.span.from || a.span.to - b.span.to),
    [flags],
  );
  const acceptedCount = Object.values(flags).filter(
    (f) => f.status !== "rejected" && f.status !== "applied" && f.autoFix !== undefined,
  ).length;

  /* ---------- Keyboard: next/prev flag (Alt+↓ / Alt+↑) ---------- */
  const selectFlagById = useCallback(
    (flagId: string) => {
      handleFlagClick(flagId);
      requestAnimationFrame(() => {
        document
          .querySelector(`[data-flag-id="${flagId}"]`)
          ?.scrollIntoView({ block: "nearest", behavior: "smooth" });
      });
    },
    [handleFlagClick],
  );

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (!e.altKey || e.metaKey || e.ctrlKey) return;
      if (e.key !== "ArrowDown" && e.key !== "ArrowUp") return;
      if (navigableFlags.length === 0) return;
      e.preventDefault();
      e.stopPropagation();

      const curIdx = selectedFlag
        ? navigableFlags.findIndex((f) => f.id === selectedFlag.id)
        : -1;
      let nextIdx: number;
      if (e.key === "ArrowDown") {
        nextIdx = curIdx < 0 ? 0 : (curIdx + 1) % navigableFlags.length;
      } else {
        nextIdx =
          curIdx < 0
            ? navigableFlags.length - 1
            : (curIdx - 1 + navigableFlags.length) % navigableFlags.length;
      }
      selectFlagById(navigableFlags[nextIdx].id);
    };
    const onEsc = (e: KeyboardEvent) => {
      if (e.key === "Escape" && selectedFlag) {
        setSelectedFlag(null);
      }
    };
    window.addEventListener("keydown", onKey, true);
    window.addEventListener("keydown", onEsc);
    return () => {
      window.removeEventListener("keydown", onKey, true);
      window.removeEventListener("keydown", onEsc);
    };
  }, [navigableFlags, selectedFlag, selectFlagById]);

  /* ---------- JSX ---------- */
  return (
    <div className="mx-auto max-w-[980px] px-8 editor-area">
      <h1 className="font-serif text-[28px] leading-[36px] mb-9 tracking-[-.01em]">
        Проверка текста
      </h1>

      <div className="relative">
        <CodeMirrorEditor
          text={text}
          flags={activeFlags}
          onChange={handleTextChange}
          onFlagClick={handleFlagClick}
        />
        {!text && (
          <div
            className="absolute inset-0 flex items-center justify-center pointer-events-none select-none z-10"
            style={{ top: "80px" }}
          >
            <span className="text-[17px] text-[#c0b8a8] font-sans">
              <svg
                className="inline-block mr-2 mb-0.5"
                width="20"
                height="20"
                viewBox="0 0 20 20"
                fill="none"
                stroke="currentColor"
                strokeWidth="2"
                strokeLinecap="round"
              >
                <path d="M3 10h14M12 4l6 6-6 6" />
              </svg>
              Писать сюда
            </span>
          </div>
        )}
      </div>

      {selectedFlag && (
        <FlagCard
          flag={selectedFlag}
          text={text}
          position={cardPosition}
          onAccept={handleAccept}
          onReject={handleReject}
          onClose={handleCloseCard}
        />
      )}

      <Codifier
        flags={flags}
        onApplyAll={acceptedCount > 0 ? handleApplyAll : undefined}
        isApplyingAll={isApplyingAll}
        acceptedCount={acceptedCount}
      />

      <div className="stats-block">
        <ScorePanel
          cleanliness={scores.cleanliness}
          readability={scores.readability}
        />
        <div className="ml-auto">
          <TextStats
            wordCount={wordCount}
            sentenceCount={sentenceCountStore}
            charCount={charCountStore}
            flags={flags}
          />
        </div>
      </div>
    </div>
  );
}

/* ------------------------------------------------------------------ */
/*  Page entry                                                         */
/* ------------------------------------------------------------------ */

export default function Home() {
  return (
    <QueryClientProvider client={queryClient}>
      <HomeContent />
    </QueryClientProvider>
  );
}
