import type { APIRoute } from "astro";
import { getCollection } from "astro:content";
import { getAlternateLinks, withSitePath } from "../components/seo-links";

const toRoute = (slug: string) => `/${slug.replace(/^\/|\/$/g, "")}/`;

const escapeXml = (value: string) =>
  value
    .replaceAll("&", "&amp;")
    .replaceAll('"', "&quot;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;");

const getRoutes = async () => {
  const docs = await getCollection("docs", ({ data }) => import.meta.env.MODE !== "production" || data.draft === false);
  const routes = new Set(["/", "/zh-cn/"]);

  for (const entry of docs) {
    routes.add(toRoute(entry.slug ?? entry.id));
  }

  return [...routes].sort((a, b) => {
    if (a === "/") return -1;
    if (b === "/") return 1;
    if (a === "/zh-cn/") return b.startsWith("/zh-cn/") ? -1 : 1;
    if (b === "/zh-cn/") return a.startsWith("/zh-cn/") ? 1 : -1;
    return a.localeCompare(b);
  });
};

export const GET: APIRoute = async ({ site }) => {
  const now = new Date().toISOString();
  const routes = await getRoutes();
  const urls = routes
    .map((route) => {
      const loc = withSitePath(site, route);
      const priority = route === "/" || route === "/zh-cn/" ? "1.0" : "0.7";
      const alternateLinks = getAlternateLinks(site, route)
        .map(
          (link) =>
            `    <xhtml:link rel="alternate" hreflang="${escapeXml(link.hreflang)}" href="${escapeXml(link.href)}" />`
        )
        .join("\n");
      return [
        "  <url>",
        `    <loc>${escapeXml(loc)}</loc>`,
        `    <lastmod>${now}</lastmod>`,
        "    <changefreq>weekly</changefreq>",
        `    <priority>${priority}</priority>`,
        alternateLinks,
        "  </url>"
      ].filter(Boolean).join("\n");
    })
    .join("\n");

  return new Response(
    `<?xml version="1.0" encoding="UTF-8"?>\n<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9" xmlns:xhtml="http://www.w3.org/1999/xhtml">\n${urls}\n</urlset>\n`,
    {
      headers: {
        "Content-Type": "application/xml; charset=utf-8"
      }
    }
  );
};
