module.exports = {
  theme: "cosmos",
  title: "IBC-Go",
  locales: {
    "/": {
      lang: "en-US",
    },
  },
  base: process.env.VUEPRESS_BASE || "/",
  head: [
    [
      "link",
      {
        rel: "apple-touch-icon",
        sizes: "180x180",
        href: "/apple-touch-icon.png",
      },
    ],
    [
      "link",
      {
        rel: "icon",
        type: "image/png",
        sizes: "32x32",
        href: "/favicon-32x32.png",
      },
    ],
    [
      "link",
      {
        rel: "icon",
        type: "image/png",
        sizes: "16x16",
        href: "/favicon-16x16.png",
      },
    ],
    ["link", { rel: "manifest", href: "/site.webmanifest" }],
    ["meta", { name: "msapplication-TileColor", content: "#2e3148" }],
    ["meta", { name: "theme-color", content: "#ffffff" }],
    ["link", { rel: "icon", type: "image/svg+xml", href: "/favicon-svg.svg" }],
    [
      "link",
      {
        rel: "apple-touch-icon-precomposed",
        href: "/apple-touch-icon-precomposed.png",
      },
    ],
  ],
  themeConfig: {
    repo: "cosmos/ibc-go",
    docsRepo: "cosmos/ibc-go",
    docsBranch: "main",
    docsDir: "docs",
    editLinks: true,
    label: "ibc",
    // TODO
    //algolia: {
    //  id: "BH4D9OD16A",
    //  key: "ac317234e6a42074175369b2f42e9754",
    //  index: "ibc-go"
    //},
    versions: [
      {
        label: "main",
        key: "main",
      },
      {
        label: "v1.1.0",
        key: "v1.1.0",
      },
      {
        label: "v1.2.0",
        key: "v1.2.0",
      },
      {
        label: "v1.3.0",
        key: "v1.3.0",
      },
      {
        label: "v1.5.0",
        key: "v1.5.0",
      },
      {
        label: "v1.4.0",
        key: "v1.4.0",
      },
      {
        label: "v2.0.0",
        key: "v2.0.0",
      },
      {
        label: "v2.1.0",
        key: "v2.1.0",
      },
      {
        label: "v2.2.0",
        key: "v2.2.0",
      },
      {
        label: "v2.3.0",
        key: "v2.3.0",
      },
      {
        label: "v2.4.0",
        key: "v2.4.0",
      },
      {
        label: "v2.5.0",
        key: "v2.5.0",
      },
      {
        label: "v3.0.0",
        key: "v3.0.0",
      },
      {
        label: "v3.1.0",
        key: "v3.1.0",
      },
      {
        label: "v3.2.0",
        key: "v3.2.0",
      },
      {
        label: "v3.3.0",
        key: "v3.3.0",
      },
      {
        label: "v3.4.0",
        key: "v3.4.0",
      },
      {
        label: "v4.0.0",
        key: "v4.0.0",
      },
      {
        label: "v4.1.0",
        key: "v4.1.0",
      },
      {
        label: "v4.2.0",
        key: "v4.2.0",
      },
      {
        label: "v4.3.0",
        key: "v4.3.0",
      },
      {
        label: "v4.4.0",
        key: "v4.4.0",
      },
      {
        label: "v5.0.0",
        key: "v5.0.0",
      },
      {
        label: "v5.1.0",
        key: "v5.1.0",
      },
      {
        label: "v5.2.0",
        key: "v5.2.0",
      },
      {
        label: "v5.3.0",
        key: "v5.3.0",
      },
      {
        label: "v6.1.0",
        key: "v6.1.0",
      },
      {
        label: "v6.2.0",
        key: "v6.2.0",
      },
      {
        label: "v7.0.0",
        key: "v7.0.0",
      },
      {
        label: "v7.1.0",
        key: "v7.1.0",
      },
    ],
    topbar: {
      banner: true,
    },
    sidebar: {
      auto: false,
      nav: [
        {
          title: "Using IBC-Go",
          children: [
            {
              title: "Overview",
              directory: false,
              path: "/ibc/overview.html",
            },
            {
              title: "Integration",
              directory: false,
              path: "/ibc/integration.html",
            },
            {
              title: "Applications",
              directory: true,
              path: "/ibc/apps",
            },
            {
              title: "Middleware",
              directory: true,
              path: "/ibc/middleware",
            },
            {
              title: "Upgrades",
              directory: true,
              path: "/ibc/upgrades",
            },
            {
              title: "Governance Proposals",
              directory: false,
              path: "/ibc/proposals.html",
            },
            {
              title: "Relayer",
              directory: false,
              path: "/ibc/relayer.html",
            },
            {
              title: "Protobuf Documentation",
              directory: false,
              path: "/ibc/proto-docs.html",
            },
            {
              title: "Roadmap",
              directory: false,
              path: "/roadmap/roadmap.html",
            },
            {
              title: "Troubleshooting",
              directory: false,
              path: "/ibc/troubleshooting.html",
            },
          ],
        },
        {
          title: "IBC Application Modules",
          children: [
            {
              title: "Interchain Accounts",
              directory: true,
              path: "/apps",
              children: [
                {
                  title: "Overview",
                  directory: false,
                  path: "/apps/interchain-accounts/overview.html",
                },
                {
                  title: "Development Use Cases",
                  directory: false,
                  path: "/apps/interchain-accounts/development.html",
                },
                {
                  title: "Authentication Modules",
                  directory: false,
                  path: "/apps/interchain-accounts/auth-modules.html",
                },
                {
                  title: "Integration",
                  directory: false,
                  path: "/apps/interchain-accounts/integration.html",
                },
                {
                  title: "Messages",
                  directory: false,
                  path: "/apps/interchain-accounts/messages.html",
                },
                {
                  title: "Parameters",
                  directory: false,
                  path: "/apps/interchain-accounts/parameters.html",
                },
                {
                  title: "Client",
                  directory: false,
                  path: "/apps/interchain-accounts/client.html",
                },
                {
                  title: "Active Channels",
                  directory: false,
                  path: "/apps/interchain-accounts/active-channels.html",
                },
                {
                  title: "Legacy",
                  directory: true,
                  path: "/apps/interchain-accounts",
                  children: [
                    {
                      title: "Authentication Modules",
                      directory: false,
                      path: "/apps/interchain-accounts/legacy/auth-modules.html",
                    },
                    {
                      title: "Integration",
                      directory: false,
                      path: "/apps/interchain-accounts/legacy/integration.html",
                    },
                    {
                      title: "Keeper API",
                      directory: false,
                      path: "/apps/interchain-accounts/legacy/keeper-api.html",
                    },
                  ]
                },
              ],
            },
            {
              title: "Transfer",
              directory: true,
              path: "/apps",
              children: [
                {
                  title: "Overview",
                  directory: false,
                  path: "/apps/transfer/overview.html",
                },
                {
                  title: "State",
                  directory: false,
                  path: "/apps/transfer/state.html",
                },
                {
                  title: "State Transitions",
                  directory: false,
                  path: "/apps/transfer/state-transitions.html",
                },
                {
                  title: "Messages",
                  directory: false,
                  path: "/apps/transfer/messages.html",
                },
                {
                  title: "Events",
                  directory: false,
                  path: "/apps/transfer/events.html",
                },
                {
                  title: "Metrics",
                  directory: false,
                  path: "/apps/transfer/metrics.html",
                },
                {
                  title: "Params",
                  directory: false,
                  path: "/apps/transfer/params.html",
                },
                {
                  title: "Authorizations",
                  directory: false,
                  path: "/apps/transfer/authorizations.html",
                },
                {
                  title: "Client",
                  directory: false,
                  path: "/apps/transfer/client.html",
                },
              ],
            },
          ],
        },
        {
          title: "IBC Light Clients",
          children: [
            {
              title: "Developer Guide",
              directory: true,
              path: "/ibc/light-clients",
              children: [
                {
                  title: "Overview",
                  directory: false,
                  path: "/ibc/light-clients/overview.html",
                },
                {
                  title: "Client State interface",
                  directory: false,
                  path: "/ibc/light-clients/client-state.html",
                },
                {
                  title: "Consensus State interface",
                  directory: false,
                  path: "/ibc/light-clients/consensus-state.html",
                },
                {
                  title: "Handling Updates and Misbehaviour",
                  directory: false,
                  path: "/ibc/light-clients/updates-and-misbehaviour.html",
                },
                {
                  title: "Handling Upgrades",
                  directory: false,
                  path: "/ibc/light-clients/upgrades.html",
                },
                {
                  title: "Existence/Non-Existence Proofs",
                  directory: false,
                  path: "/ibc/light-clients/proofs.html",
                },
                {
                  title: "Handling Proposals",
                  directory: false,
                  path: "/ibc/light-clients/proposals.html",
                },
                {
                  title: "Handling Genesis",
                  directory: false,
                  path: "/ibc/light-clients/genesis.html",
                },
                {
                  title: "Setup",
                  directory: false,
                  path: "/ibc/light-clients/setup.html",
                },
              ]
            },
            {
              title: "Localhost",
              directory: true,
              path: "/ibc/light-clients/localhost",
              children: [
                {
                  title: "Overview",
                  directory: false,
                  path: "/ibc/light-clients/localhost/overview.html",
                },
                {
                  title: "Integration",
                  directory: false,
                  path: "/ibc/light-clients/localhost/integration.html",
                },
                {
                  title: "ClientState",
                  directory: false,
                  path: "/ibc/light-clients/localhost/client-state.html",
                },
                {
                  title: "Connection",
                  directory: false,
                  path: "/ibc/light-clients/localhost/connection.html",
                },
                {
                  title: "State Verification",
                  directory: false,
                  path: "/ibc/light-clients/localhost/state-verification.html",
                },
              ],
            },
            {
              title: "Solomachine",
              directory: true,
              path: "/ibc/light-clients/solomachine",
              children: [
                {
                  title: "Solomachine",
                  directory: false,
                  path: "/ibc/light-clients/solomachine/solomachine.html",
                },
                {
                  title: "Concepts",
                  directory: false,
                  path: "/ibc/light-clients/solomachine/concepts.html",
                },
                {
                  title: "State",
                  directory: false,
                  path: "/ibc/light-clients/solomachine/state.html",
                },
                {
                  title: "State Transitions",
                  directory: false,
                  path: "/ibc/light-clients/solomachine/state_transitions.html",
                },
              ],
            },
          ],
        },
        {
          title: "IBC Middleware Modules",
          children: [
            {
              title: "Fee Middleware",
              directory: true,
              path: "/middleware",
              children: [
                {
                  title: "Overview",
                  directory: false,
                  path: "/middleware/ics29-fee/overview.html",
                },
                {
                  title: "Integration",
                  directory: false,
                  path: "/middleware/ics29-fee/integration.html",
                },
                {
                  title: "Fee Messages",
                  directory: false,
                  path: "/middleware/ics29-fee/msgs.html",
                },
                {
                  title: "Fee Distribution",
                  directory: false,
                  path: "/middleware/ics29-fee/fee-distribution.html",
                },
                {
                  title: "Events",
                  directory: false,
                  path: "/middleware/ics29-fee/events.html",
                },
                {
                  title: "End Users",
                  directory: false,
                  path: "/middleware/ics29-fee/end-users.html",
                },
              ],
            },
          ],
        },
        {
          title: "Migrations",
          children: [
            {
              title:
                "Support transfer of coins whose base denom contains slashes",
              directory: false,
              path: "/migrations/support-denoms-with-slashes.html",
            },
            {
              title: "SDK v0.43 to IBC-Go v1",
              directory: false,
              path: "/migrations/sdk-to-v1.html",
            },
            {
              title: "IBC-Go v1 to v2",
              directory: false,
              path: "/migrations/v1-to-v2.html",
            },
            {
              title: "IBC-Go v2 to v3",
              directory: false,
              path: "/migrations/v2-to-v3.html",
            },
            {
              title: "IBC-Go v3 to v4",
              directory: false,
              path: "/migrations/v3-to-v4.html",
            },
            {
              title: "IBC-Go v4 to v5",
              directory: false,
              path: "/migrations/v4-to-v5.html",
            },
            {
              title: "IBC-Go v5 to v6",
              directory: false,
              path: "/migrations/v5-to-v6.html",
            },
            {
              title: "IBC-Go v6 to v7",
              directory: false,
              path: "/migrations/v6-to-v7.html",
            },
            {
              title: "IBC-Go v7 to v7.1",
              directory: false,
              path: "/migrations/v7-to-v7_1.html",
            },
          ],
        },
        {
          title: "Resources",
          children: [
            {
              title: "IBC Specification",
              path: "https://github.com/cosmos/ibc",
            },
          ],
        },
      ],
    },
    gutter: {
      title: "Help & Support",
      editLink: true,
      chat: {
        title: "Discord",
        text: "Chat with IBC developers on Discord.",
        url: "https://discordapp.com/channels/669268347736686612",
        bg: "linear-gradient(225.11deg, #2E3148 0%, #161931 95.68%)",
      },
      github: {
        title: "Found an Issue?",
        text: "Help us improve this page by suggesting edits on GitHub.",
      },
    },
    footer: {
      question: {
        text: "Chat with IBC developers in <a href='https://discord.gg/W8trcGV' target='_blank'>Discord</a>.",
      },
      textLink: {
        text: "ibcprotocol.dev",
        url: "https://ibcprotocol.dev",
      },
      services: [
        {
          service: "medium",
          url: "https://blog.cosmos.network/",
        },
        {
          service: "twitter",
          url: "https://twitter.com/cosmos",
        },
        {
          service: "linkedin",
          url: "https://www.linkedin.com/company/interchain-gmbh",
        },
        {
          service: "reddit",
          url: "https://reddit.com/r/cosmosnetwork",
        },
        {
          service: "telegram",
          url: "https://t.me/cosmosproject",
        },
        {
          service: "youtube",
          url: "https://www.youtube.com/c/CosmosProject",
        },
      ],
      smallprint:
        "The development of IBC-Go is led primarily by [Interchain GmbH](https://interchain.berlin/). Funding for this development comes primarily from the Interchain Foundation, a Swiss non-profit.",
      links: [
        {
          title: "Documentation",
          children: [
            {
              title: "Cosmos SDK",
              url: "https://docs.cosmos.network",
            },
            {
              title: "Cosmos Hub",
              url: "https://hub.cosmos.network",
            },
            {
              title: "Tendermint Core",
              url: "https://docs.tendermint.com",
            },
          ],
        },
        {
          title: "Community",
          children: [
            {
              title: "Cosmos blog",
              url: "https://blog.cosmos.network",
            },
            {
              title: "Forum",
              url: "https://forum.cosmos.network",
            },
            {
              title: "Chat",
              url: "https://discord.gg/W8trcGV",
            },
          ],
        },
        {
          title: "Contributing",
          children: [
            {
              title: "Contributing to the docs",
              url: "https://github.com/cosmos/ibc-go/blob/main/docs/DOCS_README.md",
            },
            {
              title: "Source code on GitHub",
              url: "https://github.com/cosmos/ibc-go/",
            },
          ],
        },
      ],
    },
  },
  plugins: [
    [
      "@vuepress/google-analytics",
      {
        ga: "UA-51029217-2",
      },
    ],
    [
      "sitemap",
      {
        hostname: "https://ibc.cosmos.network",
      },
    ],
  ],
};
