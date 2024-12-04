---
title: Gas Management
sidebar_label: Gas Management
sidebar_position: 6
slug: /middleware/callbacks/gas
---

# Gas Management

## Overview

Executing arbitrary code on a chain can be arbitrarily expensive. In general, a callback may consume infinite gas (think of a callback that loops forever). This is problematic for a few reasons:

- It can block the packet lifecycle.
- It can be used to consume all of the relayer's funds and gas.
- A relayer can DOS the callback execution by sending a packet with a low amount of gas.

To prevent these, the callbacks middleware introduces two gas limits: a chain wide gas limit (`maxCallbackGas`) and a user defined gas limit.

### Chain Wide Gas Limit

Since the callbacks middleware does not have a keeper, it does not use a governance parameter to set the chain wide gas limit. Instead, the chain wide gas limit is passed in as a parameter to the callbacks middleware during initialization.

```go
// app.go

maxCallbackGas := uint64(1_000_000)

var transferStack porttypes.IBCModule
transferStack = transfer.NewIBCModule(app.TransferKeeper)
transferStack = ibcfee.NewIBCMiddleware(transferStack, app.IBCFeeKeeper)
transferStack = ibccallbacks.NewIBCMiddleware(transferStack, app.IBCFeeKeeper, app.MockContractKeeper, maxCallbackGas)
// Since the callbacks middleware itself is an ics4wrapper, it needs to be passed to the transfer keeper
app.TransferKeeper.WithICS4Wrapper(transferStack.(porttypes.ICS4Wrapper))

// Add transfer stack to IBC Router
ibcRouter.AddRoute(ibctransfertypes.ModuleName, transferStack)
```

### User Defined Gas Limit

The user defined gas limit is set by the IBC Actor during packet creation. The user defined gas limit is set in the packet memo. If the user defined gas limit is not set or if the user defined gas limit is greater than the chain wide gas limit, then the chain wide gas limit is used as the user defined gas limit.

```jsonc
{
  "src_callback": {
    "address": "callbackAddressString",
    // optional
    "gas_limit": "userDefinedGasLimitString",
  },
  "dest_callback": {
    "address": "callbackAddressString",
    // optional
    "gas_limit": "userDefinedGasLimitString",
  }
}
```

## Gas Limit Enforcement

During a callback execution, there are three types of gas limits that are enforced:

- User defined gas limit
- Chain wide gas limit
- Context gas limit (amount of gas that the relayer has left for this execution)

Chain wide gas limit is used as a maximum to the user defined gas limit as explained in the [previous section](#user-defined-gas-limit). It may also be used as a default value if no user gas limit is provided. Therefore, we can ignore the chain wide gas limit for the rest of this section and work with the minimum of the chain wide gas limit and user defined gas limit. This minimum is called the commit gas limit.

The gas limit enforcement is done by executing the callback inside a cached context with a new gas meter. The gas meter is initialized with the minimum of the commit gas limit and the context gas limit. This minimum is called the execution gas limit. We say that retries are allowed if `context gas limit < commit gas limit`. Otherwise, we say that retries are not allowed.

If the callback execution fails due to an out of gas error, then the middleware checks if retries are allowed. If retries are not allowed, then it recovers from the out of gas error, consumes execution gas limit from the original context, and continues with the packet life cycle. If retries are allowed, then it panics with an out of gas error to revert the entire tx. The packet can then be submitted again with a higher gas limit. The out of gas panic descriptor is shown below.

```go
fmt.Sprintf("ibc %s callback out of gas; commitGasLimit: %d", callbackType, callbackData.CommitGasLimit)}
```

If the callback execution does not fail due to an out of gas error then the callbacks middleware does not block the packet life cycle regardless of whether retries are allowed or not.
