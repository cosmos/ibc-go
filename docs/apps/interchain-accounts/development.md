<!--
order: 5
-->

# Development use cases

The initial version of interchain accounts allowed for the controller module to be extended by providing it with an underlying application which would handle all packet callbacks.
That functionality is now being deprecated in favor of alternative approaches. 
This document will outline potential use cases and redirect each use case to the appropriate documentation. 

## Custom authentication 

Interchain accounts may be associated with alternative types of authentication relative to the traditional public/private key signing. 
If you wish to develop or use interchain accounts with a custom authentication module, we recommend you use ibc-go v6 or greater. 

The custom authentication module should interact with the controller module via the [MsgServer](./messages.md).

## Redirection to a smart contract

It may be desirable to allow smart contracts to control an interchain account.
To faciliate such an action, the controller module may be provided an underlying application which redirects to smart contract callers. 
An improved design has been suggested in [ADR 008](https://github.com/cosmos/ibc-go/pull/1976) which performs this action via middleware. 

Implementors of this use case are recommended to follow the ADR 008 approach.
The underlying application may continue to be used as a short term solution for ADR 008 and the [legacy API](./auth-modules.md#registerinterchainaccount) should continue to be utilized in such situations. 

## Packet callbacks

If a developer requires access to packet callbacks for their use case, then they having the following options:

1. Write a smart contract which is connected via an ADR 008 or equivalent IBC application (recommended).
2. Use the controller's underlying application to implement packet callback logic.

If the first case, the smart contract should use the [MsgServer](./messages.md).

In the second case, the underlying application should use the [legacy API](./auth-modules.md#registerinterchainaccount).
