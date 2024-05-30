import { defineConfig } from "vitepress";

// https://vitepress.dev/reference/site-config
export default defineConfig({
  base: "/seelf/",
  ignoreDeadLinks: "localhostLinks",
  title: "seelf documentation",
  head: [["link", { rel: "icon", href: "/seelf/favicon.svg" }]],
  description: "Lightweight self-hosted deployment platform written in Go.",
  themeConfig: {
    siteTitle: false,
    search: {
      provider: "local",
    },
    logo: {
      light: "/logo-light.svg",
      dark: "/logo-dark.svg",
      alt: "seelf",
    },
    // https://vitepress.dev/reference/default-theme-config
    nav: [
      { text: "Home", link: "/" },
      { text: "Docs", link: "/guide/quickstart" },
    ],

    sidebar: [
      {
        text: "Guide",
        items: [
          { text: "Quickstart", link: "/guide/quickstart" },
          { text: "Installation", link: "/guide/installation" },
          { text: "Updating", link: "/guide/updating" },
          { text: "Configuration", link: "/guide/configuration" },
          { text: "Migrating major versions", link: "/guide/migration" },
          {
            text: "Continuous Integration / Deployment",
            link: "/guide/continuous-integration-deployment",
          },
        ],
      },
      {
        text: "Reference",
        items: [
          {
            text: "Targets",
            link: "/reference/targets",
          },
          {
            text: "Providers",
            link: "/reference/providers",
            items: [
              {
                text: "Docker",
                link: "/reference/providers/docker",
              },
            ],
          },
          {
            text: "Registries",
            link: "/reference/registries",
          },
          {
            text: "Applications",
            link: "/reference/applications",
          },
          {
            text: "Deployments",
            link: "/reference/deployments",
          },
          {
            text: "Sources",
            link: "/reference/deployments#sources",
          },
          {
            text: "Jobs",
            link: "/reference/jobs",
          },
          {
            text: "API",
            link: "/reference/api",
          },
          {
            text: "FAQ",
            link: "/reference/faq",
          },
        ],
      },
      {
        text: "Contributing",
        items: [
          { text: "Docs", link: "/contributing/docs" },
          { text: "Backend", link: "/contributing/backend" },
          { text: "Frontend", link: "/contributing/frontend" },
          { text: "Donating", link: "/contributing/donating" },
        ],
      },
    ],

    socialLinks: [
      { icon: "github", link: "https://github.com/YuukanOO/seelf" },
    ],
  },
});
