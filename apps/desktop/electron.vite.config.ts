import { resolve } from "path";
import { readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { defineConfig } from "electron-vite";
import react from "@vitejs/plugin-react";
import tailwindcss from "@tailwindcss/vite";
import type { Plugin, ViteDevServer } from "vite";

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
  main: {},
  preload: {},
  renderer: {
    resolve: {
      alias: {
        "@renderer": resolve("src/renderer/src")
      }
    },
    plugins: [react(), tailwindcss(), localModelPresetsPlugin()]
  }
});
