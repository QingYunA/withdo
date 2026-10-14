import { describe, expect, test } from "bun:test";
import { resolveDbPath } from "../src/paths.ts";

describe("resolveDbPath", () => {
  const home = "/home/u";

  test("ADHD_TODO_DB overrides everything", () => {
    expect(resolveDbPath({ env: { ADHD_TODO_DB: "/tmp/x.db" }, platform: "darwin", home })).toBe("/tmp/x.db");
  });

  test("macOS uses Application Support", () => {
    expect(resolveDbPath({ env: {}, platform: "darwin", home })).toBe(
      "/home/u/Library/Application Support/adhd-todo/adhd-todo.db",
    );
  });

  test("Linux honours XDG_DATA_HOME", () => {
    expect(resolveDbPath({ env: { XDG_DATA_HOME: "/xdg" }, platform: "linux", home })).toBe(
      "/xdg/adhd-todo/adhd-todo.db",
    );
  });

  test("Linux falls back to ~/.local/share", () => {
    expect(resolveDbPath({ env: {}, platform: "linux", home })).toBe("/home/u/.local/share/adhd-todo/adhd-todo.db");
  });

  test("Windows uses APPDATA", () => {
    expect(resolveDbPath({ env: { APPDATA: "C:\\Users\\u\\AppData\\Roaming" }, platform: "win32", home })).toBe(
      "C:\\Users\\u\\AppData\\Roaming\\adhd-todo\\adhd-todo.db",
    );
  });

  test("Windows without APPDATA falls back to home AppData/Roaming", () => {
    expect(resolveDbPath({ env: {}, platform: "win32", home: "C:\\Users\\u" })).toBe(
      "C:\\Users\\u\\AppData\\Roaming\\adhd-todo\\adhd-todo.db",
    );
  });

  test("empty override is ignored", () => {
    expect(resolveDbPath({ env: { ADHD_TODO_DB: "" }, platform: "linux", home })).toBe(
      "/home/u/.local/share/adhd-todo/adhd-todo.db",
    );
  });
});
