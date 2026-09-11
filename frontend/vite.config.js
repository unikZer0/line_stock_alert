import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

export default defineConfig({
  plugins: [react()],
  server: {
    host: "0.0.0.0",
    port: 5173,
    allowedHosts: ["since-kid.site", "www.since-kid.site"],
    proxy: {
      "/api": {
        target: "http://backend:8080",
        changeOrigin: true,
      },
    },
  },
});
