import { describe, expect, test } from "vitest";
import { acceleratorFromEvent, displayShortcut } from "./shortcut";

const ev = (over: Partial<Parameters<typeof acceleratorFromEvent>[0]>) => ({
  code: "KeyK",
  altKey: false,
  ctrlKey: false,
  metaKey: false,
  shiftKey: false,
  ...over,
});

describe("acceleratorFromEvent", () => {
  test("builds Alt+Space", () => {
    expect(acceleratorFromEvent(ev({ code: "Space", altKey: true }))).toBe("Alt+Space");
  });

  test("orders modifiers consistently and names letters, digits, arrows and function keys", () => {
    expect(acceleratorFromEvent(ev({ code: "KeyK", metaKey: true, shiftKey: true }))).toBe("Command+Shift+K");
    expect(acceleratorFromEvent(ev({ code: "Digit3", ctrlKey: true, altKey: true }))).toBe("Control+Alt+3");
    expect(acceleratorFromEvent(ev({ code: "ArrowUp", altKey: true }))).toBe("Alt+Up");
    expect(acceleratorFromEvent(ev({ code: "F5", metaKey: true }))).toBe("Command+F5");
  });

  test("needs a modifier and a real key", () => {
    expect(acceleratorFromEvent(ev({ code: "KeyK" }))).toBeNull();
    expect(acceleratorFromEvent(ev({ code: "ShiftLeft", shiftKey: true }))).toBeNull();
    expect(acceleratorFromEvent(ev({ code: "Escape", altKey: true }))).toBeNull();
  });
});

describe("displayShortcut", () => {
  test("uses macOS names", () => {
    expect(displayShortcut("Alt+Space")).toBe("Option+Space");
    expect(displayShortcut("Command+Shift+K")).toBe("Cmd+Shift+K");
    expect(displayShortcut("Control+Alt+3")).toBe("Ctrl+Option+3");
  });
});
