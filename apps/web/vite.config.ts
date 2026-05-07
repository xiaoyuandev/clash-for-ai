import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import tailwindcss from "@tailwindcss/vite";
import { VitePWA } from "vite-plugin-pwa";

export default defineConfig({
  plugins: [
    react(),
    tailwindcss(),
    VitePWA({
      registerType: "autoUpdate",
      includeAssets: ["icons/icon-512.png"],
      manifest: {
        name: "Clash for AI Web",
        short_name: "Clash for AI",
        description: "Supplementary web management UI for WSL and Linux server.",
        theme_color: "#14b8a6",
        background_color: "#f8fbfd",
        display: "standalone",
        start_url: "/",
        icons: [
          {
            src: "/icons/icon-512.png",
            sizes: "512x512",
            type: "image/png"
          }
        ]
      }
    })
  ],
  server: {
    host: "0.0.0.0",
    port: 4173
  }
});
