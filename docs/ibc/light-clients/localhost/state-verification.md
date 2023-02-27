<!--
order: 5
-->

# State verification

The localhost client handles state verification through the `ClientState` interface methods `VerifyMembership` and `VerifyNonMembership` by performing read-only operations directly on the core IBC store.

When verifying channel state in handshakes or processing packets the `09-localhost` client can simply compare bytes stored under the standardized key paths defined by [ICS-24](https://github.com/cosmos/ibc/tree/main/spec/core/ics-024-host-requirements).

For existence proofs via `VerifyMembership` the 09-localhost client will retrieve the value stored under the provided key path and compare it against the value provided by the caller. In contrast, non-existence proofs via `VerifyNonMembership` assert the absence of a value at the provided key path.
