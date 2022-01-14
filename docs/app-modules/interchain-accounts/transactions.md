<!--
order: 5
-->

# Transactions

Learn about Interchain Accounts transaction execution {synopsis}

## Executing a transaction

As described in [Authentication Modules](./auth-modules.md#trysendtx) transactions are executed using the interchain accounts controller API and require a `Base Application` as outlined in [ICS30 IBC Middleware](https://github.com/cosmos/ibc/tree/master/spec/app/ics-030-middleware) to facilitate authentication. The method of authentication remains unspecified to provide flexibility for the authentication module developer.

Transactions are executed via the ICS27 [`TrySendTx` API](./auth-modules.md#trysendtx). This must be invoked through an Interchain Accounts authentication module and follows the outlined path of execution below. Packet relaying semantics provided by the IBC core transport, authentication, and ordering (IBC/TAO) layer are omitted for brevity.

![send-tx-flow](../../assets/send-interchain-tx.png "Transaction Execution")

## Atomicity

As the Interchain Accounts module supports the execution of multiple transactions using the Cosmos SDK `Msg` interface, it provides the same atomicity guarantees as Cosmos SDK-based applications, leveraging the [`CacheMultiStore`](https://docs.cosmos.network/master/core/store.html#cachemultistore) architecture provided by the `Context` type. 

When a host chain receives an Interchain Accounts packet and successfully deserializes its message content, each `Msg` is authenticated and executed using a cached storage object. A new `CacheMultiStore` is obtained via the Cosmos SDK `ctx.CacheContext()` method. State changes are then performed within the context of a branched key-value store and commited when `writeCache()` is invoked. This storage mechanism ensures that state changes are committed if and only if all transactions succeed.

```go
func (k Keeper) executeTx(ctx sdk.Context, sourcePort, destPort, destChannel string, msgs []sdk.Msg) error {
	if err := k.AuthenticateTx(ctx, msgs, sourcePort); err != nil {
		return err
	}

	// CacheContext returns a new context with the multi-store branched into a cached storage object
	// writeCache is called only if all msgs succeed, performing state transitions atomically
	cacheCtx, writeCache := ctx.CacheContext()
	for _, msg := range msgs {
		if err := msg.ValidateBasic(); err != nil {
			return err
		}

		if err := k.executeMsg(cacheCtx, msg); err != nil {
			return err
		}
	}

	// NOTE: The context returned by CacheContext() creates a new EventManager, so events must be correctly propagated back to the current context
	ctx.EventManager().EmitEvents(cacheCtx.EventManager().Events())
	writeCache()

	return nil
}
```