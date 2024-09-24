# ADR 007: Solo machine sign bytes

## Changelog

- 2022-08-02: Initial draft

## Status

Accepted, applied in v7

## Context

The `06-solomachine` implementation up until ibc-go v7 constructed sign bytes using a `DataType` which described what type of data was being signed.
This design decision arose from a misunderstanding of the security implications.
It was noted that the proto definitions do not [provide uniqueness](https://github.com/cosmos/cosmos-sdk/pull/7237#discussion_r484264573) which is a necessity for ensuring two signatures over different data types can never be the same.
What was missed is that the uniqueness is not provided by the proto definition, but by the usage of the proto definition.
The path provided by core IBC will be unique and is already encoded into the signature data.
Thus two different paths with the same data values will encode differently which provides signature uniqueness.

Furthermore, the current construction does not support the proposed changes in the spec repo to support [Generic Verification functions](https://github.com/cosmos/ibc/issues/684).
This is because in order to verify a new path, a new `DataType` must be added for that path.

## Decision

Remove `DataType` and change the `DataType` in the `SignBytes` and `SignatureAndData` to be `Path`.
The new `Path` field should be bytes.
Remove all `...Data` proto definitions except for `HeaderData`
These `...Data` definitions were created previously for each `DataType`.
The proto version of the solo machine proto definitions should be bumped to `v3`.

This removes an extra layer of complexity from signature construction and allows for support of generic verification.

## Consequences

### Positive

- Simplification of solo machine signature construction
- Support for generic verification

### Negative

- Breaks existing signature construction in a non-backwards compatible way
- Solo machines must update to handle the new format
- Migration required for solo machine client and consensus states

### Neutral

No notable consequences

## References

- [#1141](https://github.com/cosmos/ibc-go/issues/1141)
