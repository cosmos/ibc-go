---
title: Handling Upgrades
sidebar_label: Handling Upgrades
sidebar_position: 5
slug: /ibc/light-clients/upgrades
---


# Handling upgrades

It is vital that high-value IBC clients can upgrade along with their underlying chains to avoid disruption to the IBC ecosystem. Thus, IBC client developers will want to implement upgrade functionality to enable clients to maintain connections and channels even across chain upgrades.

## Implementing `VerifyUpgradeAndUpdateState`

The IBC protocol allows client implementations to provide a path to upgrading clients given the upgraded `ClientState`, upgraded `ConsensusState` and proofs for each. This path is provided in the `VerifyUpgradeAndUpdateState` method:

```go	
// NOTE: proof heights are not included as upgrade to a new revision is expected to pass only on the last height committed by the current revision. Clients are responsible for ensuring that the planned last height of the current revision is somehow encoded in the proof verification process.
// This is to ensure that no premature upgrades occur, since upgrade plans committed to by the counterparty may be cancelled or modified before the last planned height.
// If the upgrade is verified, the upgraded client and consensus states must be set in the client store.
func (cs ClientState) VerifyUpgradeAndUpdateState(
  ctx sdk.Context,
  cdc codec.BinaryCodec,
  store sdk.KVStore,
  newClient ClientState,
  newConsState ConsensusState,
  proofUpgradeClient,
  proofUpgradeConsState []byte,
) error
```

> Please refer to the [Tendermint light client implementation](https://github.com/cosmos/ibc-go/blob/02-client-refactor-beta1/modules/light-clients/07-tendermint/upgrade.go#L27) as an example for implementation.

It is important to note that light clients **must** handle all management of client and consensus states including the setting of updated `ClientState` and `ConsensusState` in the client store. This can include verifying that the submitted upgraded `ClientState` is of a valid `ClientState` type, that the height of the upgraded client is not greater than the height of the current client (in order to preserve BFT monotonic time), or that certain parameters which should not be changed have not been altered in the upgraded `ClientState`.

Developers must ensure that the `MsgUpgradeClient` does not pass until the last height of the old chain has been committed, and after the chain upgrades, the `MsgUpgradeClient` should pass once and only once on all counterparty clients.

### Upgrade path

Clients should have **prior knowledge of the merkle path** that the upgraded client and upgraded consensus states will use. The height at which the upgrade has occurred should also be encoded in the proof. 
> The Tendermint client implementation accomplishes this by including an `UpgradePath` in the `ClientState` itself, which is used along with the upgrade height to construct the merkle path under which the client state and consensus state are committed.

## Chain specific vs client specific client parameters

Developers should maintain the distinction between client parameters that are uniform across every valid light client of a chain (chain-chosen parameters), and client parameters that are customizable by each individual client (client-chosen parameters); since this distinction is necessary to implement the `ZeroCustomFields` method in the [`ClientState` interface](02-client-state.md):

```go
// Utility function that zeroes out any client customizable fields in client state
// Ledger enforced fields are maintained while all custom fields are zero values
// Used to verify upgrades
func (cs ClientState) ZeroCustomFields() ClientState
```

Developers must ensure that the new client adopts all of the new client parameters that must be uniform across every valid light client of a chain (chain-chosen parameters), while maintaining the client parameters that are customizable by each individual client (client-chosen parameters) from the previous version of the client. `ZeroCustomFields` is a useful utility function to ensure only chain specific fields are updated during upgrades.

## Security

Upgrades must adhere to the IBC Security Model. IBC does not rely on the assumption of honest relayers for correctness. Thus users should not have to rely on relayers to maintain client correctness and security (though honest relayers must exist to maintain relayer liveness). While relayers may choose any set of client parameters while creating a new `ClientState`, this still holds under the security model since users can always choose a relayer-created client that suits their security and correctness needs or create a client with their desired parameters if no such client exists.

However, when upgrading an existing client, one must keep in mind that there are already many users who depend on this client's particular parameters. **We cannot give the upgrading relayer free choice over these parameters once they have already been chosen. This would violate the security model** since users who rely on the client would have to rely on the upgrading relayer to maintain the same level of security.

Thus, developers must make sure that their upgrade mechanism allows clients to upgrade the chain-specified parameters whenever a chain upgrade changes these parameters (examples in the Tendermint client include `UnbondingPeriod`, `TrustingPeriod`, `ChainID`, `UpgradePath`, etc), while ensuring that the relayer submitting the `MsgUpgradeClient` cannot alter the client-chosen parameters that the users are relying upon (examples in Tendermint client include `TrustLevel`, `MaxClockDrift`, etc). The previous paragraph discusses how `ZeroCustomFields` helps achieve this.

### Document potential client parameter conflicts during upgrades

Counterparty clients can upgrade securely by using all of the chain-chosen parameters from the chain-committed `UpgradedClient` and preserving all of the old client-chosen parameters. This enables chains to securely upgrade without relying on an honest relayer, however it can in some cases lead to an invalid final `ClientState` if the new chain-chosen parameters clash with the old client-chosen parameter. This can happen in the Tendermint client case if the upgrading chain lowers the `UnbondingPeriod` (chain-chosen) to a duration below that of a counterparty client's `TrustingPeriod` (client-chosen). Such cases should be clearly documented by developers, so that chains know which upgrades should be avoided to prevent this problem. The final upgraded client should also be validated in `VerifyUpgradeAndUpdateState` before returning to ensure that the client does not upgrade to an invalid `ClientState`.
