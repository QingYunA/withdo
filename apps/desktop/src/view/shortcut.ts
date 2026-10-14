export interface KeyLike {
  readonly code: string;
  readonly altKey: boolean;
  readonly ctrlKey: boolean;
  readonly metaKey: boolean;
  readonly shiftKey: boolean;
}

const NAMED_KEYS: Readonly<Record<string, string>> = {
  Space: "Space",
  Enter: "Enter",
  Tab: "Tab",
  ArrowUp: "Up",
  ArrowDown: "Down",
  ArrowLeft: "Left",
  ArrowRight: "Right",
};

function keyName(code: string): string | null {
  const letter = /^Key([A-Z])$/.exec(code);
  if (letter) return letter[1]!;
  const digit = /^Digit(\d)$/.exec(code);
  if (digit) return digit[1]!;
  if (/^F([1-9]|1\d|2[0-4])$/.test(code)) return code;
  return NAMED_KEYS[code] ?? null;
}

/**
 * Turn a keydown into an accelerator string the Rust side can register, such as `Alt+Space`.
 * Needs at least one modifier and one non-modifier key; otherwise null.
 */
export function acceleratorFromEvent(event: KeyLike): string | null {
  const key = keyName(event.code);
  const modifiers = [
    event.metaKey && "Command",
    event.ctrlKey && "Control",
    event.altKey && "Alt",
    event.shiftKey && "Shift",
  ].filter((m): m is string => m !== false);
  if (!key || modifiers.length === 0) return null;
  return [...modifiers, key].join("+");
}

const DISPLAY: Readonly<Record<string, string>> = {
  Alt: "Option",
  Command: "Cmd",
  CommandOrControl: "Cmd",
  Control: "Ctrl",
};

/** `Alt+Space` as users know it on a Mac: `Option+Space`. */
export function displayShortcut(accelerator: string): string {
  return accelerator
    .split("+")
    .map((part) => DISPLAY[part] ?? part)
    .join("+");
}
