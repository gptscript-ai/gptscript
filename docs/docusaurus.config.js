// @ts-check
// `@type` JSDoc annotations allow editor autocompletion and type checking
// (when paired with `@ts-check`).
// There are various equivalent ways to declare your Docusaurus config.
// See: https://docusaurus.io/docs/api/docusaurus-config

import { themes as prismThemes } from "prism-react-renderer";

/** @type {import('@docusaurus/types').Config} */
const config = {
  title: "GPTScript Docs",
  tagline: "Welcome to the GPTScript Docs",
  favicon: "img/favicon.ico",
  baseUrl: "/",
  url: "https://docs.gptscript.ai",
  organizationName: "gptscript-ai",
  projectName: "gptscript",
  onBrokenLinks: "throw",
  onBrokenMarkdownLinks: "warn",
  trailingSlash: false,

  i18n: {
    defaultLocale: "en",
    locales: ["en"],
  },

  presets: [
    [
      "classic",
      /** @type {import('@docusaurus/preset-classic').Options} */
      ({
        docs: {
          sidebarPath: "./sidebars.js",
          editUrl: "https://github.com/gptscript-ai/gptscript/tree/main/docs/",
          routeBasePath: "/", // Serve the docs at the site's root
        },
        theme: {
          customCss: "./src/css/custom.css",
        },
        blog: false,
      }),
    ],
  ],

  themeConfig:
    /** @type {import('@docusaurus/preset-classic').ThemeConfig} */
    ({
      // Replace with your project's social card
      image: "img/docusaurus-social-card.jpg",
      navbar: {
        title: "GPTScript",
        style: "dark",
        logo: {
          alt: "GPTScript Logo",
          src: "img/logo.svg",
        },
        items: [
          {
            href: "https://github.com/gptscript-ai/gptscript",
            label: "GitHub",
            position: "right",
          },
        ],
      },
      footer: {
        style: "dark",
        links: [
          {
            label: "GitHub",
            to: "https://github.com/gptscript-ai/gptscript",
          },
        ],
        copyright: `Copyright © ${new Date().getFullYear()} Acorn Labs, Inc`,
      },
      prism: {
        theme: prismThemes.github,
        darkTheme: prismThemes.dracula,
        additionalLanguages: ["cue", "docker"],
      },
    }),
};

export default config;
