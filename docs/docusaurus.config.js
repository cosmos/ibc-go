// @ts-check
// Note: type annotations allow type checking and IDEs autocompletion

const lightCodeTheme = require("prism-react-renderer/themes/github");
const darkCodeTheme = require("prism-react-renderer/themes/dracula");

/** @type {import('@docusaurus/types').Config} */
const config = {
  title: "IBC-Go",
  tagline: "Documentation for IBC-Go",
  favicon: "img/white-cosmos-icon.svg",

  // Set the production url of your site here
  // for local production tests, set to http://localhost:3000/
  url: "https://ibc.cosmos.network",
  // Set the /<baseUrl>/ pathname under which your site is served
  // For GitHub pages deployment, it is often '/<projectName>/'
  baseUrl: "/",

  // GitHub pages deployment config.
  // If you aren't using GitHub pages, you don't need these.
  organizationName: "cosmos", // Usually your GitHub org/user name.
  projectName: "ibc-go", // Usually your repo name.
  deploymentBranch: "gh-pages",
  trailingSlash: false,

  onBrokenLinks: "log",
  onBrokenMarkdownLinks: "log",

  // Even if you don't use internalization, you can use this field to set useful
  // metadata like html lang. For example, if your site is Chinese, you may want
  // to replace "en" with "zh-Hans".
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
          sidebarPath: require.resolve("./sidebars.js"),
          // Routed the docs to the root path
          routeBasePath: "/",
          // Exclude template markdown files from the docs
          exclude: ["**/*.template.md"],
          // Select the latest version
          lastVersion: "v7.3.x",
          // Assign banners to specific versions
          versions: {
            current: {
              path: "main",
              banner: "unreleased",
            },
            "v8.0.x": {
              path: "v8",
              banner: "none",
            },
            "v7.3.x": {
              path: "v7",
              banner: "none",
            },
            "v6.2.x": {
              path: "v6",
              banner: "none",
            },
            "v5.3.x": {
              path: "v5",
              banner: "none",
            },
            "v4.5.x": {
              path: "v4",
              banner: "none",
            },
          },
        },
        theme: {
          customCss: require.resolve("./src/css/custom.css"),
        },
        gtag: {
          trackingID: "G-HP8ZXWVLJG",
          anonymizeIP: true,
        },
      }),
    ],
  ],

  themeConfig:
    /** @type {import('@docusaurus/preset-classic').ThemeConfig} */
    ({
      image: "img/ibc-go-docs-social-card.png",
      navbar: {
        logo: {
          alt: "IBC Logo",
          src: "img/black-ibc-logo.svg",
          srcDark: "img/white-ibc-logo.svg",
          href: "/main/",
        },
        items: [
          {
            type: "docSidebar",
            sidebarId: "defaultSidebar",
            position: "left",
            label: "Documentation",
          },
          {
            type: "doc",
            position: "left",
            docId: "README",
            docsPluginId: "adrs",
            label: "Architecture Decision Records",
          },
          {
            type: "doc",
            position: "left",
            docId: "intro",
            docsPluginId: "tutorials",
            label: "Tutorials",
          },
          {
            type: "docsVersionDropdown",
            position: "right",
            dropdownActiveClassDisabled: true,
          },
          {
            href: "https://github.com/cosmos/ibc-go",
            html: `<svg width="24" height="24" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg" class="github-icon">
            <path fill-rule="evenodd" clip-rule="evenodd" d="M12 0.300049C5.4 0.300049 0 5.70005 0 12.3001C0 17.6001 3.4 22.1001 8.2 23.7001C8.8 23.8001 9 23.4001 9 23.1001C9 22.8001 9 22.1001 9 21.1001C5.7 21.8001 5 19.5001 5 19.5001C4.5 18.1001 3.7 17.7001 3.7 17.7001C2.5 17.0001 3.7 17.0001 3.7 17.0001C4.9 17.1001 5.5 18.2001 5.5 18.2001C6.6 20.0001 8.3 19.5001 9 19.2001C9.1 18.4001 9.4 17.9001 9.8 17.6001C7.1 17.3001 4.3 16.3001 4.3 11.7001C4.3 10.4001 4.8 9.30005 5.5 8.50005C5.5 8.10005 5 6.90005 5.7 5.30005C5.7 5.30005 6.7 5.00005 9 6.50005C10 6.20005 11 6.10005 12 6.10005C13 6.10005 14 6.20005 15 6.50005C17.3 4.90005 18.3 5.30005 18.3 5.30005C19 7.00005 18.5 8.20005 18.4 8.50005C19.2 9.30005 19.6 10.4001 19.6 11.7001C19.6 16.3001 16.8 17.3001 14.1 17.6001C14.5 18.0001 14.9 18.7001 14.9 19.8001C14.9 21.4001 14.9 22.7001 14.9 23.1001C14.9 23.4001 15.1 23.8001 15.7 23.7001C20.5 22.1001 23.9 17.6001 23.9 12.3001C24 5.70005 18.6 0.300049 12 0.300049Z" fill="currentColor"/>
            </svg>
            `,
            position: "right",
          },
        ],
      },
      footer: {
        links: [
          {
            items: [
              {
                html: `<a href="https://cosmos.network"><img src="/img/cosmos-logo-bw.svg" alt="Cosmos Logo"></a>`,
              },
            ],
          },
          {
            title: "Documentation",
            items: [
              {
                label: "Hermes Relayer",
                href: "https://hermes.informal.systems/",
              },
              {
                label: "Cosmos Hub",
                href: "https://hub.cosmos.network",
              },
              {
                label: "CometBFT",
                href: "https://docs.cometbft.com",
              },
            ],
          },
          {
            title: "Community",
            items: [
              {
                label: "Discord",
                href: "https://discord.gg/Wtmk6ZNa8G",
              },
              {
                label: "Twitter",
                href: "https://twitter.com/interchain_io",
              },
              {
                label: "YouTube",
                href: "https://www.youtube.com/@interchain_io",
              },
            ],
          },
          {
            title: "Other Tools",
            items: [
              {
                label: "Go Relayer",
                href: "https://github.com/cosmos/relayer",
              },
              {
                label: "ibc-rs",
                href: "https://github.com/cosmos/ibc-rs",
              },
              {
                label: "interchaintest",
                href: "https://github.com/strangelove-ventures/interchaintest",
              },
              {
                label: "CosmWasm",
                href: "https://cosmwasm.com/",
              },
            ],
          },
          {
            title: "More",
            items: [
              {
                label: "GitHub",
                href: "https://github.com/cosmos/ibc-go",
              },
              {
                label: "IBC Protocol Website",
                href: "https://www.ibcprotocol.dev/",
              },
              {
                label: "Privacy Policy",
                href: "https://v1.cosmos.network/privacy",
              },
            ],
          },
        ],
        logo: {
          alt: "Large IBC Logo",
          src: "img/black-large-ibc-logo.svg",
          srcDark: "img/white-large-ibc-logo.svg",
          width: 275,
        },
        copyright: `<p>The development of IBC-Go is led primarily by <a href="https://interchain.berlin/">Interchain GmbH</a>. Funding for this development comes primarily from the Interchain Foundation, a Swiss non-profit.</p>`,
      },
      prism: {
        theme: lightCodeTheme,
        darkTheme: darkCodeTheme,
        additionalLanguages: ["protobuf", "go-module", "yaml", "toml"],
        magicComments: [
          // Remember to extend the default highlight class name as well!
          {
            className: 'theme-code-block-highlighted-line',
            line: 'highlight-next-line',
            block: {start: 'highlight-start', end: 'highlight-end'},
          },
          {
            className: 'code-block-minus-diff-line',
            line: 'minus-diff-line',
            block: {start: 'minus-diff-start', end: 'minus-diff-end'},
          },
          {
            className: 'code-block-plus-diff-line',
            line: 'plus-diff-line',
            block: {start: 'plus-diff-start', end: 'plus-diff-end'},
          },
        ],
      },
    }),
  themes: ["@saucelabs/theme-github-codeblock"],
  plugins: [
    [
      "@docusaurus/plugin-content-docs",
      {
        id: "adrs",
        path: "architecture",
        routeBasePath: "architecture",
        sidebarPath: require.resolve("./sidebars.js"),
        exclude: ["**/*.template.md"],
      },
    ],
    [
      "@docusaurus/plugin-content-docs",
      {
        id: "tutorials",
        path: "tutorials",
        routeBasePath: "tutorials",
        sidebarPath: require.resolve("./sidebars.js"),
        exclude: ["**/*.template.md"],
      },
    ],
    [
      "@docusaurus/plugin-content-docs",
      {
        id: "events",
        path: "events",
        routeBasePath: "events",
        sidebarPath: false,
        exclude: ["**/*.template.md"],
      },
    ],
    [
      "@docusaurus/plugin-content-docs",
      {
        id: "params",
        path: "params",
        routeBasePath: "params",
        sidebarPath: false,
        exclude: ["**/*.template.md"],
      },
    ],
    [
      "@docusaurus/plugin-client-redirects",
      {
        // makes the default page next in production
        redirects: [
          {
            from: ["/", "/master", "/next", "/docs"],
            to: "/main/",
          },
        ],
      },
    ],
    [
      "@gracefullight/docusaurus-plugin-microsoft-clarity",
      { projectId: "idk9udvhuu" },
    ],
    [
      require.resolve("@easyops-cn/docusaurus-search-local"),
      {
        indexBlog: false,
        docsRouteBasePath: ["/", "architecture"],
        highlightSearchTermsOnTargetPage: true,
      },
    ],
    async function myPlugin(context, options) {
      return {
        name: "docusaurus-tailwindcss",
        configurePostCss(postcssOptions) {
          postcssOptions.plugins.push(require("postcss-import"));
          postcssOptions.plugins.push(require("tailwindcss/nesting"));
          postcssOptions.plugins.push(require("tailwindcss"));
          postcssOptions.plugins.push(require("autoprefixer"));
          return postcssOptions;
        },
      };
    },
  ],
};

module.exports = config;
