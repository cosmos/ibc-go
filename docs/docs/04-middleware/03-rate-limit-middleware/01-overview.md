---
title: Overview
sidebar_label: Overview
sidebar_position: 1
slug: /apps/rate-limit-middleware/overview
---

# Overview

Learn about rate limit middleware, a middleware that can be used in combination with token transfers (ICS-20) to control the amount of in and outflows of assets in a certain time period. 

## What is Rate Limit Middleware?

The rate limit middleware enforces rate limits on IBC token transfers coming into and out of a chain. It supports: 

- **Risk Mitigation:** In case of a bug exploit, attack or economic failure of a connected chain, it limits the impact to the in/outflow specified for a given time period. 
- **Token Filtering:** Through the use of a blacklist, the middleware can completely block tokens entering or leaving a domain, relevant for complicance or giving asset issuers greater control over the domains token can be sent to. 
- **Uninterupted Packet Flow:** When desired, rate limits can be bypassed by using the whitelist, to avoid any restriction on asset in or outflows. 

## How it works

The rate limiting middleware determines whether tokens can flow into or out of a chain. The middleware does this by: 

1. Check transfer limits for an asset (Quota): When tokens are recieved or sent, the middleware determines whether the amount of tokens flowing in or out have exceeded the limit. 

2. Track in or outflow: When tokens enter or leave the chain, the amount transferred is tracked in state

3. Block or allow token flow: Dependent on the limit, the middleware will either allow the tokens to pass through or block the tokens.

4. Handle failures: If the packet timesout or fails to be delivered, the middleware ensures limits are correctly recorded. 
