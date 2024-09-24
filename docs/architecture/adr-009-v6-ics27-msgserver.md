# ADR 009: ICS27 message server addition

## Changelog

- 2022/09/07: Initial draft

## Status

Accepted, implemented in v6 of ibc-go

## Context

ICS 27 (Interchain Accounts) brought a cross-chain account management protocol built upon IBC.
It enabled chains to programmatically create accounts on behalf of counterparty chains which may enable a variety of authentication methods for this interchain account.
The initial release of ICS 27 focused on enabling authentication schemes that may not require signing with a private key, such as via on-chain mechanisms like governance.

Following the initial release of ICS 27 it became evident that:

- a default authentication module would enable more usage of ICS 27
- generic authentication modules should be capable of authenticating an interchain account registration
- application logic which wraps ICS 27 packet sends does not need to be associated with the authentication logic

## Decision

The controller module should be simplified to remove the correlation between the authentication logic for an interchain account and the application logic for an interchain account.
To minimize disruption to developers working on the original design of the ICS 27 controller module, all changes will be made in a backwards compatible fashion.

### Msg server

To achieve this, as stated by [@damiannolan](https://github.com/cosmos/ibc-go/issues/2026#issue-1341640594), it was proposed to:

> Add a new `MsgServer` to `27-interchain-accounts` which exposes two distinct rpc endpoints:
>
> - `RegisterInterchainAccount`
> - `SendTx`

This will enable any SDK (authentication) module to register interchain accounts and send transactions on their behalf.
Examples of existing SDK modules which would benefit from this change include:

- x/auth
- x/gov
- x/group

The existing go functions: `RegisterInterchainAccount()` and `SendTx()` will remain to operate as they did in previous release versions.

This will be possible for SDK v0.46.x and above.

### Allow `nil` underlying applications

Authentication modules should interact with the controller module via the message server and should not be associated with application logic.
For now, it will be allowed to set a `nil` underlying application.
A future version may remove the underlying application entirely.

See issue [#2040](https://github.com/cosmos/ibc-go/issues/2040)

### Channel capability claiming

The controller module will now claim the channel capability in `OnChanOpenInit`.
Underlying applications will be passed a `nil` capability in `OnChanOpenInit`.

Channel capability migrations will be added in two steps:

- Upgrade handler migration which modifies the channel capability owner from the underlying app to the controller module
- ICS 27 module automatic migration which asserts the upgrade handler channel capability migration has been performed successfully

See issue [#2033](https://github.com/cosmos/ibc-go/issues/2033)

### Middleware enabled channels

In order to maintain backwards compatibility and avoid requiring underlying application developers to account for interchain accounts they did not register, a boolean mapping has been added to track the behaviour of how an account was created.

If the account was created via the legacy API, then the underlying application callbacks will be executed.

If the account was created with the new API (message server), then the underlying application callbacks will not be executed.

See issue [#2145](https://github.com/cosmos/ibc-go/issues/2145)

### Future considerations

[ADR 008](https://github.com/cosmos/ibc-go/pull/1976) proposes the creation of a middleware which enables callers of an IBC packet send to perform application logic in conjunction with the IBC application.
The underlying application can be removed at the availability of such a middleware as that will be the preferred method for executing application logic upon a ICS 27 packet send.

### Miscellaneous

In order to avoid import cycles, the genesis types have been moved to their own directory.
A new protobuf package has been created for the genesis types.

See PR [#2133](https://github.com/cosmos/ibc-go/pull/2133)

An additional field has been added to the `ActiveChannel` type to store the `IsMiddlewareEnabled` field upon genesis import/export.

See issue [#2165](https://github.com/cosmos/ibc-go/issues/2165)

## Consequences

### Positive

- default authentication modules are provided (x/auth, x/group, x/gov)
- any SDK authentication module may now be used with ICS 27
- separation of authentication from application logic in relation to ICS 27
- minimized disruption to existing development around ICS 27 controller module
- underlying applications no longer have to handle capabilities
- removal of the underlying application upon the creation of ADR 008 may be done in a minimally disruptive fashion
- only underlying applications which registered the interchain account will perform application logic for that account (underlying applications do not need to be aware of accounts they did not register)

### Negative

- the security model has been reduced to that of the SDK. SDK modules may send packets for any interchain account.
- additional maintenance of the messages added and the middleware enabled flag
- underlying applications which will become ADR 008 modules are not required to be aware of accounts they did not register
- calling legacy API vs the new API results in different behaviour for ICS 27 application stacks which have an underlying application

### Neutral

- A major release is required
