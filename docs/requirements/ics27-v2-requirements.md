# Business requirements

> **TL;DR**: The lack of a default underlying app (previously called the *authentication module*), and the need to separate application and authentication concerns were recognised as primary reasons for the slow adoption of ICS 27 Interchain Accounts (ICA).

## Problem

We believe that the lack of controller chains so far have been because:

- We did not develop a standardized authentication module, which created a bottleneck for chains looking to integrate the controller submodule.
- We did not have a clear understanding of all the use cases ICA would facilitate.
- We expected more chains to want to design custom authentication for ICA.
- Coupling ICA authentication and application logic was a misdesign.

## Objectives

- Make controller functionality integration in a chain easier.
- Introduce a message server in the controller submodule that exposes APIs for interchain account registration and control.
- Once application callbacks are implemented (via ADR 008) deprecate the APIs introduced in ibc-go v3.0.0.

## Scope

| Features  | Release |
| --------- | ------- |
| Register interchain accounts and send transactions to host chain via message passing. | v6.0.0 |
| Support application callbacks with message server in controller submodule (requires ADR 008). | N/A |

# User requirements

## Use cases

### Custom authentication module needs to access IBC packet callbacks

Application developers of custom authentication modules that wish to consume IBC packet callbacks and react upon packet acknowledgements or timeouts must continue using the controller submodule's legacy APIs. 

### Custom authentication module does not need access to IBC packet callbacks

The authentication module should interact with the controller submodule via the message server for registering interchain accounts and sending messages to it. 

### No need for custom authentication module

Chains not only want individual accounts to be able to use Interchain Accounts, but also for generic Cosmos SDK authentication modules such as `x/gov` and `x/group` to be able to register an interchain account and send messages. An example use case with `x/gov`: the Cosmos Hub (controller), upon governance authorization, sends some of its inflationary rewards to Osmosis (host) to provide liquidity and purchase ATOM GAMM shares, which are then sent back to the Hub in one flow.

# Functional requirements

## Assumptions

No further assumptions besides the ones listed in the v1 requirements document.

## Known limitations

1. Custom authentication modules that wish to consume IBC packet callbacks need to use the legacy APIs until ADR 008 is implemented.

## Terminology

See section [Definitions](https://github.com/cosmos/ibc/blob/main/spec/app/ics-027-interchain-accounts/README.md#definitions) in ICS 27 spec.

## Features

### 1 - Registration

| ID  | Description | Verification | Status | Release |
| --- | ----------- | ------------ | ------ | ------- |
| 1.01 | An application shall have the ability to use an RPC endpoint to create interchain accounts on the host chain. | [Acceptance test](https://github.com/cosmos/ibc-go/blob/v6.0.0/modules/apps/27-interchain-accounts/controller/keeper/msg_server_test.go#L31) | `Verified` | v6.0.0 |

### 2 - Control

| ID  | Description | Verification | Status | Release |
| --- | ----------- | ------------ | ------ | ------- |
| 2.01 | An application shall have the ability to use an RPC endpoint to submit transactions to be executed on the host chain on the behalf of the interchain account. | [Acceptance test](https://github.com/cosmos/ibc-go/blob/v6.0.0/modules/apps/27-interchain-accounts/controller/keeper/msg_server_test.go#L31) | `Verified` | v6.0.0 |

# Non-functional requirements

## 3 - Migration

| ID | Description | Verification | Status | Release |
| -- | ----------- | ------------ | ------ | ------- |     
| 3.01 | Chains shall be able to run a migration to assign ownership of the channel capability of a custom authentication module to the ICS 27 controller submodule. | [Acceptance test](https://github.com/cosmos/ibc-go/blob/v6.0.0/modules/apps/27-interchain-accounts/controller/migrations/v6/migrations_test.go#L89) | `Verified` | v6.0.0 |

# External interface requirements

## 4 - CLI

### Transaction

| ID | Description | Verification | Status | Release |
| -- | ----------- | ------------ | ------ | ------- |
| 4.01 | There shall be a CLI command available to generate the Interchain Accounts packet data required to send.  | [CLI](https://github.com/cosmos/ibc-go/blob/v6.0.0/modules/apps/27-interchain-accounts/host/client/cli/tx.go#L21) | `Verified` | v6.0.0 |
