import { useMemo } from "react";

export type TabKey =
  | "now"
  | "queue"
  | "library"
  | "upload"
  | "bluetooth"
  | "diag";

export function useTabs() {
  const tabs = useMemo(
    () => [
      { key: "now" as const, label: "播放" },
      { key: "queue" as const, label: "队列" },
      { key: "library" as const, label: "曲库" },
      { key: "upload" as const, label: "上传&BV" },
      { key: "bluetooth" as const, label: "蓝牙" },
      { key: "diag" as const, label: "诊断" },
    ],
    [],
  );
  return { tabs };
}

export function TabNav(props: {
  active: TabKey;
  onChange: (k: TabKey) => void;
}) {
  const { tabs } = useTabs();

  return (
    <div className="sticky top-14 z-40 border-b bg-white/80 backdrop-blur">
      <div className="mx-auto max-w-5xl px-3">
        <div className="flex gap-2 overflow-x-auto py-2">
          {tabs.map((t) => (
            <button
              key={t.key}
              className={[
                "px-3 py-1.5 rounded-full text-sm whitespace-nowrap",
                props.active === t.key
                  ? "bg-black text-white"
                  : "bg-gray-100 text-gray-700",
              ].join(" ")}
              onClick={() => props.onChange(t.key)}
              type="button"
            >
              {t.label}
            </button>
          ))}
        </div>
      </div>
    </div>
  );
}

