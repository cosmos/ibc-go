# ADR 27: Add support for Wasm based light client

## Changelog

- 26/11/2020: Initial Draft
- 26/05/2023: Update after 02-client refactor and re-implementation by Strangelove
- 13/12/2023: Update after upstreaming of module to ibc-go

## Status

*Accepted and applied in v0.1.0 of 08-wasm*

## Abstract

In the Cosmos SDK light clients are currently hardcoded in Go. This makes upgrading existing IBC light clients or
adding support for new light client a multi step process involving on-chain governance which is time-consuming.

To remedy this, we are proposing a Wasm VM to host light client bytecode, which allows easier upgrading of
existing IBC light clients as well as adding support for new IBC light clients without requiring a code release and 
corresponding hard-fork event.

## Context

Currently in ibc-go light clients are defined as part of the codebase and are implemented as modules under
`modules/light-clients`. Adding support for new light clients or updating an existing light client in the event
of a security issue or consensus update is a multi-step process which is both time-consuming and error-prone. 
In order to enable new IBC light client implementations it is necessary to modify the codebase of ibc-go (if the light
client is part of its codebase), re-build chains' binaries, pass a governance proposal and validators upgrade their nodes.

Another problem stemming from the above process is that if a chain wants to upgrade its own consensus, it will 
need to convince every chain or hub connected to it to upgrade its light client in order to stay connected. Due 
to the time-consuming process required to upgrade a light client, a chain with lots of connections needs to be 
disconnected for quite some time after upgrading its consensus, which can be very expensive in terms of time and effort.

We are proposing simplifying this workflow by integrating a Wasm light client module that makes adding support for
new light clients a simple governance-gated transaction. The light client bytecode, written in Wasm-compilable Rust, 
runs inside a Wasm VM. The Wasm light client submodule exposes a proxy light client interface that routes incoming 
messages to the appropriate handler function, inside the Wasm VM for execution.

With the Wasm light client module, anybody can add new IBC light client in the form of Wasm bytecode (provided they are 
able to submit the governance proposal transaction and that it passes) as well as instantiate clients using any created 
client type. This allows any chain to update its own light client in other chains without going through the steps outlined above.

## Decision

We decided to implement the Wasm light client module as a light client proxy that will interface with the actual light client
uploaded as Wasm bytecode. To enable usage of the Wasm light client module, users need to add it to the list of allowed clients
by updating the `AllowedClients` parameter in the 02-client submodule of core IBC.

```go
params := clientKeeper.GetParams(ctx)
params.AllowedClients = append(params.AllowedClients, exported.Wasm)
clientKeeper.SetParams(ctx, params)
```

Adding a new light client contract is governance-gated. To upload a new light client users need to submit 
a [governance v1 proposal](https://docs.cosmos.network/main/modules/gov#proposals) that contains the `sdk.Msg` for storing 
the Wasm contract's bytecode. The required message is `MsgStoreCode` and the bytecode is provided in the field `wasm_byte_code`:

```proto
// MsgStoreCode defines the request type for the StoreCode rpc.
message MsgStoreCode {
  // signer address
  string signer = 1;
  // wasm byte code of light client contract. It can be raw or gzip compressed
  bytes wasm_byte_code = 2;
}
```

The RPC handler processing `MsgStoreCode` will make sure that the signer of the message matches the address of authority allowed to 
submit this message (which is normally the address of the governance module).

```go
// StoreCode defines a rpc handler method for MsgStoreCode
func (k Keeper) StoreCode(goCtx context.Context, msg *types.MsgStoreCode) (*types.MsgStoreCodeResponse, error) {
  if k.GetAuthority() != msg.Signer {
    return nil, errorsmod.Wrapf(ibcerrors.ErrUnauthorized, "expected %s, got %s", k.GetAuthority(), msg.Signer)
  }

  ctx := sdk.UnwrapSDKContext(goCtx)
  checksum, err := k.storeWasmCode(ctx, msg.WasmByteCode, ibcwasm.GetVM().StoreCode)
  if err != nil {
    return nil, errorsmod.Wrap(err, "failed to store wasm bytecode")
  }

  emitStoreWasmCodeEvent(ctx, checksum)

  return &types.MsgStoreCodeResponse{
    Checksum: checksum,
  }, nil
}
```

The contract's bytecode is not stored in state (it is actually unnecessary and wasteful to store it, since
the Wasm VM already stores it and can be queried back, if needed). The checksum is simply the hash of the bytecode
of the contract and it is stored in state in an entry with key `checksums` that contains the checksums for the bytecodes that have been stored.

