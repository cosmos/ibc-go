---
title: Set Up Your Work Environment
sidebar_label: Set Up Your Work Environment
sidebar_position: 2
slug: /channel-upgrades/setup-env
---

import HighlightBox from '@site/src/components/HighlightBox';

# Set up your work environment

On this page, you can find helpful links to set up your work environment.

<HighlightBox type="info" title="Dependencies">

In this section, you can find all you need to install:

- [jq](https://jqlang.github.io/jq/)
- [gm](https://github.com/informalsystems/gm/)
- [ibc-go simd](https://github.com/cosmos/ibc-go/)
- [Hermes v1.9.0](https://hermes.informal.systems/)

</HighlightBox>

## jq

Install `jq` following the instructions on [its website](https://jqlang.github.io/jq/download/). Test if it is installed by running the following command:

```bash
jq --version
```

At the moment of writing this tutorial, the version of `jq` used was 1.6.

## gm

The [gaiad manager](https://github.com/informalsystems/gm) (`gm`) is a configurable command-line tool (CLI) that helps manage local gaiad networks. It can be used to easily and quickly run a local setup of multiple blockchains. Follow the installation steps [here](https://github.com/informalsystems/gm#how-to-run).

## ibc-go simd

Download the simd binary from the [v8.1.0 release](https://github.com/cosmos/ibc-go/releases/tag/v8.1.0). This chain binary has the Fee Middleware already wired up and wrapping the ICS 20 transfer application. If you want to know how to wire up the Fee Middleware, please read the wiring section from the Fee Middleware tutorial.

## Hermes

Install Hermes relayer version `v1.9.0` via cargo following the instructions on the [Hermes website](https://hermes.informal.systems/quick-start/installation.html#install-via-cargo) or by using the command below.

```bash
cargo install ibc-relayer-cli --version 1.9.0 --bin hermes --locked
```

Test if Hermes is installed by running the following command:

```bash
hermes version
```

# Folder structure

This tutorial assumes the following folder structure:

```text
testing
├── bin
│   ├── chain1
│   │   ├── simd
│   │   └── proposal.json
│   └── chain2
│       └── simd
├── gm
└── hermes
    ├── hermes
    └── config.toml
```

`simd` if the chain binary that will be used to run 2 blockchains (`chain1` and `chain2`). THe folder `gm` will contain the data folders for both blockchains.
