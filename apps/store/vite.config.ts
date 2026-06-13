import path from "path"
import tailwindcss from "@tailwindcss/vite"
import react from "@vitejs/plugin-react"
import { defineConfig } from "vite"

export default defineConfig({
  plugins: [react(), tailwindcss()],
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "./src"),
    },
  },
  // Dev-only proxy. The Vite dev server runs the store at :5173 and
  // forwards /api and /uploads to the local Go backend at :8000. In a
  // production build these paths are served by `VITE_API_URL` and
  // absolute image URLs in the DB, so the proxy target never affects
  // shipping. Keep it on localhost — pointing at api.mioru.store here
  // would make every local `npm run dev` hit production and risk
  // double-counting orders/stock during development (priority #1).
  server: {
    host: "0.0.0.0",
    port: 5173,
    proxy: {
      "/api": {
        target: "http://localhost:8000",
        changeOrigin: true,
        secure: false,
      },
      "/uploads": {
        target: "http://localhost:8000",
        changeOrigin: true,
        secure: false,
      },
    },
  },
})
