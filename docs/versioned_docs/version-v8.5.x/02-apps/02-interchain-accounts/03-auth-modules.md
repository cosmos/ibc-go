---
title: Authentication Modules
sidebar_label: Authentication Modules
sidebar_position: 3
slug: /apps/interchain-accounts/auth-modules
---


# Building an authentication module

:::note Synopsis
Authentication modules enable application developers to perform custom logic when interacting with the Interchain Accounts controller sumbmodule's `MsgServer`.
:::

The controller submodule is used for account registration and packet sending. It executes only logic required of all controllers of interchain accounts. The type of authentication used to manage the interchain accounts remains unspecified. There may exist many different types of authentication which are desirable for different use cases. Thus the purpose of the authentication module is to wrap the controller submodule with custom authentication logic.

In ibc-go, authentication modules can communicate with the controller submodule by passing messages through `baseapp`'s `MsgServiceRouter`. To implement an authentication module, the `IBCModule` interface need not be fulfilled; it is only required to fulfill Cosmos SDK's `AppModuleBasic` interface, just like any regular Cosmos SDK application module.

The authentication module must:

- Authenticate interchain account owners.
- Track the associated interchain account address for an owner.
- Send packets on behalf of an owner (after authentication).

## Integration into `app.go` file

To integrate the authentication module into your chain, please follow the steps outlined in [`app.go` integration](04-integration.md#example-integration).
