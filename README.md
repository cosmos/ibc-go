<div align="center">
  <h1>ibc-go</h1>
</div>

![banner](docs/static/img/IBC-go-cover.svg)

<div align="center">
  <a href="https://github.com/cosmos/ibc-go/releases/latest">
    <img alt="Version" src="https://img.shields.io/github/tag/cosmos/ibc-go.svg" />
  </a>
  <a href="https://github.com/cosmos/ibc-go/blob/main/LICENSE">
    <img alt="License: Apache-2.0" src="https://img.shields.io/github/license/cosmos/ibc-go.svg" />
  </a>
  <a href="https://pkg.go.dev/github.com/cosmos/ibc-go?tab=doc">
    <img alt="GoDoc" src="https://godoc.org/github.com/cosmos/ibc-go?status.svg" />
  </a>
  <a href="https://goreportcard.com/report/github.com/cosmos/ibc-go">
    <img alt="Go report card" src="https://goreportcard.com/badge/github.com/cosmos/ibc-go" />
  </a>
  <a href="https://codecov.io/gh/cosmos/ibc-go" > 
    <img src="https://codecov.io/gh/cosmos/ibc-go/graph/badge.svg?token=bvveHATeIn"/> 
  </a>
</div>
<div align="center">
  <a href="https://discord.com/invite/interchain">
    <img alt="Discord" src="https://img.shields.io/discord/669268347736686612.svg" />
  </a>
  <a href="https://sourcegraph.com/github.com/cosmos/ibc-go?badge">
    <img alt="Imported by" src="https://sourcegraph.com/github.com/cosmos/ibc-go/-/badge.svg" />
  </a>
    <img alt="Tests / Code Coverage Status" src="https://github.com/cosmos/ibc-go/workflows/Tests%20/%20Code%20Coverage/badge.svg" />
    <img alt="E2E Status" src="https://github.com/cosmos/ibc-go/workflows/Tests%20/%20E2E/badge.svg" />
  <a href="https://deepwiki.com/cosmos/ibc-go"><img src="https://deepwiki.com/badge.svg" alt="Ask DeepWiki"></a>
</div>

