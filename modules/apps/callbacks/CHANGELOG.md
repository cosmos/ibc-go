# Changelog

## [Unreleased]

### Dependencies

### API Breaking

### State Machine Breaking

### Improvements

### Features

### Bug Fixes

<!-- markdown-link-check-disable-next-line -->
## [v0.2.0+ibc-go-v8.0](https://github.com/cosmos/ibc-go/releases/tag/modules%2Fapps%2Fcallbacks%2Fv0.2.0%2Bibc-go-v8.0) - 2023-11-15

### Bug Fixes

* [\#4568](https://github.com/cosmos/ibc-go/pull/4568) Include error in event that is emitted when the callback cannot be executed due to a panic or an out of gas error. Packet is only sent if the `IBCSendPacketCallback` returns nil explicitly.

<!-- markdown-link-check-disable-next-line -->
## [v0.1.0+ibc-go-v7.3](https://github.com/cosmos/ibc-go/releases/tag/modules%2Fapps%2Fcallbacks%2Fv0.1.0%2Bibc-go-v7.3) - 2023-08-31

### Features

* [\#3939](https://github.com/cosmos/ibc-go/pull/3939) feat(callbacks): ADR8 implementation.
