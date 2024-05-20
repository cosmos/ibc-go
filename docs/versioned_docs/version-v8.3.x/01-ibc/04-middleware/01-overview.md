---
title: IBC middleware
sidebar_label: IBC middleware
sidebar_position: 1
slug: /ibc/middleware/overview
---

# IBC middleware

:::note Synopsis
Learn how to write your own custom middleware to wrap an IBC application, and understand how to hook different middleware to IBC base applications to form different IBC application stacks
:::

This documentation serves as a guide for middleware developers who want to write their own middleware and for chain developers who want to use IBC middleware on their chains.

After going through the overview they can consult respectively:

- [documentation on developing custom middleware](02-develop.md)
- [documentation on integrating middleware into a stack on a chain](03-integration.md)

:::note

## Pre-requisite readings

- [IBC Overview](../01-overview.md)
- [IBC Integration](../02-integration.md)
- [IBC Application Developer Guide](../03-apps/01-apps.md)

:::

## Why middleware?

IBC applications are designed to be self-contained modules that implement their own application-specific logic through a set of interfaces with the core IBC handlers. These core IBC handlers, in turn, are designed to enforce the correctness properties of IBC (transport, authentication, ordering) while delegating all application-specific handling to the IBC application modules. **However, there are cases where some functionality may be desired by many applications, yet not appropriate to place in core IBC.**

Middleware allows developers to define the extensions as separate modules that can wrap over the base application. This middleware can thus perform its own custom logic, and pass data into the application so that it may run its logic without being aware of the middleware's existence. This allows both the application and the middleware to implement its own isolated logic while still being able to run as part of a single packet flow.

## Definitions

`Middleware`: A self-contained module that sits between core IBC and an underlying IBC application during packet execution. All messages between core IBC and underlying application must flow through middleware, which may perform its own custom logic.

`Underlying Application`: An underlying application is the application that is directly connected to the middleware in question. This underlying application may itself be middleware that is chained to a base application.

`Base Application`: A base application is an IBC application that does not contain any middleware. It may be nested by 0 or multiple middleware to form an application stack.

`Application Stack (or stack)`: A stack is the complete set of application logic (middleware(s) + base application) that gets connected to core IBC. A stack may be just a base application, or it may be a series of middlewares that nest a base application.

The diagram below gives an overview of a middleware stack consisting of two middleware (one stateless, the other stateful).

![middleware-stack.png](./images/middleware-stack.png)

Keep in mind that:

- **The order of the middleware matters** (more on how to correctly define your stack in the code will follow in the [integration section](03-integration.md)).
- Depending on the type of message, it will either be passed on from the base application up the middleware stack to core IBC or down the stack in the reverse situation (handshake and packet callbacks).
- IBC middleware will wrap over an underlying IBC application and sits between core IBC and the application. It has complete control in modifying any message coming from IBC to the application, and any message coming from the application to core IBC. **Middleware must be completely trusted by chain developers who wish to integrate them**, as this gives them complete flexibility in modifying the application(s) they wrap.
