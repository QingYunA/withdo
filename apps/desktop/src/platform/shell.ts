// Everything that only exists inside the Tauri shell. In a plain browser each call degrades to
// something sensible (or does nothing), so the same UI runs in both.
import { isTauri } from "../core/endpoint";
import { taskHash } from "../view/route";

async function invoke<T>(command: string, args?: Record<string, unknown>): Promise<T> {
  const { invoke: tauriInvoke } = await import("@tauri-apps/api/core");
  return tauriInvoke<T>(command, args);
}

/** Hide the floating panel (Esc, or after opening a task). No-op in a browser. */
export async function hidePanel(): Promise<void> {
  if (isTauri()) await invoke("hide_panel");
}

/** Show a task in the main window; in a browser, navigate to it. */
export async function openTaskInMain(taskId: number): Promise<void> {
  if (isTauri()) await invoke("open_main", { hash: taskHash(taskId) });
  else window.location.hash = taskHash(taskId);
}

export async function openExternal(url: string): Promise<void> {
  if (isTauri()) {
    const { openUrl } = await import("@tauri-apps/plugin-opener");
    await openUrl(url);
  } else {
    window.open(url, "_blank", "noopener,noreferrer");
  }
}

/** The configured global shortcut, or null when not running inside the desktop app. */
export async function getShortcut(): Promise<string | null> {
  return isTauri() ? invoke<string>("get_shortcut") : null;
}

/** Register and persist a new global shortcut. Rejects with a readable message if it is not usable. */
export async function setShortcut(shortcut: string): Promise<string> {
  return invoke<string>("set_shortcut", { shortcut });
}

export type ShellEvent = "panel-shown" | "main-navigate";

/** Listen for an event emitted by the Rust shell; returns the unsubscribe function. */
export function onShellEvent(name: ShellEvent, handler: (payload: unknown) => void): () => void {
  if (!isTauri()) return () => {};
  let unlisten: (() => void) | null = null;
  let cancelled = false;
  void import("@tauri-apps/api/event").then(({ listen }) =>
    listen(name, (event) => handler(event.payload)).then((fn) => {
      if (cancelled) fn();
      else unlisten = fn;
    }),
  );
  return () => {
    cancelled = true;
    unlisten?.();
  };
}
