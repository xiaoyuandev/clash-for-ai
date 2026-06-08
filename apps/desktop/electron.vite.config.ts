import { resolve } from "path";
import { readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { defineConfig } from "electron-vite";
import react from "@vitejs/plugin-react";
import tailwindcss from "@tailwindcss/vite";
import type { Plugin, ViteDevServer } from "vite";

const workspaceRoot = resolve(fileURLToPath(new URL(".", import.meta.url)), "../..");
const localPresetRoutes = [
  {
    route: "/model-presets.json",
    sourcePath: resolve(workspaceRoot, "config/model-presets.json"),
    label: "model presets"
  },
  {
    route: "/provider-presets.json",
    sourcePath: resolve(workspaceRoot, "config/provider-presets.json"),
    label: "provider presets"
  }
];

function localPresetsPlugin(): Plugin {
  return {
    name: "local-presets",
    configureServer(server: ViteDevServer) {
      for (const item of localPresetRoutes) {
        server.middlewares.use(item.route, (req, res) => {
          if (req.method !== "GET" && req.method !== "HEAD") {
            res.statusCode = 405;
            res.end("method not allowed");
            return;
          }

          try {
            const content = readFileSync(item.sourcePath, "utf8");
            res.setHeader("Cache-Control", "no-cache");
            res.setHeader("Content-Type", "application/json; charset=utf-8");
            res.end(req.method === "HEAD" ? "" : content);
          } catch (error) {
            res.statusCode = 500;
            res.end(error instanceof Error ? error.message : `failed to read ${item.label}`);
          }
        });
      }
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
    plugins: [react(), tailwindcss(), localPresetsPlugin()]
  }
});
