<!--
order: 2
-->

# Implement `IBCModule` interface and callbacks

Learn how to implement the `IBCModule` interface and all of the callbacks it requires. {synopsis}

The Cosmos SDK expects all IBC modules to implement the [`IBCModule`
interface](https://github.com/cosmos/ibc-go/tree/main/modules/core/05-port/types/module.go). This interface contains all of the callbacks IBC expects modules to implement. They include callbacks related to channel handshake, closing and packet callbacks (`OnRecvPacket`, `OnAcknowledgementPacket` and `OnTimeoutPacket`).

```go
// IBCModule implements the ICS26 interface for given the keeper.
// The implementation of the IBCModule interface could for example be in a file called ibc_module.go,
// but ultimately file structure is up to the developer
type IBCModule struct {
	keeper keeper.Keeper
}
```

Additionally, in the `module.go` file, add the following line:

```go
var (
    _ module.AppModule      = AppModule{}
    _ module.AppModuleBasic = AppModuleBasic{}
    // Add this line
    _ porttypes.IBCModule   = IBCModule{}
)
```

## Pre-requisites Readings

- [IBC Overview](../overview.md)) {prereq}
- [IBC default integration](../integration.md) {prereq}

## Channel handshake callbacks

This section will describe the callbacks that are called during channel handshake execution. Among other things, it will claim channel capabilities passed on from core IBC. For a refresher on capabilities, check [the Overview section](../overview.md#capabilities).

Here are the channel handshake callbacks that modules are expected to implement:

> Note that some of the code below is _pseudo code_, indicating what actions need to happen but leaving it up to the developer to implement a custom implementation. E.g. the `checkArguments` and `negotiateAppVersion` functions.

```go
// Called by IBC Handler on MsgOpenInit
func (im IBCModule) OnChanOpenInit(ctx sdk.Context,
    order channeltypes.Order,
    connectionHops []string,
    portID string,
    channelID string,
    channelCap *capabilitytypes.Capability,
    counterparty channeltypes.Counterparty,
    version string,
) (string, error) {
    // ... do custom initialization logic

    // Use above arguments to determine if we want to abort handshake
    // Examples:
    // - Abort if order == UNORDERED,
    // - Abort if version is unsupported
    if err := checkArguments(args); err != nil {
        return "", err
    }

     // OpenInit must claim the channelCapability that IBC passes into the callback
    if err := im.keeper.ClaimCapability(ctx, chanCap, host.ChannelCapabilityPath(portID, channelID)); err != nil {
			return "", err
	}

    return version, nil
}

// Called by IBC Handler on MsgOpenTry
func (im IBCModule) OnChanOpenTry(
    ctx sdk.Context,
    order channeltypes.Order,
    connectionHops []string,
    portID,
    channelID string,
    channelCap *capabilitytypes.Capability,
    counterparty channeltypes.Counterparty,
    counterpartyVersion string,
) (string, error) {
    // ... do custom initialization logic

    // Use above arguments to determine if we want to abort handshake
    if err := checkArguments(args); err != nil {
        return "", err
    }

    // OpenTry must claim the channelCapability that IBC passes into the callback
    if err := im.keeper.scopedKeeper.ClaimCapability(ctx, chanCap, host.ChannelCapabilityPath(portID, channelID)); err != nil {
        return err
    }

    // Construct application version
    // IBC applications must return the appropriate application version
    // This can be a simple string or it can be a complex version constructed
    // from the counterpartyVersion and other arguments.
    // The version returned will be the channel version used for both channel ends.
    appVersion := negotiateAppVersion(counterpartyVersion, args)

    return appVersion, nil
}

// Called by IBC Handler on MsgOpenAck
func (im IBCModule) OnChanOpenAck(
    ctx sdk.Context,
    portID,
    channelID string,
    counterpartyVersion string,
) error {
    if counterpartyVersion != types.Version {
		return sdkerrors.Wrapf(types.ErrInvalidVersion, "invalid counterparty version: %s, expected %s", counterpartyVersion, types.Version)
	}

    // do custom logic

    return nil
}

// Called by IBC Handler on MsgOpenConfirm
func (im IBCModule) OnChanOpenConfirm(
    ctx sdk.Context,
    portID,
    channelID string,
) error {
    // do custom logic

    return nil
}
```

The channel closing handshake will also invoke module callbacks that can return errors to abort the closing handshake. Closing a channel is a 2-step handshake, the initiating chain calls `ChanCloseInit` and the finalizing chain calls `ChanCloseConfirm`.

```go
// Called by IBC Handler on MsgCloseInit
func (im IBCModule) OnChanCloseInit(
    ctx sdk.Context,
    portID,
    channelID string,
) error {
    // ... do custom finalization logic

    // Use above arguments to determine if we want to abort handshake
    err := checkArguments(args)
    return err
}

// Called by IBC Handler on MsgCloseConfirm
func (im IBCModule) OnChanCloseConfirm(
    ctx sdk.Context,
    portID,
    channelID string,
) error {
    // ... do custom finalization logic

    // Use above arguments to determine if we want to abort handshake
    err := checkArguments(args)
    return err
}
```

### Channel handshake version negotiation

Application modules are expected to verify versioning used during the channel handshake procedure.

- `OnChanOpenInit` will verify that the relayer-chosen parameters
  are valid and perform any custom `INIT` logic.
  It may return an error if the chosen parameters are invalid
  in which case the handshake is aborted.
  If the provided version string is non-empty, `OnChanOpenInit` should return
  the version string if valid or an error if the provided version is invalid.
  **If the version string is empty, `OnChanOpenInit` is expected to
  return a default version string representing the version(s)
  it supports.**
  If there is no default version string for the application,
  it should return an error if the provided version is an empty string.
- `OnChanOpenTry` will verify the relayer-chosen parameters along with the
  counterparty-chosen version string and perform custom `TRY` logic.
  If the relayer-chosen parameters
  are invalid, the callback must return an error to abort the handshake.
  If the counterparty-chosen version is not compatible with this module's
  supported versions, the callback must return an error to abort the handshake.
  If the versions are compatible, the try callback must select the final version
  string and return it to core IBC.
  `OnChanOpenTry` may also perform custom initialization logic.
- `OnChanOpenAck` will error if the counterparty selected version string
  is invalid and abort the handshake. It may also perform custom ACK logic.

Versions must be strings but can implement any versioning structure. If your application plans to
have linear releases then semantic versioning is recommended. If your application plans to release
various features in between major releases then it is advised to use the same versioning scheme
as IBC. This versioning scheme specifies a version identifier and compatible feature set with
that identifier. Valid version selection includes selecting a compatible version identifier with
a subset of features supported by your application for that version. The struct used for this
scheme can be found in [03-connection/types](https://github.com/cosmos/ibc-go/blob/main/modules/core/03-connection/types/version.go#L16).

Since the version type is a string, applications have the ability to do simple version verification
via string matching or they can use the already impelemented versioning system and pass the proto
encoded version into each handhshake call as necessary.

ICS20 currently implements basic string matching with a single supported version.

## Packet callbacks

Just as IBC expects modules to implement callbacks for channel handshakes, it also expects modules to implement callbacks for handling the packet flow through a channel, as defined in the `IBCModule` interface.

Once a module A and module B are connected to each other, relayers can start relaying packets and acknowledgements back and forth on the channel.

![IBC packet flow diagram](https://ibcprotocol.org/_nuxt/img/packet_flow.1d89ee0.png)

Briefly, a successful packet flow works as follows:

1. module A sends a packet through the IBC module
2. the packet is received by module B
3. if module B writes an acknowledgement of the packet then module A will process the
   acknowledgement
4. if the packet is not successfully received before the timeout, then module A processes the
   packet's timeout.

### Sending packets

Modules **do not send packets through callbacks**, since the modules initiate the action of sending packets to the IBC module, as opposed to other parts of the packet flow where messages sent to the IBC
module must trigger execution on the port-bound module through the use of callbacks. Thus, to send a packet a module simply needs to call `SendPacket` on the `IBCChannelKeeper`.

> Note that some of the code below is _pseudo code_, indicating what actions need to happen but leaving it up to the developer to implement a custom implementation. E.g. the `EncodePacketData(customPacketData)` function.

```go
// retrieve the dynamic capability for this channel
channelCap := scopedKeeper.GetCapability(ctx, channelCapName)
// Sending custom application packet data
data := EncodePacketData(customPacketData)
packet.Data = data
// Send packet to IBC, authenticating with channelCap
IBCChannelKeeper.SendPacket(ctx, channelCap, packet)
```

::: warning
In order to prevent modules from sending packets on channels they do not own, IBC expects
modules to pass in the correct channel capability for the packet's source channel.
:::

### Receiving packets

To handle receiving packets, the module must implement the `OnRecvPacket` callback. This gets
invoked by the IBC module after the packet has been proved valid and correctly processed by the IBC
keepers. Thus, the `OnRecvPacket` callback only needs to worry about making the appropriate state
changes given the packet data without worrying about whether the packet is valid or not.

Modules may return to the IBC handler an acknowledgement which implements the `Acknowledgement` interface.
The IBC handler will then commit this acknowledgement of the packet so that a relayer may relay the
acknowledgement back to the sender module.

The state changes that occurred during this callback will only be written if:

- the acknowledgement was successful as indicated by the `Success()` function of the acknowledgement
- if the acknowledgement returned is nil indicating that an asynchronous process is occurring

NOTE: Applications which process asynchronous acknowledgements must handle reverting state changes
when appropriate. Any state changes that occurred during the `OnRecvPacket` callback will be written
for asynchronous acknowledgements.

> Note that some of the code below is _pseudo code_, indicating what actions need to happen but leaving it up to the developer to implement a custom implementation. E.g. the `DecodePacketData(packet.Data)` function.

```go
func (im IBCModule) OnRecvPacket(
    ctx sdk.Context,
    packet channeltypes.Packet,
) ibcexported.Acknowledgement {
    // Decode the packet data
    packetData := DecodePacketData(packet.Data)

    // do application state changes based on packet data and return the acknowledgement
    // NOTE: The acknowledgement will indicate to the IBC handler if the application
    // state changes should be written via the `Success()` function. Application state
    // changes are only written if the acknowledgement is successful or the acknowledgement
    // returned is nil indicating that an asynchronous acknowledgement will occur.
    ack := processPacket(ctx, packet, packetData)

    return ack
}
```

Reminder, the `Acknowledgement` interface:

```go
// Acknowledgement defines the interface used to return
// acknowledgements in the OnRecvPacket callback.
type Acknowledgement interface {
	Success() bool
	Acknowledgement() []byte
}
```

### Acknowledging packets

After a module writes an acknowledgement, a relayer can relay back the acknowledgement to the sender module. The sender module can
then process the acknowledgement using the `OnAcknowledgementPacket` callback. The contents of the
acknowledgement is entirely up to the modules on the channel (just like the packet data); however, it
may often contain information on whether the packet was successfully processed along
with some additional data that could be useful for remediation if the packet processing failed.

Since the modules are responsible for agreeing on an encoding/decoding standard for packet data and
acknowledgements, IBC will pass in the acknowledgements as `[]byte` to this callback. The callback
is responsible for decoding the acknowledgement and processing it.

> Note that some of the code below is _pseudo code_, indicating what actions need to happen but leaving it up to the developer to implement a custom implementation. E.g. the `DecodeAcknowledgement(acknowledgments)` and `processAck(ack)` functions.

```go
func (im IBCModule) OnAcknowledgementPacket(
    ctx sdk.Context,
    packet channeltypes.Packet,
    acknowledgement []byte,
) (*sdk.Result, error) {
    // Decode acknowledgement
    ack := DecodeAcknowledgement(acknowledgement)

    // process ack
    res, err := processAck(ack)
    return res, err
}
```

### Timeout packets

If the timeout for a packet is reached before the packet is successfully received or the
counterparty channel end is closed before the packet is successfully received, then the receiving
chain can no longer process it. Thus, the sending chain must process the timeout using
`OnTimeoutPacket` to handle this situation. Again the IBC module will verify that the timeout is
indeed valid, so our module only needs to implement the state machine logic for what to do once a
timeout is reached and the packet can no longer be received.

```go
func (im IBCModule) OnTimeoutPacket(
    ctx sdk.Context,
    packet channeltypes.Packet,
) (*sdk.Result, error) {
    // do custom timeout logic
}
```
