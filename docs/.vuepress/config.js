module.exports = {
  theme: "cosmos",
  title: "IBC-GO",
  locales: {
    "/": {
      lang: "en-US"
    },
  },
  base: process.env.VUEPRESS_BASE || "/",
  head: [
    ['link', { rel: "apple-touch-icon", sizes: "180x180", href: "/apple-touch-icon.png" }],
    ['link', { rel: "icon", type: "image/png", sizes: "32x32", href: "/favicon-32x32.png" }],
    ['link', { rel: "icon", type: "image/png", sizes: "16x16", href: "/favicon-16x16.png" }],
    ['link', { rel: "manifest", href: "/site.webmanifest" }],
    ['meta', { name: "msapplication-TileColor", content: "#2e3148" }],
    ['meta', { name: "theme-color", content: "#ffffff" }],
    ['link', { rel: "icon", type: "image/svg+xml", href: "/favicon-svg.svg" }],
    ['link', { rel: "apple-touch-icon-precomposed", href: "/apple-touch-icon-precomposed.png" }],
  ],
  themeConfig: {
    repo: "cosmos/ibc-go",
    docsRepo: "cosmos/ibc-go",
    docsDir: "docs",
    editLinks: true,
    label: "sdk",
    algolia: {
      id: "BH4D9OD16A",
      key: "ac317234e6a42074175369b2f42e9754",
      index: "cosmos-sdk"
    },
    versions: [
      {
        "label": "main",
        "key": "main"
      }
    ],
    topbar: {
      banner: true
    },
    sidebar: { 
      auto: true,
      nav: [
        {
          title: "Using IBC-GO",
          children: [
            {
              title: "Modules",
              directory: true,
              path: "/modules"
            }
          ]
        },
        {
          title: "Resources",
          children: [
            {
              title: "IBC Specification",
              path: "https://github.com/cosmos/ibc"
            },
          ]
        }
      ]
    },
    gutter: {
      title: "Help & Support",
      editLink: true,
      chat: {
        title: "Discord",
        text: "Chat with IBC developers on Discord.",
        url: "https://discordapp.com/channels/669268347736686612",
        bg: "linear-gradient(225.11deg, #2E3148 0%, #161931 95.68%)"
      },
      github: {
        title: "Found an Issue?",
        text: "Help us improve this page by suggesting edits on GitHub."
      }
    },
    footer: {
      logo: "/logo-bw.svg",
      textLink: {
        text: "ibcprotocol.org",
        url: "https://ibcprotocol.org"
      },
      links: [
        {
          title: "Documentation",
          children: [
            {
              title: "IBC GO",
              url: "https://ibc.cosmos.network"
            },
            {
              title: "Cosmos SDK",
              url: "https://docs.cosmos.network"
            },
            {
              title: "Cosmos Hub",
              url: "https://hub.cosmos.network"
            },
            {
              title: "Tendermint Core",
              url: "https://docs.tendermint.com"
            }
          ]
        },
        {
          title: "Community",
          children: [
            {
              title: "Cosmos blog",
              url: "https://blog.cosmos.network"
            },
            {
              title: "Forum",
              url: "https://forum.cosmos.network"
            },
            {
              title: "Chat",
              url: "https://discord.gg/W8trcGV"
            }
          ]
        },
        {
          title: "Contributing",
          children: [
            {
              title: "Contributing to the docs",
              url:
                "https://github.com/cosmos/ibc-go/blob/main/docs/DOCS_README.md"
            },
            {
              title: "Source code on GitHub",
              url: "https://github.com/cosmos/ibc-go/"
            }
          ]
        }
      ]
    }
  },
  plugins: [
    [
      "@vuepress/google-analytics",
      {
        ga: "UA-51029217-2"
      }
    ],
    [
      "sitemap",
      {
        hostname: "https://ibc.cosmos.network"
      }
    ]
  ]
};
