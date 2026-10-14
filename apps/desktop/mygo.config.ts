import { defineConfig } from "mygo-cli";

// Vite stays on port 1420: the core's CORS allow-list contains http://localhost:1420.
export default defineConfig({
  name: "ADHD Todo",
  identifier: "dev.adhdtodo.desktop",
  version: "0.1.0",
  devUrl: "http://localhost:1420",
  // The sidecar lands in resources/<platform>/bin before the app bundle is assembled.
  devCommand: "bun run sidecar && bun run dev:web",
  buildCommand: "bun run sidecar && bun run build:web",
  frontendDist: "dist",
  bindings: "src/mygo.ts",
  out: "build",
});
