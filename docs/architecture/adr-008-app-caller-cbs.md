# ADR 008: Callback to IBC Actors

## Changelog

- 2022-08-10: Initial Draft
- 2023-03-22: Merged
- 2023-09-13: Updated with decisions made in implementation

## Status

Accepted, middleware implemented

## Context

IBC was designed with callbacks between core IBC and IBC applications. IBC apps would send a packet to core IBC. When the result of the packet lifecycle eventually resolved into either an acknowledgement or a timeout, core IBC called a callback on the IBC application so that the IBC application could take action on the basis of the result (e.g. unescrow tokens for ICS-20).

This setup worked well for off-chain users interacting with IBC applications.

We are now seeing the desire for secondary applications (e.g. smart contracts, modules) to call into IBC apps as part of their state machine logic and then do some actions on the basis of the packet result. Or to receive a packet from IBC and do some logic upon receipt.

Example Usecases:

- Send an ICS-20 packet, and if it is successful, then send an ICA-packet to swap tokens on LP and return funds to sender
- Execute some logic upon receipt of token transfer to a smart contract address

This requires a second layer of callbacks. The IBC application already gets the result of the packet from core IBC, but currently there is no standardized way to pass this information on to an actor module/smart contract.

## Definitions

- Actor: an actor is an on-chain module (this may be a hardcoded module in the chain binary or a smart contract) that wishes to execute custom logic whenever IBC receives a packet flow that it has either sent or received. It **must** be addressable by a string value.

## Decision

Create a middleware that can interface between IBC applications and smart contract VMs. The IBC applications and smart contract VMs will implement respective interfaces that will then be composed together by the callback middleware to allow a smart contract of any compatible VM to interact programmatically with an IBC application.

## Data structures

The `CallbackPacketData` struct will get constructed from custom callback data in the application packet. The `CallbackAddress` is the IBC Actor address on which the callback should be called on. The `SenderAddress` is also provided to optionally allow a VM to ensure that the sender is the same as the callback address.

The struct also defines a `CommitGasLimit` which is the maximum gas a callback is allowed to use. If the callback exceeds this limit, the callback will panic and the tx will commit without the callback's state changes.

The `ExecutionGasLimit` is the practical limit of the tx execution that is set in the context gas meter. It is the minimum of the `CommitGasLimit` and the gas left in the context gas meter which is determined by the relayer's choice of tx gas limit. If `ExecutionGasLimit < CommitGasLimit`, then an out-of-gas error will revert the entire transaction without committing anything, allowing for a different relayer to retry with a larger tx gas limit.

Any middleware targeting this interface for callback handling should define a global limit that caps the gas that a callback is allowed to take (especially on AcknowledgePacket and TimeoutPacket) so that a custom callback does not prevent the packet lifecycle from completing. However, since this is a global cap it is likely to be very large. Thus, users may specify a smaller limit to cap the amount of fees a relayer must pay in order to complete the packet lifecycle on the user's behalf.

```go
// Implemented by any packet data type that wants to support PacketActor callbacks
// PacketActor's will be unable to act on any packet data type that does not implement
// this interface. 
type CallbackPacketData struct {
    CallbackAddress: string
    ExecutionGasLimit: uint64
    SenderAddress: string
    CommitGasLimit: uint64
}
```

IBC Apps or middleware can then call the IBCActor callbacks like so in their own callbacks:

### Callback Middleware

The CallbackMiddleware wraps an underlying IBC application along with a contractKeeper that delegates the callback to a virtual machine. This allows the Callback middleware to interface any compatible IBC application with any compatible VM (e.g. EVM, WASM) so long as the application implements the `CallbacksCompatibleModule` interface and the VM implements the `ContractKeeper` interface.

```go
// IBCMiddleware implements the ICS26 callbacks for the ibc-callbacks middleware given
// the underlying application.
type IBCMiddleware struct {
	app         types.CallbacksCompatibleModule
	ics4Wrapper porttypes.ICS4Wrapper

	contractKeeper types.ContractKeeper

	// maxCallbackGas defines the maximum amount of gas that a callback actor can ask the
	// relayer to pay for. If a callback fails due to insufficient gas, the entire tx
	// is reverted if the relayer hadn't provided the minimum(userDefinedGas, maxCallbackGas).
	// If the actor hasn't defined a gas limit, then it is assumed to be the maxCallbackGas.
	maxCallbackGas uint64
}
```

