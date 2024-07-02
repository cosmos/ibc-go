---
title: Development Use Cases
sidebar_label: Development Use Cases
sidebar_position: 2
slug: /apps/interchain-accounts/development
---


# Development use cases

The initial version of Interchain Accounts allowed for the controller submodule to be extended by providing it with an underlying application which would handle all packet callbacks.
That functionality is now being deprecated in favor of alternative approaches. 
This document will outline potential use cases and redirect each use case to the appropriate documentation. 

## Custom authentication 

Interchain accounts may be associated with alternative types of authentication relative to the traditional public/private key signing. 
If you wish to develop or use Interchain Accounts with a custom authentication module and do not need to execute custom logic on the packet callbacks, we recommend you use ibc-go v6 or greater and that your custom authentication module interacts with the controller submodule via the [`MsgServer`](05-messages.md).

If you wish to consume and execute custom logic in the packet callbacks, then please read the section [Packet callbacks](#packet-callbacks) below.

## Redirection to a smart contract

It may be desirable to allow smart contracts to control an interchain account.
To facilitate such an action, the controller submodule may be provided an underlying application which redirects to smart contract callers. 
An improved design has been suggested in [ADR 008](https://github.com/cosmos/ibc-go/pull/1976) which performs this action via middleware. 

Implementers of this use case are recommended to follow the ADR 008 approach.
The underlying application may continue to be used as a short term solution for ADR 008 and the [legacy API](./10-legacy/01-auth-modules.md) should continue to be utilized in such situations. 

## Packet callbacks

If a developer requires access to packet callbacks for their use case, then they have the following options:

1. Write a smart contract which is connected via an ADR 008 or equivalent IBC application (recommended).
2. Use the controller's underlying application to implement packet callback logic.

In the first case, the smart contract should use the [`MsgServer`](05-messages.md).

In the second case, the underlying application should use the [legacy API](10-legacy/03-keeper-api.md).
