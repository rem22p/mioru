import path from "path";
import tailwindcss from "@tailwindcss/vite";
import react from "@vitejs/plugin-react";
import { defineConfig } from "vitest/config";

export default defineConfig({
  plugins: [react(), tailwindcss()],
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "./src"),
    },
  },
  server: {
    host: "0.0.0.0",
    port: 5174,
    proxy: {
      "/api": {
        target: "http://localhost:8000",
        changeOrigin: true,
      },
      "/uploads": {
        target: "http://localhost:8000",
        changeOrigin: true,
      },
      "/ws": {
        target: "ws://localhost:8000",
        ws: true,
      },
    },
  },
  test: {
    // Force Vite to transform React / ReactDOM through the dev pipeline so the
    // *development* react-dom-test-utils bundle is loaded — the production
    // bundle in React 19 still references `React.act(...)` which is no longer
    // exposed on the namespace and crashes @testing-library/react@16.
    server: {
      deps: {
        inline: [/react/, /react-dom/],
      },
    },
  },
});
