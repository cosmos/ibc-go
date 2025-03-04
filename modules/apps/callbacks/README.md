# Callbacks Middleware

IBC was designed with callbacks between core IBC and IBC applications. IBC apps would send a packet to core IBC, and receive a callback on every step of that packet's lifecycle. This allows IBC applications to be built on top of core IBC, and to be able to execute custom logic on packet lifecycle events (e.g. unescrow tokens for ICS-20).

This setup worked well for off-chain users interacting with IBC applications. However, we are now seeing the desire for secondary applications (e.g. smart contracts, modules) to call into IBC apps as part of their state machine logic and then do some actions on packet lifecycle events.

The Callbacks Middleware allows for this functionality by allowing the packets of the underlying IBC applications to register callbacks to secondary applications for lifecycle events. These callbacks are then executed by the Callbacks Middleware when the corresponding packet lifecycle event occurs.

After much discussion, the design was expanded to [an ADR](/architecture/adr-008-app-caller-cbs), and the Callbacks Middleware is an implementation of that ADR.

## Version Matrix

The callbacks middleware has no stable releases yet. To use it, you need to import the git commit that contains the module with the compatible version of `ibc-go`. To do so, run the following command with the desired git commit in your project:

```sh
go get github.com/cosmos/ibc-go/modules/apps/callbacks@342c00b0f8bd7feeebf0780f208a820b0faf90d1
```

You can find the version matrix for the callbacks middleware [here](https://github.com/cosmos/ibc-go/blob/main/RELEASES.md#callbacks-middleware-1)
