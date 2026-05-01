import { useEffect, useState } from "react";
import { getJSON, postJSON } from "../../lib/api";

type QueueItem = {
  pos: number;
  path: string;
  title?: string;
  durationMs?: number;
};

type QueueResp = {
  items: QueueItem[];
};

function fmt(ms?: number) {
  const v = Math.max(0, Math.floor((ms ?? 0) / 1000));
  const m = Math.floor(v / 60);
  const s = v % 60;
  return `${m}:${String(s).padStart(2, "0")}`;
}

export function QueueTab() {
  const [items, setItems] = useState<QueueItem[]>([]);
  const [error, setError] = useState<string | null>(null);

  const refresh = () => {
    setError(null);
    getJSON<QueueResp>("/api/v1/queue")
      .then((r) => setItems(r.items ?? []))
      .catch((e: unknown) => setError(String(e)));
  };

  useEffect(() => {
    refresh();
  }, []);

  return (
    <div className="space-y-3">
      <div className="flex items-center justify-between">
        <div className="text-sm font-medium">队列</div>
        <div className="flex items-center gap-2">
          <button
            className="px-3 py-1.5 rounded bg-gray-100 text-sm"
            type="button"
            onClick={refresh}
          >
            刷新
          </button>
          <button
            className="px-3 py-1.5 rounded bg-gray-900 text-white text-sm"
            type="button"
            onClick={() =>
              postJSON("/api/v1/queue/clear")
                .then(() => refresh())
                .catch(() => refresh())
            }
          >
            清空
          </button>
        </div>
      </div>

      {error ? (
        <div className="rounded border bg-red-50 text-red-700 text-sm p-3">
          {error}
        </div>
      ) : null}

      <div className="rounded border bg-white">
        {items.length === 0 ? (
          <div className="p-4 text-sm text-gray-500">空队列</div>
        ) : (
          <ul className="divide-y">
            {items.map((it) => (
              <li key={it.pos} className="p-3 flex items-center gap-3">
                <div className="w-10 text-xs text-gray-500">{it.pos}</div>
                <div className="min-w-0 flex-1">
                  <div className="text-sm truncate">
                    {it.title ? it.title : it.path}
                  </div>
                  <div className="text-xs text-gray-500 truncate">{it.path}</div>
                </div>
                <div className="w-12 text-xs text-gray-500 text-right">
                  {fmt(it.durationMs)}
                </div>
                <button
                  className="px-2 py-1 rounded bg-gray-100 text-xs"
                  type="button"
                  onClick={() =>
                    postJSON("/api/v1/queue/remove", { pos: it.pos })
                      .then(() => refresh())
                      .catch(() => refresh())
                  }
                >
                  删除
                </button>
              </li>
            ))}
          </ul>
        )}
      </div>
    </div>
  );
}

