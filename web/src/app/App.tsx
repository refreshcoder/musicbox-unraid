import { useCallback, useEffect, useRef, useState } from "react";
import { getJSON, postJSON } from "../lib/api";
import { connectWs } from "../lib/ws";
import { PlayerBar } from "./PlayerBar";
import { TabNav } from "./Tabs";
import type { TabKey } from "./Tabs";

type StatusResp = {
  player?: {
    status?: string;
    volume?: number;
  };
};

type PlayerProgress = {
  positionMs: number;
  durationMs: number;
  bitrateKbps: number;
};

export function App() {
  const [tab, setTab] = useState<TabKey>("now");
  const wsRef = useRef<WebSocket | null>(null);
  const [playerStatus, setPlayerStatus] = useState<string | undefined>(undefined);
  const [playerVolume, setPlayerVolume] = useState<number | undefined>(undefined);
  const [posMs, setPosMs] = useState<number>(0);
  const [durMs, setDurMs] = useState<number>(0);

  useEffect(() => {
    getJSON<StatusResp>("/api/v1/status")
      .then((st) => {
        setPlayerStatus(st.player?.status);
        setPlayerVolume(st.player?.volume);
      })
      .catch(() => {});
  }, []);

  useEffect(() => {
    const ws = connectWs((evt) => {
      if (evt.type === "player.progress") {
        const p = evt.data as PlayerProgress;
        setPosMs(p.positionMs ?? 0);
        setDurMs(p.durationMs ?? 0);
      }
    });
    wsRef.current = ws;
    return () => {
      ws.close();
      wsRef.current = null;
    };
  }, []);

  const call = useCallback(async (path: string, body?: unknown) => {
    try {
      await postJSON<{ ok: boolean }>(path, body);
      const st = await getJSON<StatusResp>("/api/v1/status");
      setPlayerStatus(st.player?.status);
      setPlayerVolume(st.player?.volume);
    } catch {
      return;
    }
  }, []);

  return (
    <div className="min-h-dvh bg-gray-50">
      <PlayerBar
        status={playerStatus}
        volume={playerVolume}
        positionMs={posMs}
        durationMs={durMs}
        onPrev={() => void call("/api/v1/player/prev")}
        onToggle={() => void call("/api/v1/player/toggle")}
        onNext={() => void call("/api/v1/player/next")}
        onSetVolume={(v) => {
          setPlayerVolume(v);
          void call("/api/v1/player/volume", { volume: v });
        }}
      />
      <TabNav active={tab} onChange={setTab} />
      <main className="mx-auto max-w-5xl px-3 py-4">
        <div className="rounded-lg border bg-white p-4 text-sm text-gray-700">
          当前：{tab}
        </div>
      </main>
    </div>
  );
}