### Callback-Compatible IBC Application

The `CallbacksCompatibleModule` extends `porttypes.IBCModule` to include an `UnmarshalPacketData` function that allows the middleware to request that the underlying app unmarshal the packet data. This will then allow the middleware to retrieve the callback specific data from an arbitrary set of IBC application packets.

```go
// CallbacksCompatibleModule is an interface that combines the IBCModule and PacketDataUnmarshaler
// interfaces to assert that the underlying application supports both.
type CallbacksCompatibleModule interface {
	porttypes.IBCModule
	porttypes.PacketDataUnmarshaler
}

// PacketDataUnmarshaler defines an optional interface which allows a middleware to
// request the packet data to be unmarshaled by the base application.
type PacketDataUnmarshaler interface {
	// UnmarshalPacketData unmarshals the packet data into a concrete type
	// ctx, portID, channelID are provided as arguments, so that (if needed)
	// the packet data can be unmarshaled based on the channel version.
	UnmarshalPacketData(ctx sdk.Context, portID, channelID string, bz []byte) (interface{}, error)
}
```

The application's packet data must additionally implement the following interfaces:

```go
// PacketData defines an optional interface which an application's packet data structure may implement.
type PacketData interface {
	// GetPacketSender returns the sender address of the packet data.
	// If the packet sender is unknown or undefined, an empty string should be returned.
	GetPacketSender(sourcePortID string) string
}

// PacketDataProvider defines an optional interfaces for retrieving custom packet data stored on behalf of another application.
// An existing problem in the IBC middleware design is the inability for a middleware to define its own packet data type and insert packet sender provided information.
// A short term solution was introduced into several application's packet data to utilize a memo field to carry this information on behalf of another application.
// This interfaces standardizes that behaviour. Upon realization of the ability for middleware's to define their own packet data types, this interface will be deprecated and removed with time.
type PacketDataProvider interface {
	// GetCustomPacketData returns the packet data held on behalf of another application.
	// The name the information is stored under should be provided as the key.
	// If no custom packet data exists for the key, nil should be returned.
	GetCustomPacketData(key string) interface{}
}
```

The callback data can be embedded in an application packet by providing custom packet data for source and destination callback in the custom packet data under the appropriate key.

