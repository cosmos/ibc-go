<div align="left">
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

The [Inter-Blockchain Communication protocol (IBC)](https://ibcprotocol.dev/) is a blockchain interoperability solution that allows blockchains to talk to each other. Blockchains that speak IBC can transfer any kind of data encoded in bytes — including tokens, messages, and arbitrary application logic. IBC is secure, permissionless, and designed to connect sovereign blockchains into a single interoperable network. For a high-level explanation of what IBC is and how it works, please read [this article](https://ibcprotocol.dev/how-ibc-works).

The IBC implementation in Golang `ibc-go` has been used in production by the majority of the 200+ chains that have utilized IBC. It is built as a Cosmos SDK module. To understand more about how to use the `ibc-go` module as well as learn more about the IBC Protocol, please check out [our docs](./docs/docs/01-ibc/01-overview.md).

## Releases

The release lines currently supported are v8, and v10. Please note that v9 has been retracted and has been replaced by v10.

Please refer to our [versioning guide](https://github.com/cosmos/ibc-go/blob/main/RELEASES.md) for more information on how to understand our release versioning.

## Applications, Middleware, and Tools

IBC has an extensive list of applications, middleware, and tools, including relayers. View the list on the [IBC technical resource catalogue](https://ibcprotocol.dev/technical-resource-catalog) on our website.

## Developer Community and Support

The issue list of this repo is exclusively for bug reports and feature requests. We have active, helpful communities on Discord, Telegram, and Slack.

**| Need Help? | Support & Community: [Discord](https://discord.com/invite/interchain) - [Telegram](https://t.me/CosmosOG) - [Talk to an Expert](https://cosmos.network/interest-form) - [Join the #Cosmos-tech Slack Channel](https://forms.gle/A8jawLgB8zuL1FN36) |**

## Security

To report a security vulnerability, see our [Coordinated Vulnerability Disclosure Policy](./SECURITY.md).

## Audits

The following audits have been performed on the `ibc-go` source code:

- [ICS20 Fungible Token Transfer](https://github.com/informalsystems/audits/tree/dc8b503727adcbb8e29c3d3a25a9070e0bf1ec87/IBC-GO) by Informal Systems.
- [ICS20 Fungible Token Transfer V2](https://github.com/cosmos/ibc-go/blob/main/docs/audits/20-token-transfer/Atredis%20Partners%20-%20Interchain%20ICS20%20v2%20New%20Features%20Assessment%20-%20Report%20v1.0.pdf) by Atredis Partners.
- ICS27 Interchain Accounts by [Trail of Bits](https://github.com/cosmos/ibc-go/blob/main/docs/audits/27-interchain-accounts/Trail%20of%20Bits%20audit%20-%20Final%20Report.pdf) and [Informal Systems](https://github.com/cosmos/ibc-go/issues/631).
- [ICS08 Wasm Clients](https://github.com/cosmos/ibc-go/blob/main/docs/audits/08-wasm/Ethan%20Frey%20-%20Wasm%20Client%20Review.pdf) by Ethan Frey/Confio.
- [ICS04 Channel upgradability](https://github.com/cosmos/ibc-go/blob/main/docs/audits/04-channel-upgrades/Atredis%20Partners%20-%20Interchain%20Foundation%20IBC-Go%20Channel%20Upgrade%20Feature%20Assessment%20-%20Report%20v1.1.pdf) by Atredis Partners.

## Maintainers
[Cosmos Labs](https://cosmoslabs.io/) maintains the core components of the stack: Cosmos SDK, CometBFT, IBC, Cosmos EVM, and various developer tools and frameworks. The detailed maintenance policy can be found [here](https://github.com/cosmos/security/blob/main/POLICY.md). In addition to developing and maintaining the Cosmos Stack, Cosmos Labs provides advisory and engineering services for blockchain solutions. [Get in touch with Cosmos Labs](https://www.cosmoslabs.io/contact).

Cosmos Labs is a wholly-owned subsidiary of the [Interchain Foundation](https://interchain.io/), the Swiss nonprofit responsible for treasury management, funding public goods, and supporting governance for Cosmos. 

The Cosmos Stack is supported by a robust community of open-source contributors. 

## Contributing to ibc-go

If you're interested in contributing to ibc-go, please take a look at the [contributing guidelines](./CONTRIBUTING.md). We welcome and appreciate community contributions!

To help contributors understand which issues are good to pick up, we have the following two categories:

- Issues with the label [`good first issue`](https://github.com/cosmos/ibc-go/issues?q=is%3Aopen+is%3Aissue+label%3A%22good+first+issue%22) should be pretty well defined and are best suited for developers new to ibc-go.
- Issues with the label [`help wanted`](https://github.com/cosmos/ibc-go/issues?q=is%3Aopen+is%3Aissue+label%3A%22help+wanted%22) are a bit more involved and they usually require some familiarity already with the codebase.

If you are interested in working on an issue, please comment on it. We will be happy to answer any questions you may have and help you out while you work on the issue.

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

### IBC Information
- [IBC Website](https://ibcprotocol.dev/)
- [IBC Protocol Specification and Standards](https://github.com/cosmos/ibc)
- [Documentation](./docs/docs/01-ibc/01-overview.md)

### Cosmos Stack Libraries

- [Cosmos SDK](http://github.com/cosmos/cosmos-sdk) - A framework for building
  applications in Golang
- [CometBFT](https://github.com/cometbft/cometbft) - High-performance, 10k+ TPS configurable BFT consensus engine.
- [Cosmos EVM](https://github.com/cosmos/evm) - Native EVM layer for Cosmos SDK chains. 
