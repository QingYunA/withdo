import { expect, test } from "bun:test";
import { join } from "node:path";
import { useTmpDir } from "./helpers/tmp.ts";

const tmp = useTmpDir();

type Reader = { read(): Promise<{ done: boolean; value?: Uint8Array }> };
const cli = join(import.meta.dir, "..", "src", "cli.ts");

/** Read stdout until the first newline; keep the reader so the rest can be drained later. */
async function firstLine(reader: Reader): Promise<{ line: string; rest: string }> {
  const decoder = new TextDecoder();
  let buffer = "";
  while (!buffer.includes("\n")) {
    const { done, value } = await reader.read();
    if (done) break;
    buffer += decoder.decode(value, { stream: true });
  }
  const idx = buffer.indexOf("\n");
  return { line: buffer.slice(0, idx), rest: buffer.slice(idx + 1) };
}

async function drain(reader: Reader): Promise<string> {
  const decoder = new TextDecoder();
  let text = "";
  for (;;) {
    const { done, value } = await reader.read();
    if (done) return text;
    text += decoder.decode(value, { stream: true });
  }
}

test("`cli.ts serve` prints only the ready line on stdout and exits when stdin closes", async () => {
  const proc = Bun.spawn([process.execPath, cli, "serve"], {
    env: { PATH: process.env.PATH ?? "", ADHD_TODO_DB: join(tmp(), "proc.db") },
    stdin: "pipe",
    stdout: "pipe",
    stderr: "pipe",
  });
  const stdout = proc.stdout.getReader();
  const { line, rest } = await firstLine(stdout);
  const ready = JSON.parse(line) as { port: number; token: string };
  expect(Object.keys(ready).sort()).toEqual(["port", "token"]);

  const res = await fetch(`http://127.0.0.1:${ready.port}/overview`, {
    headers: { authorization: `Bearer ${ready.token}` },
  });
  expect(res.status).toBe(200);

  proc.stdin.end();
  expect(await proc.exited).toBe(0);
  expect(rest + (await drain(stdout))).toBe("");
  const stderr = await new Response(proc.stderr).text();
  expect(stderr).toContain(`listening on 127.0.0.1:${ready.port}`);
  expect(stderr).not.toContain(ready.token);
});
