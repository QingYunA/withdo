import { homedir } from "node:os";
import { posix, win32 } from "node:path";

export interface PathOptions {
  readonly env?: Readonly<Record<string, string | undefined>>;
  readonly platform?: NodeJS.Platform;
  readonly home?: string;
}

const APP_DIR = "adhd-todo";
const DB_FILE = "adhd-todo.db";

/** Resolve the SQLite database path for the current OS, honouring ADHD_TODO_DB. */
export function resolveDbPath(options: PathOptions = {}): string {
  const env = options.env ?? process.env;
  const platform = options.platform ?? process.platform;
  const home = options.home ?? homedir();

  const override = env.ADHD_TODO_DB;
  if (override) return override;

  if (platform === "win32") {
    const base = env.APPDATA || win32.join(home, "AppData", "Roaming");
    return win32.join(base, APP_DIR, DB_FILE);
  }
  if (platform === "darwin") {
    return posix.join(home, "Library", "Application Support", APP_DIR, DB_FILE);
  }
  const base = env.XDG_DATA_HOME || posix.join(home, ".local", "share");
  return posix.join(base, APP_DIR, DB_FILE);
}
