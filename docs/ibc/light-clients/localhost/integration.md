<!--
order: 2
-->

# Integration

The 09-localhost light client module registers codec types within the core IBC module. This differs from other light client module implementations which are expected to register codec types using the `AppModuleBasic` interface.

The localhost client is added to the 02-client submodule param [`allowed_clients`](https://github.com/cosmos/ibc-go/blob/v7.0.0-rc0/proto/ibc/core/client/v1/client.proto#L102) by default in ibc-go.

```go
var (
  // DefaultAllowedClients are the default clients for the AllowedClients parameter.
  DefaultAllowedClients = []string{exported.Solomachine, exported.Tendermint, exported.Localhost}
)
```