<!--
order: 5
-->

# Messages

## `MsgRegisterInterchainAccount`

An Interchain Accounts channel handshake can be initated using `MsgRegisterInterchainAccount`:

```go
type MsgRegisterInterchainAccount struct {
  Owner        string
  ConnectionID string
  Version      string
}
```

This message is expected to fail if:

- `Owner` is invalid (see [24-host naming requirements](https://github.com/cosmos/ibc/blob/master/spec/core/ics-024-host-requirements/README.md#paths-identifiers-separators).
- `ConnectionID` is invalid (see [24-host naming requirements](https://github.com/cosmos/ibc/blob/master/spec/core/ics-024-host-requirements/README.md#paths-identifiers-separators)).

This message will send a fungible token to the counterparty chain represented by the counterparty Channel End connected to the Channel End with the identifiers `SourcePort` and `SourceChannel`.

The denomination provided for transfer should correspond to the same denomination represented on this chain. The prefixes will be added as necessary upon by the receiving chain.

## `MsgSendTx`

An Interchain Accounts transaction can be executed on a remote `host` chain by using `MsgSendTx` from the `controller` chain:

```go
type MsgSendTx struct {
  Owner           string
  ConnectionID    string
  PacketData      InterchainAccountPacketData 
  RelativeTimeout uint64
}
```

This message is expected to fail if:

- `Owner` is invalid (see [24-host naming requirements](https://github.com/cosmos/ibc/blob/master/spec/core/ics-024-host-requirements/README.md#paths-identifiers-separators).
- `ConnectionID` is invalid (see [24-host naming requirements](https://github.com/cosmos/ibc/blob/master/spec/core/ics-024-host-requirements/README.md#paths-identifiers-separators)).
- `PacketData` is invalid (see [24-host naming requirements](https://github.com/cosmos/ibc/blob/master/spec/core/ics-024-host-requirements/README.md#paths-identifiers-separators).
- `RelativeTimeout` is invalid (see [24-host naming requirements](https://github.com/cosmos/ibc/blob/master/spec/core/ics-024-host-requirements/README.md#paths-identifiers-separators).

This message will send a fungible token to the counterparty chain represented by the counterparty Channel End connected to the Channel End with the identifiers `SourcePort` and `SourceChannel`.

The denomination provided for transfer should correspond to the same denomination represented on this chain. The prefixes will be added as necessary upon by the receiving chain.

### Example integration

```go
// app.go

// Register the AppModule for the Interchain Accounts module and the authentication module
// Note: No `icaauth` exists, this must be substituted with an actual Interchain Accounts authentication module
ModuleBasics = module.NewBasicManager(
    ...
    ica.AppModuleBasic{},
    icaauth.AppModuleBasic{},
    ...
)
```