<!--
order: 7
-->

# Implementing `VerifyUpgradeAndUpdateState`

It is vital that high-value IBC clients can upgrade along with their underlying chains to avoid disruption to the IBC ecosystem. Thus, IBC client developers will want to implement upgrade functionality to enable clients to maintain connections and channels even across chain upgrades.

The IBC protocol allows client implementations to provide a path to upgrading clients given the upgraded client state, upgraded consensus state and proofs for each. This path is provided in the `VerifyUpgradeAndUpdateState` function:

```golang	
// NOTE: proof heights are not included as upgrade to a new revision is expected to pass only on the last height committed by the current revision. Clients are responsible for ensuring that the planned last height of the current revision is somehow encoded in the proof verification process.
// This is to ensure that no premature upgrades occur, since upgrade plans committed to by the counterparty may be cancelled or modified before the last planned height.
// If the upgrade is verified, the upgraded client and consensus states must be set in the client store.
VerifyUpgradeAndUpdateState(
    ctx sdk.Context,
    cdc codec.BinaryCodec,
    store sdk.KVStore,
    newClient ClientState,
    newConsState ConsensusState,
    proofUpgradeClient,
    proofUpgradeConsState []byte,
) error
```

It is important to note that light clients **must** handle all management of client and consensus states including the setting of updated client state and consensus state in the client store. This can include verifying that the submitted upgraded `ClientState` is of a valid `ClientState` type, that the height of the upgraded client is not greater than the height of the current client (in order to preserve BFT monotonic time), or that certain parameters which should not be changed have not been altered in the upgraded client state.

Please refer to the [`07-tendermint` implementation](https://github.com/cosmos/ibc-go/blob/02-client-refactor-beta1/modules/light-clients/07-tendermint/upgrade.go#L27) as an example.