The [Inter-Blockchain Communication protocol (IBC)](https://ibcprotocol.dev/) allows blockchains to talk to each other. This end-to-end, connection-oriented, stateful protocol provides reliable, ordered, and authenticated communication between heterogeneous blockchains. For a high-level explanation of what IBC is and how it works, please read [this blog post](https://medium.com/the-interchain-foundation/eli5-what-is-ibc-def44d7b5b4c).

This IBC implementation in Golang is built as a Cosmos SDK module. To understand more about how to use the `ibc-go` module as well as about the IBC protocol, please check out the Interchain Developer Academy [section on IBC](https://tutorials.cosmos.network/academy/3-ibc/), or [our docs](./docs/docs/01-ibc/01-overview.md).

## Roadmap

For an overview of upcoming changes to ibc-go take a look at the [roadmap project board](https://github.com/orgs/cosmos/projects/38/views/14).

## Releases

The release lines currently supported are v7, and v8 (Note: v9 has been retracted and will be replaced by v10).

Please refer to the [Stable Release Policy section of RELEASES.md](https://github.com/cosmos/ibc-go/blob/main/RELEASES.md#stable-release-policy) for more details.

Please refer to our [versioning guide](https://github.com/cosmos/ibc-go/blob/main/RELEASES.md) for more information on how to understand our release versioning.

## Ecosystem

Discover more applications and middleware in the [cosmos/ibc-apps repository](https://github.com/cosmos/ibc-apps#-bonus-content).

## Community

We have active, helpful communities on Discord and Telegram.

For questions and support please use the `developers` channel in the [Cosmos Network Discord server](https://discord.com/channels/669268347736686612/1019978171367559208) or join the [Interchain Discord server](https://discord.com/invite/interchain). The issue list of this repo is exclusively for bug reports and feature requests.

To receive announcements of new releases or other technical updates, please join the [Telegram group that Interchain Labs administers](https://t.me/cosmostechstack/1).

## Contributing

If you're interested in contributing to ibc-go, please take a look at the [contributing guidelines](./CONTRIBUTING.md). We welcome and appreciate community contributions!

This project adheres to ibc-go's [code of conduct](./CODE_OF_CONDUCT.md). By participating, you are expected to uphold this code.

To help contributors understand which issues are good to pick up, we have the following two categories:

- Issues with the label [`good first issue`](https://github.com/cosmos/ibc-go/issues?q=is%3Aopen+is%3Aissue+label%3A%22good+first+issue%22) should be pretty well defined and are best suited for developers new to ibc-go.
- Issues with the label [`help wanted`](https://github.com/cosmos/ibc-go/issues?q=is%3Aopen+is%3Aissue+label%3A%22help+wanted%22) are a bit more involved and they usually require some familiarity already with the codebase.

If you are interested in working on an issue, please comment on it; then we will be able to assign it to you. We will be happy to answer any questions you may have and help you out while you work on the issue.

If you have any general questions or feedback, please reach out to us in the [Interchain Discord server](https://discord.com/invite/interchain).

## Security

To report a security vulnerability, see our [Coordinated Vulnerability Disclosure Policy](./SECURITY.md).

## Audits

The following audits have been performed on the `ibc-go` source code:

- [ICS20 Fungible Token Transfer](https://github.com/informalsystems/audits/tree/dc8b503727adcbb8e29c3d3a25a9070e0bf1ec87/IBC-GO) by Informal Systems.
- [ICS20 Fungible Token Transfer V2](https://github.com/cosmos/ibc-go/blob/main/docs/audits/20-token-transfer/Atredis%20Partners%20-%20Interchain%20ICS20%20v2%20New%20Features%20Assessment%20-%20Report%20v1.0.pdf) by Atredis Partners.
- ICS27 Interchain Accounts by [Trail of Bits](https://github.com/cosmos/ibc-go/blob/main/docs/audits/27-interchain-accounts/Trail%20of%20Bits%20audit%20-%20Final%20Report.pdf) and [Informal Systems](https://github.com/cosmos/ibc-go/issues/631).
- [ICS08 Wasm Clients](https://github.com/cosmos/ibc-go/blob/main/docs/audits/08-wasm/Ethan%20Frey%20-%20Wasm%20Client%20Review.pdf) by Ethan Frey/Confio.
- [ICS04 Channel upgradability](https://github.com/cosmos/ibc-go/blob/main/docs/audits/04-channel-upgrades/Atredis%20Partners%20-%20Interchain%20Foundation%20IBC-Go%20Channel%20Upgrade%20Feature%20Assessment%20-%20Report%20v1.1.pdf) by Atredis Partners.

## Quick Navigation

1. **[Core IBC Implementation](https://github.com/cosmos/ibc-go/tree/main/modules/core)**

   1.1 [ICS 02 Client](https://github.com/cosmos/ibc-go/tree/main/modules/core/02-client)

   1.2 [ICS 03 Connection](https://github.com/cosmos/ibc-go/tree/main/modules/core/03-connection)

   1.3 [ICS 04 Channel](https://github.com/cosmos/ibc-go/tree/main/modules/core/04-channel)

   1.4 [ICS 05 Port](https://github.com/cosmos/ibc-go/tree/main/modules/core/05-port)

   1.5 [ICS 23 Commitment](https://github.com/cosmos/ibc-go/tree/main/modules/core/23-commitment/types)

   1.6 [ICS 24 Host](https://github.com/cosmos/ibc-go/tree/main/modules/core/24-host)

2. **Applications**

   2.1 [ICS 20 Fungible Token Transfers](https://github.com/cosmos/ibc-go/tree/main/modules/apps/transfer)

   2.2 [ICS 27 Interchain Accounts](https://github.com/cosmos/ibc-go/tree/main/modules/apps/27-interchain-accounts)

3. **Middleware**

    3.1 [Callbacks Middleware](https://github.com/cosmos/ibc-go/tree/main/modules/apps/callbacks)

4. **Light Clients**

   4.1 [ICS 07 Tendermint](https://github.com/cosmos/ibc-go/tree/main/modules/light-clients/07-tendermint)

   4.2 [ICS 06 Solo Machine](https://github.com/cosmos/ibc-go/tree/main/modules/light-clients/06-solomachine)

    4.3 [ICS 09 Localhost](https://github.com/cosmos/ibc-go/tree/main/modules/light-clients/09-localhost)

5. **[E2E Integration Tests](https://github.com/cosmos/ibc-go/tree/main/e2e)**

## Documentation and Resources

- [IBC Website](https://ibcprotocol.dev/)
- [IBC Protocol Specification](https://github.com/cosmos/ibc)
- [Documentation](./docs/docs/01-ibc/01-overview.md)
- [Interchain Developer Academy](https://tutorials.cosmos.network/academy/3-ibc/)

---

The development of ibc-go is led primarily by Interchain GmbH. Funding for this development comes primarily from the [Interchain Foundation](https://interchain.io), a Swiss non-profit.
