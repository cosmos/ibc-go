<!--
order: 2
-->

# Integration into Cosmos SDK chain

The `09-localhost` light client module can be integrated into a Cosmos SDK chain by simply registering the `AppModuleBasic` in `app.go` and adding the `09-localhost` client type to the [`allowed_clients`](https://github.com/cosmos/ibc-go/blob/v7.0.0-rc0/proto/ibc/core/client/v1/client.proto#L102) list as defined by the 02-client submodule on-chain parameters.

```go
import (
  // ...
  localhost "github.com/cosmos/ibc-go/v7/modules/light-clients/09-localhost"
)

// ...

ModuleBasics = module.NewBasicManager(
  ...
  ibc.AppModuleBasic{},
  localhost.AppModuleBasic{},
  ...
)
```

Note that the localhost client is added to `allowed_clients` by default in ibc-go/v7.1.

```go
var (
  // DefaultAllowedClients are "06-solomachine", "07-tendermint" and "09-localhost"
  DefaultAllowedClients = []string{exported.Solomachine, exported.Tendermint, exported.Localhost}
)
```