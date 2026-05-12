import type { APIRoute } from "astro";

const routes = [
  "/",
  "/introduction/",
  "/quick-start/",
  "/user-guide/",
  "/tool-integration/",
  "/deep-link-import/",
  "/providers/",
  "/faq/",
  "/zh-cn/",
  "/zh-cn/introduction/",
  "/zh-cn/quick-start/",
  "/zh-cn/user-guide/",
  "/zh-cn/tool-integration/",
  "/zh-cn/deep-link-import/",
  "/zh-cn/providers/",
  "/zh-cn/faq/"
];

const withSiteBase = (site: URL | undefined, path: string) => {
  const baseUrl = site ?? new URL("https://www.clashforai.com");
  const basePath = baseUrl.pathname.replace(/\/$/, "");
  return new URL(`${basePath}${path}`, baseUrl).toString();
};

export const GET: APIRoute = ({ site }) => {
  const now = new Date().toISOString();
  const urls = routes
    .map((route) => {
      const loc = withSiteBase(site, route);
      const priority = route === "/" || route === "/zh-cn/" ? "1.0" : "0.7";
      return [
        "  <url>",
        `    <loc>${loc}</loc>`,
        `    <lastmod>${now}</lastmod>`,
        "    <changefreq>weekly</changefreq>",
        `    <priority>${priority}</priority>`,
        "  </url>"
      ].join("\n");
    })
    .join("\n");

  return new Response(
    `<?xml version="1.0" encoding="UTF-8"?>\n<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">\n${urls}\n</urlset>\n`,
    {
      headers: {
        "Content-Type": "application/xml; charset=utf-8"
      }
    }
  );
};
