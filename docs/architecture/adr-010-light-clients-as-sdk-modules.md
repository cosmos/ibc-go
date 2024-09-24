# ADR 010: IBC light clients as SDK modules

## Changelog

- 12/12/2022: initial draft

## Status

Proposed

## Context

ibc-go has 3 main consumers:

- IBC light clients
- IBC applications
- relayers

Relayers listen and respond to events emitted by ibc-go while IBC light clients and applications are invoked by core IBC.
Currently there exists two different approaches to callbacks being invoked by core IBC.

IBC light clients currently are invoked by a `ClientState` and `ConsensusState` interface as defined by [core IBC](https://github.com/cosmos/ibc-go/blob/v7.0.0/modules/core/exported/client.go#L36).
The 02-client submodule will retrieve the `ClientState` or `ConsensusState` from the IBC store in order to perform callbacks to the light client.
This design requires all required information for the light client to function to be stored in the `ClientState` or `ConsensusState` or potentially under metadata keys for a specific client instance.
Additional information may be provided by core IBC via the defined interface arguments if that information is generic enough to be useful to all IBC light clients.
This constraint has proved problematic as pass through clients (such as wasm) cannot maintain easy access to a VM instance.
In addition, without increasing the size of the defined `ClientState` interface, light clients are unable to take advantage of basic built-in SDK functionality such as genesis import/export and migrations.

The other approach used to perform callback logic is via registered SDK modules.
This approach is used by core IBC to interact with IBC applications.
IBC applications will register their callbacks on the IBC router at compile time.
When a packet comes in, core IBC will use the IBC router to lookup the registered callback functions for the provided packet.
The benefit of registered callbacks opposed to interface functions is that additional information may be accessed via external keepers.
Because the IBC applications are also SDK modules, they additionally get access to a host of functionality provided by the SDK.
This includes: genesis import/export, migrations, query/transaction CLI commands, type registration, gRPC query registration, and message server registration.

As described in [ADR 006](./adr-006-02-client-refactor.md), generalizing light client behaviour is difficult.
IBC light clients will obtain greater flexibility and control via the registered SDK module approach.

## Decision

Instead of using two different approaches to invoking callbacks, IBC light clients should be invoked as SDK modules.
Over time and as necessary, core IBC should adjust its interactions with light clients such that they are SDK modules as opposed to interfaces.

One immediate decision that has already been applied is to formalize light client type registration via the inclusion of an `AppModuleBasic` within the `ModuleManager` for a chain.
The [tendermint](https://github.com/cosmos/ibc-go/pull/2825) and [solo machine](https://github.com/cosmos/ibc-go/pull/2826) clients were refactored to include this `AppModuleBasic` implementation and core IBC will no longer include either type as registered by default.

Longer term solutions include using internal module communication as described in [ADR 033](https://github.com/cosmos/cosmos-sdk/blob/main/docs/architecture/adr-033-protobuf-inter-module-comm.md) on the SDK.
The following functions should become callbacks invoked via intermodule communication:

- `Status`
- `GetTimestampAtHeight`
- `VerifyMembership`
- `VerifyNonMembership`
- `Initialize`
- `VerifyClientMessage`
- `CheckForMisbehaviour`
- `UpdateStateOnMisbehaviour`
- `UpdateState`
- `CheckSubstituteAndUpdateState`
- `VerifyUpgradeAndUpdateState`

The ClientState interface should eventually be trimmed down to something along the lines of:

```go
type ClientState interface {
    proto.Message

    ClientType() string
    GetLatestHeight() Height
    Validate() error

    ZeroCustomFields() ClientState

    // ADDITION
    Route() string // route used for intermodule communication
}
```

For the most part, any functions which require access to the client store should likely not be an interface function of the `ClientState`.

`ExportMetadata` should eventually be replaced by a light client's ability to import/export it's own genesis information.

### Intermodule communication

To keep the transition from interface callbacks to SDK module callbacks as simple as possible, intermodule communication (when available) should be used to route to light client modules.
Without intermodule communication, a routing system would need to be developed/maintained to register callbacks.
This functionality of routing to another SDK module should and will be provided by the SDK.
Once it is possible to route to SDK modules, a `ClientState` type could expose the function `Route` which returns the callback route used to call the light client module.

## Consequences

### Positive

- use a single approach for interacting with callbacks
- greater flexibility and control for IBC light clients
- does not require developing another routing system

### Negative

- requires breaking changes
- requires waiting for intermodule communication

### Neutral

N/A