```jsonc
// Custom Packet data embedded as a JSON object in the packet data

// src callback custom data
{
  "src_callback": {
    "address": "callbackAddressString",
    // optional
    "gas_limit": "userDefinedGasLimitString",
  }
}

// dest callback custom data
{
  "dest_callback": {
    "address": "callbackAddressString",
    // optional
    "gas_limit": "userDefinedGasLimitString",
  }
}

// src and dest callback custom data embedded together
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

## ContractKeeper

The `ContractKeeper` interface must be implemented by any VM that wants to support IBC callbacks. This allows for separation of concerns
between the middleware which is handling logic intended for all VMs (e.g. setting gas meter, extracting callback data, emitting events), 
while the ContractKeeper can handle the specific details of calling into the VM in question.

The `ContractKeeper` **may** impose additional checks such as ensuring that the contract address is the same as the packet sender in source callbacks. 
It may also disable certain callback methods by simply performing a no-op.

```go
// ContractKeeper defines the entry points exposed to the VM module which invokes a smart contract
type ContractKeeper interface {
	// IBCSendPacketCallback is called in the source chain when a PacketSend is executed. The
	// packetSenderAddress is determined by the underlying module, and may be empty if the sender is
	// unknown or undefined. The contract is expected to handle the callback within the user defined
	// gas limit, and handle any errors, or panics gracefully.
	// This entry point is called with a cached context. If an error is returned, then the changes in
	// this context will not be persisted, and the error will be propagated to the underlying IBC
	// application, resulting in a packet send failure.
	//
	// Implementations are provided with the packetSenderAddress and MAY choose to use this to perform
	// validation on the origin of a given packet. It is recommended to perform the same validation
	// on all source chain callbacks (SendPacket, AcknowledgementPacket, TimeoutPacket). This
	// defensively guards against exploits due to incorrectly wired SendPacket ordering in IBC stacks.
	IBCSendPacketCallback(
		cachedCtx sdk.Context,
		sourcePort string,
		sourceChannel string,
		timeoutHeight clienttypes.Height,
		timeoutTimestamp uint64,
		packetData []byte,
		contractAddress,
		packetSenderAddress string,
	) error
	// IBCOnAcknowledgementPacketCallback is called in the source chain when a packet acknowledgement
	// is received. The packetSenderAddress is determined by the underlying module, and may be empty if
	// the sender is unknown or undefined. The contract is expected to handle the callback within the
	// user defined gas limit, and handle any errors, or panics gracefully.
	// This entry point is called with a cached context. If an error is returned, then the changes in
	// this context will not be persisted, but the packet lifecycle will not be blocked.
	//
	// Implementations are provided with the packetSenderAddress and MAY choose to use this to perform
	// validation on the origin of a given packet. It is recommended to perform the same validation
	// on all source chain callbacks (SendPacket, AcknowledgementPacket, TimeoutPacket). This
	// defensively guards against exploits due to incorrectly wired SendPacket ordering in IBC stacks.
	IBCOnAcknowledgementPacketCallback(
		cachedCtx sdk.Context,
		packet channeltypes.Packet,
		acknowledgement []byte,
		relayer sdk.AccAddress,
		contractAddress,
		packetSenderAddress string,
	) error
	// IBCOnTimeoutPacketCallback is called in the source chain when a packet is not received before
	// the timeout height. The packetSenderAddress is determined by the underlying module, and may be
	// empty if the sender is unknown or undefined. The contract is expected to handle the callback
	// within the user defined gas limit, and handle any error, out of gas, or panics gracefully.
	// This entry point is called with a cached context. If an error is returned, then the changes in
	// this context will not be persisted, but the packet lifecycle will not be blocked.
	//
	// Implementations are provided with the packetSenderAddress and MAY choose to use this to perform
	// validation on the origin of a given packet. It is recommended to perform the same validation
	// on all source chain callbacks (SendPacket, AcknowledgementPacket, TimeoutPacket). This
	// defensively guards against exploits due to incorrectly wired SendPacket ordering in IBC stacks.
	IBCOnTimeoutPacketCallback(
		cachedCtx sdk.Context,
		packet channeltypes.Packet,
		relayer sdk.AccAddress,
		contractAddress,
		packetSenderAddress string,
	) error
	// IBCReceivePacketCallback is called in the destination chain when a packet acknowledgement is written.
	// The contract is expected to handle the callback within the user defined gas limit, and handle any errors,
	// out of gas, or panics gracefully.
	// This entry point is called with a cached context. If an error is returned, then the changes in
	// this context will not be persisted, but the packet lifecycle will not be blocked.
	IBCReceivePacketCallback(
		cachedCtx sdk.Context,
		packet ibcexported.PacketI,
		ack ibcexported.Acknowledgement,
		contractAddress string,
	) error
}
```

### PacketCallbacks

The packet callbacks implemented in the middleware will first call the underlying application and then route to the IBC actor callback in the post-processing step. 
It will extract the callback data from the application packet and set the callback gas meter depending on the global limit, the user limit, and the gas left in the transaction gas meter. 
The callback will then be routed through the callback keeper which will either panic or return a result (success or failure). In the event of a (non-oog) panic or an error, the callback state changes 
are discarded and the transaction is committed.

If the relayer-defined gas limit is exceeded before the user-defined gas limit or global callback gas limit is exceeded, then the entire transaction is reverted to allow for resubmission. If the chain-defined or user-defined gas limit is reached, 
the callback state changes are reverted and the transaction is committed.

For the `SendPacket` callback, we will revert the entire transaction on any kind of error or panic. This is because the packet lifecycle has not yet started, so we can revert completely to avoid starting the packet lifecycle if the callback is not successful.

```go
// SendPacket implements source callbacks for sending packets.
// It defers to the underlying application and then calls the contract callback.
// If the contract callback returns an error, panics, or runs out of gas, then
// the packet send is rejected.
func (im IBCMiddleware) SendPacket(
	ctx sdk.Context,
	chanCap *capabilitytypes.Capability,
	sourcePort string,
	sourceChannel string,
	timeoutHeight clienttypes.Height,
	timeoutTimestamp uint64,
	data []byte,
) (uint64, error) {
    // run underlying app logic first
    // IBCActor logic will postprocess
	seq, err := im.ics4Wrapper.SendPacket(ctx, chanCap, sourcePort, sourceChannel, timeoutHeight, timeoutTimestamp, data)
	if err != nil {
		return 0, err
	}

    // use underlying app to get source callback information from packet data
	callbackData, err := types.GetSourceCallbackData(im.app, data, sourcePort, ctx.GasMeter().GasRemaining(), im.maxCallbackGas)
	// SendPacket is not blocked if the packet does not opt-in to callbacks
	if err != nil {
		return seq, nil
	}

	callbackExecutor := func(cachedCtx sdk.Context) error {
		return im.contractKeeper.IBCSendPacketCallback(
			cachedCtx, sourcePort, sourceChannel, timeoutHeight, timeoutTimestamp, data, callbackData.CallbackAddress, callbackData.SenderAddress,
		)
	}

	err = im.processCallback(ctx, types.CallbackTypeSendPacket, callbackData, callbackExecutor)
	// contract keeper is allowed to reject the packet send.
	if err != nil {
		return 0, err
	}

    types.EmitCallbackEvent(ctx, sourcePort, sourceChannel, seq, types.CallbackTypeSendPacket, callbackData, nil)
	return seq, nil
}

