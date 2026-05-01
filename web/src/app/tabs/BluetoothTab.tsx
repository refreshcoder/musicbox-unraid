import { useEffect, useMemo, useState } from "react";
import { getJSON, postJSON } from "../../lib/api";

type BtStatus = {
  scanning: boolean;
  defaultMac: string;
};

type BtDevice = {
  mac: string;
  name: string;
};

type BtDevicesResp = {
  devices: BtDevice[];
};

export function BluetoothTab() {
  const [status, setStatus] = useState<BtStatus | null>(null);
  const [devices, setDevices] = useState<BtDevice[]>([]);
  const [error, setError] = useState<string | null>(null);

  const refresh = () => {
    setError(null);
    Promise.all([
      getJSON<BtStatus>("/api/v1/bluetooth/status"),
      getJSON<BtDevicesResp>("/api/v1/bluetooth/devices"),
    ])
      .then(([st, ds]) => {
        setStatus(st);
        setDevices(ds.devices ?? []);
      })
      .catch((e: unknown) => setError(String(e)));
  };

  useEffect(() => {
    refresh();
  }, []);

  const defaultMac = status?.defaultMac ?? "";
  const isScanning = status?.scanning ?? false;

  const sorted = useMemo(() => {
    const copy = [...devices];
    copy.sort((a, b) => a.name.localeCompare(b.name));
    return copy;
  }, [devices]);

  return (
    <div className="space-y-3">
      <div className="flex items-center justify-between">
        <div>
          <div className="text-sm font-medium">蓝牙</div>
          <div className="text-xs text-gray-500">
            默认设备：{defaultMac ? defaultMac : "未设置"}
          </div>
        </div>
        <div className="flex items-center gap-2">
          <button
            className="px-3 py-1.5 rounded bg-gray-100 text-sm"
            type="button"
            onClick={refresh}
          >
            刷新
          </button>
          <button
            className={[
              "px-3 py-1.5 rounded text-sm",
              isScanning ? "bg-gray-900 text-white" : "bg-gray-100",
            ].join(" ")}
            type="button"
            onClick={() =>
              postJSON(isScanning ? "/api/v1/bluetooth/scan/stop" : "/api/v1/bluetooth/scan/start")
                .then(() => refresh())
                .catch(() => refresh())
            }
          >
            {isScanning ? "停止扫描" : "开始扫描"}
          </button>
        </div>
      </div>

      {error ? (
        <div className="rounded border bg-red-50 text-red-700 text-sm p-3">
          {error}
        </div>
      ) : null}

      <div className="rounded border bg-white">
        {sorted.length === 0 ? (
          <div className="p-4 text-sm text-gray-500">未发现设备</div>
        ) : (
          <ul className="divide-y">
            {sorted.map((d) => (
              <li key={d.mac} className="p-3 flex items-center gap-3">
                <div className="min-w-0 flex-1">
                  <div className="text-sm truncate">{d.name}</div>
                  <div className="text-xs text-gray-500 truncate">{d.mac}</div>
                </div>
                <button
                  className={[
                    "px-2 py-1 rounded text-xs",
                    defaultMac === d.mac ? "bg-black text-white" : "bg-gray-100",
                  ].join(" ")}
                  type="button"
                  onClick={() =>
                    postJSON("/api/v1/bluetooth/default", { mac: d.mac })
                      .then(() => refresh())
                      .catch(() => refresh())
                  }
                >
                  设为默认
                </button>
                <button
                  className="px-2 py-1 rounded bg-gray-100 text-xs"
                  type="button"
                  onClick={() =>
                    postJSON(`/api/v1/bluetooth/devices/${encodeURIComponent(d.mac)}/connect`)
                      .then(() => refresh())
                      .catch(() => refresh())
                  }
                >
                  连接
                </button>
                <button
                  className="px-2 py-1 rounded bg-gray-100 text-xs"
                  type="button"
                  onClick={() =>
                    postJSON(`/api/v1/bluetooth/devices/${encodeURIComponent(d.mac)}/disconnect`)
                      .then(() => refresh())
                      .catch(() => refresh())
                  }
                >
                  断开
                </button>
              </li>
            ))}
          </ul>
        )}
      </div>
    </div>
  );
}

