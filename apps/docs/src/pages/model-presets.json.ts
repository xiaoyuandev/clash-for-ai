import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import type { APIRoute } from "astro";
import { readModelPresetCatalog } from "@relay-switch/model-presets/node";

export const prerender = true;

const sourcePath = resolve(dirname(fileURLToPath(import.meta.url)), "../../../../config/model-presets.json");

export const GET: APIRoute = () => {
  const { catalog } = readModelPresetCatalog(sourcePath);

  return new Response(`${JSON.stringify(catalog, null, 2)}\n`, {
    headers: {
      "Cache-Control": "public, max-age=300",
      "Content-Type": "application/json; charset=utf-8"
    }
  });
};
