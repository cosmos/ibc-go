---
title: Overview
sidebar_label: Overview
sidebar_position: 1
slug: /apps/packet-forward-middleware/overview
---

:::warning
Packet forward middleware is only compatible with IBC classic, not IBC v2
:::

# Overview

Learn about packet forward middleware, a middleware that can be used in combination with token transfers (ICS-20)

## What is Packet Forward Middleware?

Packet Forward Middleware enables multi-hop token transfers by forwarding IBC packets through intermediate chains, which may not be directly connected. It supports:

- **Path-Unwinding Functionality:**
 Because the fungibility of tokens transferred between chains is determined by [the path the tokens have travelled](/02-apps/01-transfer/01-overview/#denomination-trace), i.e. the same token sent from chain A to chain B is not fungible with the same token sent from chain A, to chain C and then to chain B, packet forward middleware also enables routing tokens back through their source, before sending onto the final destination.
- **Asynchronous Acknowledgements:**
 Acknowledgements are only written to the origin chain after all forwarding steps succeed or fail, users only need to monitor the source chain for the result. 
- **Retry and Timeout Handling:**
The middleware can be configured to retry forwarding in the case that there was a timeout.
- **Forwarding across multiple chains with nested memos:**
Instructions on which route to take to forward a packet across more than one chain can be set within a nested JSON with the memo field
- **Configurable Fee Deduction on Recieve:**
Integrators of PFM can choose to deduct a percentage of tokens forwarded through their chain and distribute these tokens to the community pool.

## How it works?

1. User initiates a `MsgTransfer` with a memo JSON payload containing forwarding instructions.

2. Intermediate chains (with PFM enabled) parse the memo and forward the packet to the destination specified.

3. Acknowledgements are passed back step-by-step to the origin chain after the final hop succeeds or fails, along the same path used for forwarding.

In practise, it can be challenging to correctly format the memo for the desired route. It is recommended to use the Skip API to correctly format the memo needed in `MsgTransfer` to make this easy. 
