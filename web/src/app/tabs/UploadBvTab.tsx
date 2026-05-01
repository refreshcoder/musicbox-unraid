import { useMemo, useState } from "react";
import { postJSON } from "../../lib/api";

export type TaskStatus = "queued" | "running" | "success" | "failed" | "canceled";

export type Task = {
  id: string;
  type: string;
  input: string;
  status: TaskStatus;
  stage?: string;
  progress01?: number;
  resultPath?: string;
  error?: string;
  createdAt?: number;
  updatedAt?: number;
};

function pct(v?: number) {
  const p = Math.round(Math.max(0, Math.min(1, v ?? 0)) * 100);
  return `${p}%`;
}

export function UploadBvTab(props: { tasks: Task[]; onRefresh: () => void }) {
  const [bv, setBv] = useState("");
  const [submitting, setSubmitting] = useState(false);

  const sorted = useMemo(() => {
    const items = [...props.tasks];
    items.sort((a, b) => (b.createdAt ?? 0) - (a.createdAt ?? 0));
    return items;
  }, [props.tasks]);

  const submit = async () => {
    const v = bv.trim();
    if (!v) return;
    setSubmitting(true);
    try {
      await postJSON("/api/v1/tasks/bv", { bv: v });
      setBv("");
      props.onRefresh();
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <div className="space-y-3">
      <div className="text-sm font-medium">上传 & BV</div>

      <div className="rounded border bg-white p-3 space-y-2">
        <div className="text-sm font-medium">BV 下载</div>
        <div className="flex flex-col sm:flex-row gap-2">
          <input
            className="flex-1 rounded border px-3 py-2 text-sm"
            placeholder="BVxxxx 或者直接粘贴链接"
            value={bv}
            onChange={(e) => setBv(e.target.value)}
          />
          <button
            className="px-3 py-2 rounded bg-black text-white text-sm disabled:opacity-50"
            type="button"
            disabled={submitting}
            onClick={() => void submit()}
          >
            {submitting ? "提交中" : "开始下载"}
          </button>
          <button
            className="px-3 py-2 rounded bg-gray-100 text-sm"
            type="button"
            onClick={props.onRefresh}
          >
            刷新
          </button>
        </div>
        <div className="text-xs text-gray-500">
          默认行为：只入库不打扰（不自动加入队列、不自动播放）
        </div>
      </div>

      <div className="rounded border bg-white">
        {sorted.length === 0 ? (
          <div className="p-4 text-sm text-gray-500">暂无任务</div>
        ) : (
          <ul className="divide-y">
            {sorted.map((t) => (
              <li key={t.id} className="p-3 space-y-2">
                <div className="flex items-center justify-between gap-3">
                  <div className="min-w-0">
                    <div className="text-sm font-medium truncate">
                      {t.type}:{t.input}
                    </div>
                    <div className="text-xs text-gray-500 truncate">
                      {t.status}
                      {t.stage ? ` / ${t.stage}` : ""}
                      {t.resultPath ? ` / ${t.resultPath}` : ""}
                    </div>
                  </div>
                  <div className="flex items-center gap-2">
                    {t.status === "queued" || t.status === "running" ? (
                      <button
                        className="px-2 py-1 rounded bg-gray-100 text-xs"
                        type="button"
                        onClick={() =>
                          postJSON(`/api/v1/tasks/${encodeURIComponent(t.id)}/cancel`)
                            .then(() => props.onRefresh())
                            .catch(() => props.onRefresh())
                        }
                      >
                        取消
                      </button>
                    ) : null}
                  </div>
                </div>
                <div className="h-1.5 rounded bg-gray-200 overflow-hidden">
                  <div
                    className="h-full bg-black"
                    style={{ width: pct(t.progress01) }}
                  />
                </div>
                {t.error ? (
                  <div className="text-xs text-red-700 bg-red-50 border rounded p-2">
                    {t.error}
                  </div>
                ) : null}
              </li>
            ))}
          </ul>
        )}
      </div>
    </div>
  );
}

