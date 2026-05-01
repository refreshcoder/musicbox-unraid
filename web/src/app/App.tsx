import { useEffect, useRef, useState } from "react";
import { connectWs } from "../lib/ws";
import { PlayerBar } from "./PlayerBar";
import { TabNav } from "./Tabs";
import type { TabKey } from "./Tabs";

export function App() {
  const [tab, setTab] = useState<TabKey>("now");
  const wsRef = useRef<WebSocket | null>(null);

  useEffect(() => {
    const ws = connectWs(() => {});
    wsRef.current = ws;
    return () => {
      ws.close();
      wsRef.current = null;
    };
  }, []);

  return (
    <div className="min-h-dvh bg-gray-50">
      <PlayerBar />
      <TabNav active={tab} onChange={setTab} />
      <main className="mx-auto max-w-5xl px-3 py-4">
        <div className="rounded-lg border bg-white p-4 text-sm text-gray-700">
          当前：{tab}
        </div>
      </main>
    </div>
  );
}