// WriteAcknowledgement implements the ReceivePacket destination callbacks for the ibc-callbacks middleware
// during asynchronous packet acknowledgement.
// It defers to the underlying application and then calls the contract callback.
// If the contract callback runs out of gas and may be retried with a higher gas limit then the state changes are
// reverted via a panic.
func (im IBCMiddleware) WriteAcknowledgement(
	ctx sdk.Context,
	chanCap *capabilitytypes.Capability,
	packet ibcexported.PacketI,
	ack ibcexported.Acknowledgement,
) error {
    // run underlying app logic first
    // IBCActor logic will postprocess
	err := im.ics4Wrapper.WriteAcknowledgement(ctx, chanCap, packet, ack)
	if err != nil {
		return err
	}

    // use underlying app to get destination callback information from packet data
	callbackData, err := types.GetDestCallbackData(
		im.app, packet.GetData(), packet.GetSourcePort(), ctx.GasMeter().GasRemaining(), im.maxCallbackGas,
	)
	// WriteAcknowledgement is not blocked if the packet does not opt-in to callbacks
	if err != nil {
		return nil
	}

	callbackExecutor := func(cachedCtx sdk.Context) error {
		return im.contractKeeper.IBCReceivePacketCallback(cachedCtx, packet, ack, callbackData.CallbackAddress)
	}

	// callback execution errors are not allowed to block the packet lifecycle, they are only used in event emissions
	err = im.processCallback(ctx, types.CallbackTypeReceivePacket, callbackData, callbackExecutor)
	// emit events
    types.EmitCallbackEvent(
		ctx, packet.GetSourcePort(), packet.GetSourceChannel(), packet.GetSequence(),
		types.CallbackTypeAcknowledgementPacket, callbackData, err,
	)

	return nil
}

