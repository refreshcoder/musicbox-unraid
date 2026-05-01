export async function getJSON<T>(path: string): Promise<T> {
  const resp = await fetch(path, { headers: { Accept: "application/json" } });
  if (!resp.ok) {
    throw new Error(`HTTP ${resp.status}`);
  }
  return (await resp.json()) as T;
}

export async function postJSON<T>(path: string, body?: unknown): Promise<T> {
  const resp = await fetch(path, {
    method: "POST",
    headers: {
      Accept: "application/json",
      "Content-Type": "application/json",
    },
    body: body === undefined ? "{}" : JSON.stringify(body),
  });
  if (!resp.ok) {
    throw new Error(`HTTP ${resp.status}`);
  }
  return (await resp.json()) as T;
}

