export function PlayerBar() {
  return (
    <div className="sticky top-0 z-50 border-b bg-white/80 backdrop-blur">
      <div className="mx-auto max-w-5xl px-3 h-14 flex items-center justify-between">
        <div className="min-w-0">
          <div className="text-sm font-medium truncate">MusicBox</div>
          <div className="text-xs text-gray-500 truncate">未连接</div>
        </div>
        <div className="flex items-center gap-2">
          <button
            className="px-3 py-1.5 rounded bg-gray-100 text-sm"
            type="button"
          >
            上一首
          </button>
          <button
            className="px-3 py-1.5 rounded bg-black text-white text-sm"
            type="button"
          >
            播放/暂停
          </button>
          <button
            className="px-3 py-1.5 rounded bg-gray-100 text-sm"
            type="button"
          >
            下一首
          </button>
        </div>
      </div>
    </div>
  );
}

