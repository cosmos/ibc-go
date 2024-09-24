---
title: Setup
sidebar_label: Setup
sidebar_position: 9
slug: /ibc/light-clients/setup
---


# Setup

:::note Synopsis
Learn how to configure light client modules and create clients using core IBC and the `02-client` submodule. 
:::

A last step to finish the development of the light client, is to implement the `AppModuleBasic` interface to allow it to be added to the chain's `app.go` alongside other light client types the chain enables.

Finally, a succinct rundown is given of the remaining steps to make the light client operational, getting the light client type passed through governance and creating the clients.

## Configuring a light client module

An IBC light client module must implement the [`AppModuleBasic`](https://github.com/cosmos/cosmos-sdk/blob/main/types/module/module.go#L50) interface in order to register its concrete types against the core IBC interfaces defined in `modules/core/exported`. This is accomplished via the `RegisterInterfaces` method which provides the light client module with the opportunity to register codec types using the chain's `InterfaceRegistry`. Please refer to the [`07-tendermint` codec registration](https://github.com/cosmos/ibc-go/blob/02-client-refactor-beta1/modules/light-clients/07-tendermint/codec.go#L11).

The `AppModuleBasic` interface may also be leveraged to install custom CLI handlers for light client module users. Light client modules can safely no-op for interface methods which it does not wish to implement.

Please refer to the [core IBC documentation](../../01-ibc/02-integration.md#integrating-light-clients) for how to configure additional light client modules alongside `07-tendermint` in `app.go`.

See below for an example of the `07-tendermint` implementation of `AppModuleBasic`.

```go
var _ module.AppModuleBasic = AppModuleBasic{}

// AppModuleBasic defines the basic application module used by the tendermint light client.
// Only the RegisterInterfaces function needs to be implemented. All other function perform
// a no-op.
type AppModuleBasic struct{}

// Name returns the tendermint module name.
func (AppModuleBasic) Name() string {
  return ModuleName
}

// RegisterLegacyAminoCodec performs a no-op. The Tendermint client does not support amino.
func (AppModuleBasic) RegisterLegacyAminoCodec(*codec.LegacyAmino) {}

// RegisterInterfaces registers module concrete types into protobuf Any. This allows core IBC
// to unmarshal tendermint light client types.
func (AppModuleBasic) RegisterInterfaces(registry codectypes.InterfaceRegistry) {
  RegisterInterfaces(registry)
}

// DefaultGenesis performs a no-op. Genesis is not supported for the tendermint light client.
func (AppModuleBasic) DefaultGenesis(cdc codec.JSONCodec) json.RawMessage {
  return nil
}

// ValidateGenesis performs a no-op. Genesis is not supported for the tendermint light client.
func (AppModuleBasic) ValidateGenesis(cdc codec.JSONCodec, config client.TxEncodingConfig, bz json.RawMessage) error {
  return nil
}

// RegisterGRPCGatewayRoutes performs a no-op.
func (AppModuleBasic) RegisterGRPCGatewayRoutes(clientCtx client.Context, mux *runtime.ServeMux) {}

// GetTxCmd performs a no-op. Please see the 02-client cli commands.
func (AppModuleBasic) GetTxCmd() *cobra.Command {
  return nil
}

// GetQueryCmd performs a no-op. Please see the 02-client cli commands.
func (AppModuleBasic) GetQueryCmd() *cobra.Command {
  return nil
}
```

## Creating clients

A client is created by executing a new `MsgCreateClient` transaction composed with a valid `ClientState` and initial `ConsensusState` encoded as protobuf `Any`s.
Generally, this is performed by an off-chain process known as an [IBC relayer](https://github.com/cosmos/ibc/tree/main/spec/relayer/ics-018-relayer-algorithms) however, this is not a strict requirement.

See below for a list of IBC relayer implementations:

- [cosmos/relayer](https://github.com/cosmos/relayer)
- [informalsystems/hermes](https://github.com/informalsystems/hermes)
- [confio/ts-relayer](https://github.com/confio/ts-relayer)

Stateless checks are performed within the [`ValidateBasic`](https://github.com/cosmos/ibc-go/blob/02-client-refactor-beta1/modules/core/02-client/types/msgs.go#L48) method of `MsgCreateClient`.

```protobuf
// MsgCreateClient defines a message to create an IBC client
message MsgCreateClient {
  option (gogoproto.equal)           = false;
  option (gogoproto.goproto_getters) = false;

  // light client state
  google.protobuf.Any client_state = 1 [(gogoproto.moretags) = "yaml:\"client_state\""];
  // consensus state associated with the client that corresponds to a given
  // height.
  google.protobuf.Any consensus_state = 2 [(gogoproto.moretags) = "yaml:\"consensus_state\""];
  // signer address
  string signer = 3;
}
```

Leveraging protobuf `Any` encoding allows core IBC to [unpack](https://github.com/cosmos/ibc-go/blob/02-client-refactor-beta1/modules/core/keeper/msg_server.go#L28-L36) both the `ClientState` and `ConsensusState` into their respective interface types registered previously using the light client module's `RegisterInterfaces` method.

Within the `02-client` submodule, the [`ClientState` is then initialized](https://github.com/cosmos/ibc-go/blob/02-client-refactor-beta1/modules/core/02-client/keeper/client.go#L30-L34) with its own isolated key-value store, namespaced using a unique client identifier.
 
In order to successfully create an IBC client using a new client type, it [must be supported](https://github.com/cosmos/ibc-go/blob/02-client-refactor-beta1/modules/core/02-client/keeper/client.go#L18-L24). Light client support in IBC is gated by on-chain governance. The allow list may be updated by submitting a new governance proposal to update the `02-client` parameter `AllowedClients`.

<!-- 
- TODO: update when params are managed by ibc-go 
- https://github.com/cosmos/ibc-go/issues/2010
-->
See below for example:

```shell
%s tx gov submit-proposal param-change <path/to/proposal.json> --from=<key_or_address>
```

where `proposal.json` contains:

```json
{
  "title": "IBC Clients Param Change",
  "description": "Update allowed clients",
  "changes": [
    {
      "subspace": "ibc",
      "key": "AllowedClients",
      "value": ["06-solomachine", "07-tendermint", "0x-new-client"]
    }
  ],
  "deposit": "1000stake"
}
```
