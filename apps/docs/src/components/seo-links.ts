const pairedSlugs = new Set([
  "introduction",
  "quick-start",
  "user-guide",
  "tool-integration",
  "deep-link-import",
  "providers",
  "faq"
]);

const normalizePath = (pathname: string) => {
  const normalized = pathname.endsWith("/") ? pathname : `${pathname}/`;
  return normalized.startsWith("/") ? normalized : `/${normalized}`;
};

export const withSitePath = (site: URL | undefined, path: string) => {
  const baseUrl = site ?? new URL("https://www.relayswitch.dev");
  const basePath = baseUrl.pathname.replace(/\/$/, "");
  return new URL(`${basePath}${normalizePath(path)}`, baseUrl).toString();
};

export const getAlternateLinks = (site: URL | undefined, pathname: string) => {
  const path = normalizePath(pathname);
  const withoutChinesePrefix = path.replace(/^\/zh-cn\//, "/");
  const slug = withoutChinesePrefix.replace(/^\/|\/$/g, "");

  if (path === "/" || path === "/zh-cn/") {
    return [
      { hreflang: "en", href: withSitePath(site, "/") },
      { hreflang: "zh-CN", href: withSitePath(site, "/zh-cn/") },
      { hreflang: "x-default", href: withSitePath(site, "/") }
    ];
  }

  if (!pairedSlugs.has(slug)) {
    return [];
  }

  return [
    { hreflang: "en", href: withSitePath(site, `/${slug}/`) },
    { hreflang: "zh-CN", href: withSitePath(site, `/zh-cn/${slug}/`) },
    { hreflang: "x-default", href: withSitePath(site, `/${slug}/`) }
  ];
};
