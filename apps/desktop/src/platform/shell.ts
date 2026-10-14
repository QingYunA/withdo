// Everything that only exists inside the mygo shell. In a plain browser each call degrades to
// something sensible (or does nothing), so the same UI runs in both.
import { currentWindow, isMyGo } from "mygo-runtime";
import { Shell, events } from "../mygo";
import { taskHash } from "../view/route";

/** Close the quick panel (after opening a task). Esc closes it by itself. No-op in a browser. */
export async function hidePanel(): Promise<void> {
  if (isMyGo()) await currentWindow.close();
}

/** Show a task in the main window; in a browser, navigate to it. */
export async function openTaskInMain(taskId: number): Promise<void> {
  if (isMyGo()) await Shell.openMain(taskHash(taskId));
  else window.location.hash = taskHash(taskId);
}

/** Open a URL in the default browser. mygo sends `target="_blank"` links there. */
export function openExternal(url: string): void {
  window.open(url, "_blank", "noopener,noreferrer");
}

/** Listen for "show this hash" requests from the Go shell (notification click, Dock icon). */
export function onMainNavigate(handler: (hash: string) => void): () => void {
  return isMyGo() ? events.mainNavigate.on(handler) : () => {};
}
