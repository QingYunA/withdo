import type { KeyboardEvent, RefObject } from "react";

export interface QuickAddProps {
  readonly value: string;
  readonly onChange: (value: string) => void;
  readonly onEnter: () => void;
  readonly inputRef: RefObject<HTMLInputElement | null>;
  readonly notice: string | null;
  readonly error: string | null;
  readonly busy: boolean;
}

export function QuickAdd({ value, onChange, onEnter, inputRef, notice, error, busy }: QuickAddProps) {
  const onKeyDown = (event: KeyboardEvent<HTMLInputElement>) => {
    // Enter confirms an IME candidate (e.g. Chinese input); only a plain Enter submits.
    if (event.key === "Enter" && !event.nativeEvent.isComposing) {
      event.preventDefault();
      onEnter();
    }
  };
  return (
    <div className="border-b border-border px-4 pt-3 pb-2">
      <input
        ref={inputRef}
        type="text"
        autoFocus
        autoComplete="off"
        spellCheck={false}
        aria-label="New task"
        placeholder="Add a task. The AI assistant is coming soon."
        className="w-full bg-transparent text-[15px] text-fg outline-none placeholder:text-muted"
        value={value}
        disabled={busy}
        onChange={(event) => onChange(event.target.value)}
        onKeyDown={onKeyDown}
      />
      <p className="h-4 text-xs" role="status">
        {error ? <span className="text-danger">{error}</span> : notice ? <span className="text-muted">{notice}</span> : null}
      </p>
    </div>
  );
}
