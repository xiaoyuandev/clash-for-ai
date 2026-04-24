import { defineConfig } from "astro/config";
import starlight from "@astrojs/starlight";
import tailwindcss from "@tailwindcss/vite";

const repository = process.env.GITHUB_REPOSITORY?.split("/")[1] ?? "clash-for-ai";
const site = process.env.DOCS_SITE_URL ?? `https://xiaoyuandev.github.io/${repository}`;
const siteURL = new URL(site);
const normalizedBase =
  siteURL.pathname && siteURL.pathname !== "/"
    ? siteURL.pathname.replace(/\/$/, "")
    : undefined;

export default defineConfig({
  site,
  base: normalizedBase,
  vite: {
    plugins: [tailwindcss()]
  },
  integrations: [
    starlight({
      title: "Clash for AI Docs",
      description: "Documentation for Clash for AI, a local desktop gateway for switching AI relay providers behind one stable endpoint.",
      customCss: ["/src/styles/site.css"],
      defaultLocale: "root",
      locales: {
        root: {
          label: "English",
          lang: "en"
        },
        "zh-cn": {
          label: "简体中文",
          lang: "zh-CN"
        }
      },
      social: [
        {
          icon: "github",
          label: "GitHub",
          href: "https://github.com/xiaoyuandev/clash-for-ai"
        }
      ],
      sidebar: [
        {
          label: "Get Started",
          items: [
            { slug: "introduction" },
            { slug: "quick-start" },
            { slug: "user-guide" },
            { slug: "tool-integration" }
          ]
        },
        {
          label: "Reference",
          items: [{ slug: "providers" }, { slug: "faq" }]
        }
      ]
    })
  ]
});
