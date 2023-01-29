<!--
order: 9
-->

# Implementing `CheckSubstituteAndUpdateState`

`CheckSubstituteAndUpdateState` will try to update the client with the state of the substitute client. [This type of governance proposal](https://ibc.cosmos.network/main/ibc/proposals.html) is typically used to recover an expired and frozen client, as it can recover the entire state and therefore all existing channels built on top of the client.

Prior to updating, this function must verify that:
- the substitute client is the same type as the subject client. For a reference implementation, please see the [Tendermint light client](https://github.com/cosmos/ibc-go/blob/02-client-refactor-beta1/modules/light-clients/07-tendermint/proposal_handle.go#L32).
- the provided substitute may be used to update the subject client. This may mean that certain parameters must remain unaltered. For example, a [valid substitute Tendermint light client](https://github.com/cosmos/ibc-go/blob/02-client-refactor-beta1/modules/light-clients/07-tendermint/proposal_handle.go#L84) must NOT change the chain ID, trust level, max clock drift, unbonding period, proof specs or upgrade path. Please note that `AllowUpdateAfterMisbehaviour` and `AllowUpdateAfterExpiry` have been deprecated (see ADR 026 for more information).

After these checks are performed, the function must [set the updated client and consensus states](https://github.com/cosmos/ibc-go/blob/02-client-refactor-beta1/modules/light-clients/07-tendermint/proposal_handle.go#L77) within the clientStore for the subject client.