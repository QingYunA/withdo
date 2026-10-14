import { useEffect, useState } from "react";
import { getShortcut, setShortcut } from "../platform/shell";
import { acceleratorFromEvent, displayShortcut } from "../view/shortcut";

type Current = { readonly kind: "loading" } | { readonly kind: "unavailable" } | { readonly kind: "set"; readonly value: string };

const describe = (error: unknown): string => (error instanceof Error ? error.message : String(error));

export function SettingsView() {
  const [current, setCurrent] = useState<Current>({ kind: "loading" });
  const [recording, setRecording] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    void getShortcut().then((value) => setCurrent(value === null ? { kind: "unavailable" } : { kind: "set", value }));
  }, []);

  useEffect(() => {
    if (!recording) return;
    const onKey = (event: KeyboardEvent) => {
      event.preventDefault();
      if (event.key === "Escape") return setRecording(false);
      const accelerator = acceleratorFromEvent(event);
      if (!accelerator) return;
      setRecording(false);
      setError(null);
      setShortcut(accelerator).then(
        (value) => setCurrent({ kind: "set", value }),
        (failure: unknown) => setError(describe(failure)),
      );
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [recording]);

  return (
    <div className="max-w-xl p-6">
      <h1 className="mb-4 text-base font-semibold">Settings</h1>
      <h2 className="label">Global shortcut</h2>
      <p className="mb-2 text-xs text-muted">Opens the quick panel from any app, including full-screen ones.</p>
      {current.kind === "unavailable" ? (
        <p className="text-xs text-muted">The shortcut is only available in the desktop app.</p>
      ) : (
        <div className="flex items-center gap-3">
          <span className="code">{current.kind === "set" ? displayShortcut(current.value) : "..."}</span>
          {recording ? (
            <span className="text-xs text-muted">Press the new shortcut, or Esc to cancel</span>
          ) : (
            <button type="button" className="btn" onClick={() => setRecording(true)} disabled={current.kind !== "set"}>
              Change shortcut
            </button>
          )}
        </div>
      )}
      {error ? <p className="mt-2 text-xs text-danger">{error}</p> : null}
    </div>
  );
}
