---
title: Set Up Your Work Environment
sidebar_label: Set Up Your Work Environment
sidebar_position: 2
slug: /fee/setup-env
---

import HighlightBox from '@site/src/components/HighlightBox';

# Set up your work environment

On this page, you can find helpful links to set up your work environment.

<HighlightBox type="info" title="Dependencies">

In this section, you can find all you need to install:

- [Git](https://git-scm.com/)
- [Go](https://go.dev/)
- [Hermes v1.6.0](https://hermes.informal.systems/)
- [Node.js v18](https://nodejs.org/en/)
- [Ignite v0.27.1](https://docs.ignite.com/)
- [Keplr](https://www.keplr.app/)

</HighlightBox>

<HighlightBox type="note" title="Note">

On a general note, it is advisable to prepare a separate project folder to keep all your Cosmos exercises.

</HighlightBox>

## Git

Install Git following the instructions on the [Git website](https://git-scm.com/). Test if Git is installed by running the following command:

```bash
git --version
```

## Go

Install the latest version of Go following the instructions on the [Go website](https://go.dev/). Test if Go is installed by running the following command:

```bash
go version
```

## Hermes

Install Hermes relayer version `v1.6.0` via cargo following the instructions on the [Hermes website](https://hermes.informal.systems/quick-start/installation.html#install-via-cargo) or by using the command below.

```bash
cargo install ibc-relayer-cli --version 1.6.0 --bin hermes --locked
```

Test if Hermes is installed by running the following command:

```bash
hermes version
```

## Node.js

Install version 18 of Node.js following the instructions on the [Node.js website](https://nodejs.org/en/). Test if Node.js is installed by running the following command:

```bash
node --version
```

## Ignite

Install Ignite CLI version `v0.27.1` by running the following command or following the instructions on the [Ignite website](https://docs.ignite.com/welcome/install):

```bash
curl https://get.ignite.com/cli@v0.27.1! | bash
```

Test if the correct version of Ignite is installed by running the following command:

```bash
ignite version
```

## Keplr

Install Keplr to your browser following the instructions on the [Keplr website](https://www.keplr.app/).
