export type WsEvent<T = unknown> = {
  type: string;
  ts: number;
  data: T;
};

export function connectWs(onEvent: (evt: WsEvent) => void): WebSocket {
  const proto = location.protocol === "https:" ? "wss" : "ws";
  const ws = new WebSocket(`${proto}://${location.host}/ws`);
  ws.onmessage = (msg) => {
    try {
      onEvent(JSON.parse(msg.data) as WsEvent);
    } catch {
      return;
    }
  };
  return ws;
}

