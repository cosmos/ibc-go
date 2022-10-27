<!--
order: 7
-->

# Parameters

The IBC transfer application module contains the following parameters:

| Key              | Type | Default Value |
|------------------|------|---------------|
| `SendEnabled`    | bool | `true`        |
| `ReceiveEnabled` | bool | `true`        |

## `SendEnabled`

The transfers enabled parameter controls send cross-chain transfer capabilities for all fungible tokens.

To prevent a single token from being transferred from the chain, set the `SendEnabled` parameter to `true` and then, depending on the Cosmos SDK version, do one of the following:

- For Cosmos SDK v0.46.x or earlier, set the bank module's [`SendEnabled` parameter](https://github.com/cosmos/cosmos-sdk/blob/release/v0.46.x/x/bank/spec/05_params.md#sendenabled) for the denomination to `false`.
- For Cosmos SDK versions above v0.46.x, set the bank module's `SendEnabled` entry for the denomination to `false` using `MsgSetSendEnabled` as a governance proposal.

## `ReceiveEnabled`

The transfers enabled parameter controls receive cross-chain transfer capabilities for all fungible tokens.

To prevent a single token from being transferred to the chain, set the `ReceiveEnabled` parameter to `true` and then, depending on the Cosmos SDK version, do one of the following:

- For Cosmos SDK v0.46.x or earlier, set the bank module's [`SendEnabled` parameter](https://github.com/cosmos/cosmos-sdk/blob/release/v0.46.x/x/bank/spec/05_params.md#sendenabled) for the denomination to `false`.
- For Cosmos SDK versions above v0.46.x, set the bank module's `SendEnabled` entry for the denomination to `false` using `MsgSetSendEnabled` as a governance proposal.
