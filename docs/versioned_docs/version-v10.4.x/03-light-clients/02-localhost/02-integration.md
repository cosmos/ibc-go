---
title: Integration
sidebar_label: Integration
sidebar_position: 2
slug: /ibc/light-clients/localhost/integration
---


# Integration

The 09-localhost light client module registers codec types within the core IBC module. This differs from other light client module implementations which are expected to register codec types using the `AppModuleBasic` interface.

The localhost client is implicitly enabled by using the `AllowAllClients` wildcard (`"*"`) in the 02-client submodule default value for param  [`allowed_clients`](https://github.com/cosmos/ibc-go/blob/v7.0.0/proto/ibc/core/client/v1/client.proto#L102).

```go
// DefaultAllowedClients are the default clients for the AllowedClients parameter.
// By default it allows all client types.
var DefaultAllowedClients = []string{AllowAllClients}
```
