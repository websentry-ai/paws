import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

// The bundle is committed to dist/ and embedded in the Go binary, so the build must be
// self-contained (no CDN) and byte-stable for the CI check that rebuilds it.
export default defineConfig({
  plugins: [react()],
  base: "./",
  build: {
    outDir: "dist",
    emptyOutDir: true,
    modulePreload: { polyfill: false },
  },
  server: {
    // `npm run dev` against a running `paws ui --port 4242 --no-open`.
    proxy: { "/api": { target: "http://127.0.0.1:4242", changeOrigin: true } },
  },
});
