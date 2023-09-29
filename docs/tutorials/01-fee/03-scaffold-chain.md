---
title: Scaffold a Cosmos SDK Blockchain with Ignite
sidebar_label: Scaffold a Cosmos SDK Blockchain with Ignite
sidebar_position: 3
slug: /fee/scaffold-chain
---

import CodeBlock from '@theme/CodeBlock';

# Scaffold a Cosmos SDK blockchain with Ignite

In this tutorial, we will not be going through the process of creating a Cosmos SDK module. Instead, we will integrate the ICS-29 Fee Middleware into an existing Cosmos SDK blockchain. Scaffold the blockchain without any custom modules using the following command.

```bash
ignite scaffold chain foo --no-module
```

This will create a new blockchain in the `foo` directory. The `foo` directory will contain [these files and directories](https://github.com/srdtrk/cosmoverse2023-ibc-fee-demo/tree/0f41b3c6b4e065aa1a860de3e3038d489c37a28a). Verify that this chain runs with

```bash
cd foo
ignite chain serve --reset-once
```

Once it is running quit by pressing `q`. This blockchain comes with Cosmos SDK `v0.47.3` and IBC-Go `v7.1.0`. We can update Cosmos SDK to its latest patch version and update IBC-Go to its latest minor version by running these two commands.

<CodeBlock className="language-bash" title=<a href="https://github.com/srdtrk/cosmoverse2023-ibc-fee-demo/tree/88e2fa73c833523cba2122d4b2a41eb8e3b8d86e">View Source</a>>
go get github.com/cosmos/cosmos-sdk@v0.47.5 && go mod tidy
</CodeBlock>

<CodeBlock className="language-bash" title=<a href="https://github.com/srdtrk/cosmoverse2023-ibc-fee-demo/tree/2e2c2a3b8e13fd5e23c3b59894438494af6fc32a">View Source</a>>
go get github.com/cosmos/ibc-go/v7@v7.3.0 && go mod tidy
</CodeBlock>

Feel free to test that the chain still runs with `ignite chain serve --reset-once`. Do not forget to quit by pressing `q`.
