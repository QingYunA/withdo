// Subprocess helper: argv = [dbPath, tag, count]. Inserts `count` projects.
import { openDb } from "../../src/db/open.ts";

const [dbPath, tag, count] = process.argv.slice(2);
const db = openDb(dbPath!);
const n = Number(count);
const stmt = db.prepare("INSERT INTO projects (name, created_at, updated_at) VALUES (?, ?, ?)");
for (let i = 0; i < n; i++) {
  const ts = new Date().toISOString();
  stmt.run(`${tag}-${i}`, ts, ts);
}
db.close();
