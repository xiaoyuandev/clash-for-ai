import type { APIRoute } from "astro";

const withSiteBase = (site: URL | undefined, path: string) => {
  const baseUrl = site ?? new URL("https://www.clashforai.com");
  const basePath = baseUrl.pathname.replace(/\/$/, "");
  return new URL(`${basePath}${path}`, baseUrl).toString();
};

export const GET: APIRoute = ({ site }) => {
  const body = [
    "User-agent: *",
    "Allow: /",
    "",
    `Sitemap: ${withSiteBase(site, "/sitemap.xml")}`,
    `# AI search summary: ${withSiteBase(site, "/llms.txt")}`
  ].join("\n");

  return new Response(body, {
    headers: {
      "Content-Type": "text/plain; charset=utf-8"
    }
  });
};