// Call the IBCActor recvPacket callback after processing the packet
// if the recvPacket callback exists. If the callback returns an error
// then return an error ack to revert all packet data processing. 
func (im IBCMiddleware) OnRecvPacket(
    ctx sdk.Context,
    packet channeltypes.Packet,
    relayer sdk.AccAddress,
) (ack exported.Acknowledgement) {
    // run underlying app logic first
    // IBCActor logic will postprocess
    ack := im.app.OnRecvPacket(ctx, packet, relayer)
	// if ack is nil (asynchronous acknowledgements), then the callback will be handled in WriteAcknowledgement
	// if ack is not successful, all state changes are reverted. If a packet cannot be received, then there is
	// no need to execute a callback on the receiving chain.
	if ack == nil || !ack.Success() {
		return ack
	}

    // use underlying app to get destination callback information from packet data
    callbackData, err := types.GetDestCallbackData(
		im.app, packet.GetData(), packet.GetSourcePort(), ctx.GasMeter().GasRemaining(), im.maxCallbackGas,
	)
	// OnRecvPacket is not blocked if the packet does not opt-in to callbacks
	if err != nil {
		return ack
	}

	callbackExecutor := func(cachedCtx sdk.Context) error {
		return im.contractKeeper.IBCReceivePacketCallback(cachedCtx, packet, ack, callbackData.CallbackAddress)
	}

    // callback execution errors are not allowed to block the packet lifecycle, they are only used in event emissions
	err = im.processCallback(ctx, types.CallbackTypeReceivePacket, callbackData, callbackExecutor)
	types.EmitCallbackEvent(
		ctx, packet.GetDestPort(), packet.GetDestChannel(), packet.GetSequence(),
		types.CallbackTypeReceivePacket, callbackData, err,
	)

	return ack
}

// Call the IBCActor acknowledgementPacket callback after processing the packet
// if the ackPacket callback exists and returns an error
// DO NOT return the error upstream. The acknowledgement must complete for the packet
// lifecycle to end, so the custom callback cannot block completion.
// Instead we emit error events and set the error in state
// so that users and on-chain logic can handle this appropriately
func (im IBCModule) OnAcknowledgementPacket(
    ctx sdk.Context,
    packet channeltypes.Packet,
    acknowledgement []byte,
    relayer sdk.AccAddress,
) error {
    // we first call the underlying app to handle the acknowledgement
    // IBCActor logic will postprocess
	err := im.app.OnAcknowledgementPacket(ctx, packet, acknowledgement, relayer)
	if err != nil {
		return err
	}

    // use underlying app to get source callback information from packet data
	callbackData, err := types.GetSourceCallbackData(
		im.app, packet.GetData(), packet.GetSourcePort(), ctx.GasMeter().GasRemaining(), im.maxCallbackGas,
	)
	// OnAcknowledgementPacket is not blocked if the packet does not opt-in to callbacks
	if err != nil {
		return nil
	}

	callbackExecutor := func(cachedCtx sdk.Context) error {
		return im.contractKeeper.IBCOnAcknowledgementPacketCallback(
			cachedCtx, packet, acknowledgement, relayer, callbackData.CallbackAddress, callbackData.SenderAddress,
		)
	}

	// callback execution errors are not allowed to block the packet lifecycle, they are only used in event emissions
	err = im.processCallback(ctx, types.CallbackTypeAcknowledgementPacket, callbackData, callbackExecutor)
    types.EmitCallbackEvent(
		ctx, packet.GetSourcePort(), packet.GetSourceChannel(), packet.GetSequence(),
		types.CallbackTypeAcknowledgementPacket, callbackData, err,
	)

	return nil
}

// Call the IBCActor timeoutPacket callback after processing the packet
// if the timeoutPacket callback exists and returns an error
// DO NOT return the error upstream. The timeout must complete for the packet
// lifecycle to end, so the custom callback cannot block completion.
// Instead we emit error events and set the error in state
// so that users and on-chain logic can handle this appropriately
func (im IBCModule) OnTimeoutPacket(
    ctx sdk.Context,
    packet channeltypes.Packet,
    relayer sdk.AccAddress,
) error {
    // application-specific onTimeoutPacket logic
    err := im.app.OnTimeoutPacket(ctx, packet, relayer)
	if err != nil {
		return err
	}

    // use underlying app to get source callback information from packet data
	callbackData, err := types.GetSourceCallbackData(
		im.app, packet.GetData(), packet.GetSourcePort(), ctx.GasMeter().GasRemaining(), im.maxCallbackGas,
	)
	// OnTimeoutPacket is not blocked if the packet does not opt-in to callbacks
	if err != nil {
		return nil
	}

	callbackExecutor := func(cachedCtx sdk.Context) error {
		return im.contractKeeper.IBCOnTimeoutPacketCallback(cachedCtx, packet, relayer, callbackData.CallbackAddress, callbackData.SenderAddress)
	}

	// callback execution errors are not allowed to block the packet lifecycle, they are only used in event emissions
	err = im.processCallback(ctx, types.CallbackTypeTimeoutPacket, callbackData, callbackExecutor)
	types.EmitCallbackEvent(
		ctx, packet.GetSourcePort(), packet.GetSourceChannel(), packet.GetSequence(),
		types.CallbackTypeTimeoutPacket, callbackData, err,
	)

	return nil
}

