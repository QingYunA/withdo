export interface SseEvent {
  readonly event: string;
  readonly data: any;
}

export interface SseClient {
  readonly status: number;
  readonly contentType: string | null;
  readonly events: readonly SseEvent[];
  readonly comments: readonly string[];
  /** Resolve with the first event (from now on or already seen) matching `predicate`. */
  waitFor(predicate: (e: SseEvent) => boolean, timeoutMs?: number): Promise<SseEvent>;
  close(): void;
}

const sleep = (ms: number) => new Promise((r) => setTimeout(r, ms));

function parseBlock(block: string, events: SseEvent[], comments: string[]): void {
  const lines = block.split("\n");
  const dataLines: string[] = [];
  let event = "message";
  for (const line of lines) {
    if (line.startsWith(":")) comments.push(line.slice(1).trim());
    else if (line.startsWith("event:")) event = line.slice(6).trim();
    else if (line.startsWith("data:")) dataLines.push(line.slice(5).trim());
  }
  if (dataLines.length > 0) events.push({ event, data: JSON.parse(dataLines.join("\n")) });
}

/** Open an SSE stream and collect events in the background. Resolves once headers and the first chunk arrived. */
export async function openSse(url: string, headers: Record<string, string> = {}): Promise<SseClient> {
  const abort = new AbortController();
  const res = await fetch(url, { headers, signal: abort.signal });
  const events: SseEvent[] = [];
  const comments: string[] = [];
  if (res.status === 200 && res.body) {
    const reader = res.body.getReader();
    const decoder = new TextDecoder();
    let buffer = "";
    void (async () => {
      try {
        for (;;) {
          const { done, value } = await reader.read();
          if (done) return;
          buffer += decoder.decode(value, { stream: true });
          let idx: number;
          while ((idx = buffer.indexOf("\n\n")) >= 0) {
            parseBlock(buffer.slice(0, idx), events, comments);
            buffer = buffer.slice(idx + 2);
          }
        }
      } catch {
        // aborted
      }
    })();
  } else {
    await res.body?.cancel();
  }
  return {
    status: res.status,
    contentType: res.headers.get("content-type"),
    events,
    comments,
    async waitFor(predicate, timeoutMs = 3000) {
      const deadline = Date.now() + timeoutMs;
      while (Date.now() < deadline) {
        const found = events.find(predicate);
        if (found) return found;
        await sleep(10);
      }
      throw new Error(`timed out waiting for event; saw ${JSON.stringify(events)}`);
    },
    close: () => abort.abort(),
  };
}

export { sleep };
