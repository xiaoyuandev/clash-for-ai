import { defineConfig, type Plugin, type ViteDevServer } from "vite";
import react from "@vitejs/plugin-react";
import tailwindcss from "@tailwindcss/vite";
import { VitePWA } from "vite-plugin-pwa";
import { readFileSync } from "node:fs";
import { resolve } from "node:path";
import { fileURLToPath } from "node:url";

const workspaceRoot = resolve(fileURLToPath(new URL(".", import.meta.url)), "../..");
const localModelPresetsPath = resolve(workspaceRoot, "config/model-presets.json");

function localModelPresetsPlugin(): Plugin {
  return {
    name: "local-model-presets",
    configureServer(server: ViteDevServer) {
      server.middlewares.use("/model-presets.json", (req, res) => {
        if (req.method !== "GET" && req.method !== "HEAD") {
          res.statusCode = 405;
          res.end("method not allowed");
          return;
        }

        try {
          const content = readFileSync(localModelPresetsPath, "utf8");
          res.setHeader("Cache-Control", "no-cache");
          res.setHeader("Content-Type", "application/json; charset=utf-8");
          res.end(req.method === "HEAD" ? "" : content);
        } catch (error) {
          res.statusCode = 500;
          res.end(error instanceof Error ? error.message : "failed to read model presets");
        }
      });
    }
  };
}

export default defineConfig({
  plugins: [
    react(),
    tailwindcss(),
    localModelPresetsPlugin(),
    VitePWA({
      registerType: "autoUpdate",
      includeAssets: ["icons/icon-512.png"],
      manifest: {
        name: "Relay Switch Web",
        short_name: "Relay Switch",
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
