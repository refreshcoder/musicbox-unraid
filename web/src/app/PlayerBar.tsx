type Props = {
  status?: string;
  volume?: number;
  positionMs?: number;
  durationMs?: number;
  onPrev: () => void;
  onToggle: () => void;
  onNext: () => void;
  onSetVolume: (v: number) => void;
};

function fmt(ms?: number) {
  const v = Math.max(0, Math.floor((ms ?? 0) / 1000));
  const m = Math.floor(v / 60);
  const s = v % 60;
  return `${m}:${String(s).padStart(2, "0")}`;
}

export function PlayerBar(props: Props) {
  const statusText =
    props.status === "play"
      ? "播放中"
      : props.status === "pause"
        ? "暂停"
        : props.status
          ? props.status
          : "MPD未配置";

  const pos = props.positionMs ?? 0;
  const dur = props.durationMs ?? 0;
  const pct = dur > 0 ? Math.min(100, Math.max(0, (pos / dur) * 100)) : 0;

  return (
    <div className="sticky top-0 z-50 border-b bg-white/80 backdrop-blur">
      <div className="mx-auto max-w-5xl px-3 h-14 flex items-center justify-between">
        <div className="min-w-0">
          <div className="text-sm font-medium truncate">MusicBox</div>
          <div className="text-xs text-gray-500 truncate">{statusText}</div>
        </div>
        <div className="flex items-center gap-2">
          <button
            className="px-3 py-1.5 rounded bg-gray-100 text-sm"
            type="button"
            onClick={props.onPrev}
          >
            上一首
          </button>
          <button
            className="px-3 py-1.5 rounded bg-black text-white text-sm"
            type="button"
            onClick={props.onToggle}
          >
            播放/暂停
          </button>
          <button
            className="px-3 py-1.5 rounded bg-gray-100 text-sm"
            type="button"
            onClick={props.onNext}
          >
            下一首
          </button>
        </div>
      </div>
      <div className="mx-auto max-w-5xl px-3 pb-2">
        <div className="flex items-center gap-3">
          <div className="text-xs text-gray-500 w-12 text-right">{fmt(pos)}</div>
          <div className="h-1.5 rounded bg-gray-200 flex-1 overflow-hidden">
            <div className="h-full bg-black" style={{ width: `${pct}%` }} />
          </div>
          <div className="text-xs text-gray-500 w-12">{fmt(dur)}</div>
          <input
            className="w-28"
            type="range"
            min={0}
            max={100}
            value={props.volume ?? 0}
            onChange={(e) => props.onSetVolume(Number(e.target.value))}
          />
        </div>
      </div>
    </div>
  );
}