// processCallback executes the callbackExecutor and reverts contract changes if the callbackExecutor fails.
//
// Error Precedence and Returns:
//   - oogErr: Takes the highest precedence. If the callback runs out of gas, an error wrapped with types.ErrCallbackOutOfGas is returned.
//   - panicErr: Takes the second-highest precedence. If a panic occurs and it is not propagated, an error wrapped with types.ErrCallbackPanic is returned.
//   - callbackErr: If the callbackExecutor returns an error, it is returned as-is.
//
// panics if
//   - the contractExecutor panics for any reason, and the callbackType is SendPacket, or
//   - the contractExecutor runs out of gas and the relayer has not reserved gas grater than or equal to
//     CommitGasLimit.
func (IBCMiddleware) processCallback(
	ctx sdk.Context, callbackType types.CallbackType,
	callbackData types.CallbackData, callbackExecutor func(sdk.Context) error,
) (err error) {
	cachedCtx, writeFn := ctx.CacheContext()
	cachedCtx = cachedCtx.WithGasMeter(storetypes.NewGasMeter(callbackData.ExecutionGasLimit))

	defer func() {
		// consume the minimum of g.consumed and g.limit
		ctx.GasMeter().ConsumeGas(cachedCtx.GasMeter().GasConsumedToLimit(), fmt.Sprintf("ibc %s callback", callbackType))

		// recover from all panics except during SendPacket callbacks
		if r := recover(); r != nil {
			if callbackType == types.CallbackTypeSendPacket {
				panic(r)
			}
			err = errorsmod.Wrapf(types.ErrCallbackPanic, "ibc %s callback panicked with: %v", callbackType, r)
		}

		// if the callback ran out of gas and the relayer has not reserved enough gas, then revert the state
		if cachedCtx.GasMeter().IsPastLimit() {
			if callbackData.AllowRetry() {
				panic(storetypes.ErrorOutOfGas{Descriptor: fmt.Sprintf("ibc %s callback out of gas; commitGasLimit: %d", callbackType, callbackData.CommitGasLimit)})
			}
			err = errorsmod.Wrapf(types.ErrCallbackOutOfGas, "ibc %s callback out of gas", callbackType)
		}

		// allow the transaction to be committed, continuing the packet lifecycle
	}()

	err = callbackExecutor(cachedCtx)
	if err == nil {
		writeFn()
	}

	return err
}
```

Chains are expected to specify a `maxCallbackGas` to ensure that callbacks do not consume an arbitrary amount of gas. Thus, it should always be possible for a relayer to complete the packet lifecycle even if the actor callbacks cannot run successfully.

## Consequences

### Positive

- IBC Actors can now programmatically execute logic that involves sending a packet and then performing some additional logic once the packet lifecycle is complete
- Middleware implementing ADR-8 can be generally used for any application
- Leverages a similar callback architecture to the one used between core IBC and IBC applications

### Negative

- Callbacks may now have unbounded gas consumption since the actor may execute arbitrary logic. Chains implementing this feature should take care to place limitations on how much gas an actor callback can consume.
- The relayer pays for the callback gas instead of the IBCActor

### Neutral

- Application packets that want to support ADR-8 must additionally have their packet data implement `PacketDataProvider` and `PacketData` interfaces.
- Applications must implement `PacketDataUnmarshaler` interface
- Callback receiving module must implement the `ContractKeeper` interface

## References

- [Original issue](https://github.com/cosmos/ibc-go/issues/1660)
- [CallbackPacketData interface implementation](https://github.com/cosmos/ibc-go/pull/3287) 
- [ICS 20, ICS 27 implementations of the CallbackPacketData interface](https://github.com/cosmos/ibc-go/pull/3287)
