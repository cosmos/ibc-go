# ADR 006: ICS-02 client refactor

## Changelog

- 2022-08-01: Initial Draft

## Status

Accepted and applied in v7 of ibc-go

## Context

During the initial development of the 02-client submodule, each light client supported (06-solomachine, 07-tendermint, 09-localhost) was referenced through hardcoding.
Here is an example of the [code](https://github.com/cosmos/cosmos-sdk/commit/b93300288e3a04faef9c0774b75c13b24450ba1c#diff-c5f6b956947375f28d611f18d0e670cf28f8f305300a89c5a9b239b0eeec5064R83) that existed in the 02-client submodule:

```go
func (k Keeper) UpdateClient(ctx sdk.Context, clientID string, header exported.Header) (exported.ClientState, error) {
  ...

  switch clientType {
  case exported.Tendermint:
    clientState, consensusState, err = tendermint.CheckValidityAndUpdateState(
    clientState, header, ctx.BlockTime(),
    )
  case exported.Localhost:
    // override client state and update the block height
    clientState = localhosttypes.NewClientState(
    ctx.ChainID(), // use the chain ID from context since the client is from the running chain (i.e self).
    ctx.BlockHeight(),
    )
  default:
    err = types.ErrInvalidClientType
  }
```

To add additional light clients, code would need to be added directly to the 02-client submodule.
Evidently, this would likely become problematic as IBC scaled to many chains using consensus mechanisms beyond the initial supported light clients.
Issue [#6064](https://github.com/cosmos/cosmos-sdk/issues/6064) on the SDK addressed this problem by creating a more modular 02-client submodule.
The 02-client submodule would now interact with each light client via an interface.
While, this change was positive in development, increasing the flexibility and adoptability of IBC, it also opened the door to new problems.

The difficulty of generalizing light clients became apparent once changes to those light clients were required.
Each light client represents a different consensus algorithm which may contain a host of complexity and nuances.
Here are some examples of issues which arose for light clients that are not applicable to all the light clients supported (06-solomachine, 07-tendermint, 09-localhost):

### Tendermint non-zero height upgrades

Before the launch of IBC, it was determined that the golang implementation of [tendermint](https://github.com/tendermint/tendermint) would not be capable of supporting non-zero height upgrades.
This implies that any upgrade would require changing of the chain ID and resetting the height to 0.
A chain is uniquely identified by its chain-id and validator set.
Two different chain ID's can be viewed as different chains and thus a normal update produced by a validator set cannot change the chain ID.
To work around the lack of support for non-zero height upgrades, an abstract height type was created along with an upgrade mechanism.
This type would indicate the revision number (the number of times the chain ID has been changed) and revision height (the current height of the blockchain).

Refs:

- Issue [#439](https://github.com/cosmos/ibc/issues/439) on IBC specification repository.
- Specification changes in [#447](https://github.com/cosmos/ibc/pull/447)
- Implementation changes for the abstract height type, [SDK#7211](https://github.com/cosmos/cosmos-sdk/pull/7211)

### Tendermint requires misbehaviour detection during updates

The initial release of the IBC module and the 07-tendermint light client implementation did not support misbehaviour detection during update nor did it prevent overwriting of previous updates.
Despite the fact that we designed the `ClientState` interface and developed the 07-tendermint client, we failed to detect even a duplicate update that constituted misbehaviour and thus should freeze the client.
This was fixed in PR [#141](https://github.com/cosmos/ibc-go/pull/141) which required light client implementations to be aware that they must handle duplicate updates and misbehaviour detection.
Misbehaviour detection during updates is not applicable to the solomachine nor localhost.
It is also not obvious that `CheckHeaderAndUpdateState` should be performing this functionality.

### Localhost requires access to the entire client store

The localhost has been broken since the initial version of the IBC module.
The localhost tried to be developed underneath the 02-client interfaces without special exception, but this proved to be impossible.
The issues were outlined in [#27](https://github.com/cosmos/ibc-go/issues/27) and further discussed in the attempted ADR in [#75](https://github.com/cosmos/ibc-go/pull/75).
Unlike all other clients, the localhost requires access to the entire IBC store and not just the prefixed client store.

### Solomachine doesn't set consensus states

The 06-solomachine does not set the consensus states within the prefixed client store.
It has a single consensus state that is stored within the client state.
This causes setting of the consensus state at the 02-client level to use unnecessary storage.
It also causes timeouts to fail with solo machines.
Previously, the timeout logic within IBC would obtain the consensus state at the height a timeout is being proved.
This is problematic for the solo machine as no consensus state is set.
See issue [#562](https://github.com/cosmos/ibc/issues/562) on the IBC specification repo.

### New clients may want to do batch updates

New light clients may not function in a similar fashion to 06-solomachine and 07-tendermint.
They may require setting many consensus states in a single update.
As @seunlanlege [states](https://github.com/cosmos/ibc-go/issues/284#issuecomment-1005583679):

> I'm in support of these changes for 2 reasons:
>
> - This would allow light clients to handle batch header updates in CheckHeaderAndUpdateState, for the special case of 11-beefy proving the finality for a batch of headers is much more space and time efficient than the space/time complexity of proving each individual headers in that batch, combined.
>
> - This also allows for a single light client instance of 11-beefy be used to prove finality for every parachain connected to the relay chain (Polkadot/Kusama). We achieve this by setting the appropriate ConsensusState for individual parachain headers in CheckHeaderAndUpdateState

## Decision

### Require light clients to set client and consensus states

The IBC specification states:

> If the provided header was valid, the client MUST also mutate internal state to store now-finalised consensus roots and update any necessary signature authority tracking (e.g. changes to the validator set) for future calls to the validity predicate.

The initial version of the IBC go SDK based module did not fulfill this requirement.
Instead, the 02-client submodule required each light client to return the client and consensus state which should be updated in the client prefixed store.
This decision lead to the issues "Solomachine doesn't set consensus states" and "New clients may want to do batch updates".

Each light client should be required to set its own client and consensus states on any update necessary.
The go implementation should be changed to match the specification requirements.
This will allow more flexibility for light clients to manage their own internal storage and do batch updates.

### Merge `Header`/`Misbehaviour` interface and rename to `ClientMessage`

Remove `GetHeight()` from the header interface (as light clients now set the client/consensus states).
This results in the `Header`/`Misbehaviour` interfaces being the same.
To reduce complexity of the codebase, the `Header`/`Misbehaviour` interfaces should be merged into `ClientMessage`.
`ClientMessage` will provide the client with some authenticated information which may result in regular updates, misbehaviour detection, batch updates, or other custom functionality a light client requires.

### Split `CheckHeaderAndUpdateState` into 4 functions

See [#668](https://github.com/cosmos/ibc-go/issues/668).

Split `CheckHeaderAndUpdateState` into 4 functions:

- `VerifyClientMessage`
- `CheckForMisbehaviour`
- `UpdateStateOnMisbehaviour`
- `UpdateState`

`VerifyClientMessage` checks the that the structure of a `ClientMessage` is correct and that all authentication data provided is valid.

`CheckForMisbehaviour` checks to see if a `ClientMessage` is evidence of misbehaviour.

`UpdateStateOnMisbehaviour` freezes the client and updates its state accordingly.

`UpdateState` performs a regular update or a no-op on duplicate updates.

The code roughly looks like:

```go
func (k Keeper) UpdateClient(ctx sdk.Context, clientID string, header exported.Header) error {
  ...

  if err := clientState.VerifyClientMessage(clientMessage); err != nil {
    return err
  }
  
  foundMisbehaviour := clientState.CheckForMisbehaviour(clientMessage)
  if foundMisbehaviour {
    clientState.UpdateStateOnMisbehaviour(header)
    // emit misbehaviour event
    return 
  }
  
  clientState.UpdateState(clientMessage) // expects no-op on duplicate header
  // emit update event
  return
}
```

### Add `GetTimestampAtHeight` to the client state interface

By adding `GetTimestampAtHeight` to the ClientState interface, we allow light clients which do non-traditional consensus state/timestamp storage to process timeouts correctly.
This fixes the issues outlined for the solo machine client.

### Add generic verification functions

As the complexity and the functionality grows, new verification functions will be required for additional paths.
This was explained in [#684](https://github.com/cosmos/ibc/issues/684) on the specification repo.
These generic verification functions would be immediately useful for the new paths added in connection/channel upgradability as well as for custom paths defined by IBC applications such as Interchain Queries.
The old verification functions (`VerifyClientState`, `VerifyConnection`, etc) should be removed in favor of the generic verification functions.

## Consequences

### Positive

- Flexibility for light client implementations
- Well defined interfaces and their required functionality
- Generic verification functions
- Applies changes necessary for future client/connection/channel upgrabability features
- Timeout processing for solo machines
- Reduced code complexity

### Negative

- The refactor touches on sensitive areas of the ibc-go codebase
- Changing of established naming (`Header`/`Misbehaviour` to `ClientMessage`)

### Neutral

No notable consequences

## References

Issues:

- [#284](https://github.com/cosmos/ibc-go/issues/284)

PRs:

- [#1871](https://github.com/cosmos/ibc-go/pull/1871)
