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

The following table shows the compatibility matrix between the callbacks middleware, `ibc-go`.

|      **Version**     |         **Git commit to import**         |
|:--------------------:|:----------------------------------------:|
| `v0.2.0+ibc-go-v8.0` | 342c00b0f8bd7feeebf0780f208a820b0faf90d1 |
| `v0.1.0+ibc-go-v7.3` | 17cf1260a9cdc5292512acc9bcf6336ef0d917f4 |