### How light client proxy works?

The light client proxy behind the scenes will call a CosmWasm smart contract instance with incoming arguments serialized 
in JSON format with appropriate environment information. Data returned by the smart contract is deserialized and
returned to the caller.

Consider the example of the `VerifyClientMessage` function of `ClientState` interface. Incoming arguments are
packaged inside a payload object that is then JSON serialized and passed to `queryContract`, which executes `WasmVm.Query` 
and returns the slice of bytes returned by the smart contract. This data is deserialized and passed as return argument.

```go
type QueryMsg struct {
  Status               *StatusMsg               `json:"status,omitempty"`
  ExportMetadata       *ExportMetadataMsg       `json:"export_metadata,omitempty"`
  TimestampAtHeight    *TimestampAtHeightMsg    `json:"timestamp_at_height,omitempty"`
  VerifyClientMessage  *VerifyClientMessageMsg  `json:"verify_client_message,omitempty"`
  CheckForMisbehaviour *CheckForMisbehaviourMsg `json:"check_for_misbehaviour,omitempty"`
}

type verifyClientMessageMsg struct {
  ClientMessage *ClientMessage `json:"client_message"`
}

// VerifyClientMessage must verify a ClientMessage. 
// A ClientMessage could be a Header, Misbehaviour, or batch update.
// It must handle each type of ClientMessage appropriately. 
// Calls to CheckForMisbehaviour, UpdateStaåte, and UpdateStateOnMisbehaviour
// will assume that the content of the ClientMessage has been verified
// and can be trusted. An error should be returned
// if the ClientMessage fails to verify.
func (cs ClientState) VerifyClientMessage(
  ctx sdk.Context,
  _ codec.BinaryCodec,
  clientStore storetypes.KVStore,
  clientMsg exported.ClientMessage
) error {
  clientMessage, ok := clientMsg.(*ClientMessage)
  if !ok {
    return errorsmod.Wrapf(ibcerrors.ErrInvalidType, "expected type: %T, got: %T", &ClientMessage{}, clientMsg)
  }

  payload := QueryMsg{
    VerifyClientMessage: &VerifyClientMessageMsg{ClientMessage: clientMessage.Data},
  }
  _, err := wasmQuery[EmptyResult](ctx, clientStore, &cs, payload)
  return err
}
```

### Global Wasm VM variable

The 08-wasm keeper structure keeps a reference to the Wasm VM instantiated in the keeper constructor function. The keeper uses 
the Wasm VM to store the bytecode of light client contracts. However, the Wasm VM is also needed in the 08-wasm implementations of
some of the `ClientState` interface functions to initialise a contract, execute calls on the contract and query the contract. Since
the `ClientState` functions do not have access to the 08-wasm keeper, then it has been decided to keep a global pointer variable that
points to the same instance as the one in the 08-wasm keeper. This global pointer variable is then used in the implementations of
the `ClientState` functions. 

## Consequences

### Positive

- Adding support for new light client or upgrading existing light client is way easier than before and only requires single transaction instead of a hard-fork.
- Improves maintainability of ibc-go, since no change in codebase is required to support new client or upgrade it.
- The existence of support for Rust dependencies in light clients which may not exist in Go.

### Negative

- Light clients written in Rust need to be written in a subset of Rust which could compile in Wasm.
- Introspecting light client code is difficult as only compiled bytecode exists in the blockchain.
