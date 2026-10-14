import tailwindcss from "@tailwindcss/vite";
import react from "@vitejs/plugin-react";
import { defineConfig } from "vitest/config";

// The core's CORS allow-list contains http://localhost:1420, so the dev server must stay on 1420.
export default defineConfig({
  plugins: [react(), tailwindcss()],
  clearScreen: false,
  server: { port: 1420, strictPort: true, host: "localhost" },
  build: { target: "safari15" },
  test: {
    environment: "jsdom",
    setupFiles: ["./src/test/setup.ts"],
    include: ["src/**/*.test.{ts,tsx}"],
    testTimeout: 20_000,
    hookTimeout: 30_000,
    coverage: {
      provider: "v8",
      include: ["src/core/**", "src/view/**"],
      exclude: ["src/**/*.test.*"],
      thresholds: { lines: 80 },
    },
  },
});
