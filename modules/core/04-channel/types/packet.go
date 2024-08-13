package types

import (
	"crypto/sha256"
	"slices"

	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"

	clienttypes "github.com/cosmos/ibc-go/v9/modules/core/02-client/types"
	host "github.com/cosmos/ibc-go/v9/modules/core/24-host"
	"github.com/cosmos/ibc-go/v9/modules/core/exported"
)

// CommitPacket returns the packet commitment bytes based on
// the ProtocolVersion specified in the Packet. The commitment
// must commit to all fields in the packet apart from the source port
// source channel and sequence (which will be committed to in the packet commitment key path)
// and the ProtocolVersion which is defining how to hash the packet fields.
// NOTE: The commitment MUST be a fixed length preimage to prevent relayers from being able
// to malleate the packet fields and create a commitment hash that matches the original packet.
func CommitPacket(packet Packet) []byte {
	switch packet.ProtocolVersion {
	case IBC_VERSION_UNSPECIFIED, IBC_VERSION_1:
		return commitV1Packet(packet)
	case IBC_VERSION_2:
		return commitV2Packet(packet)
	default:
		panic("unsupported version")
	}
}

// commitV1Packet returns the V1 packet commitment bytes. The commitment consists of:
// sha256_hash(timeout_timestamp + timeout_height.RevisionNumber + timeout_height.RevisionHeight + sha256_hash(data))
// from a given packet. This results in a fixed length preimage.
// NOTE: A fixed length preimage is ESSENTIAL to prevent relayers from being able
// to malleate the packet fields and create a commitment hash that matches the original packet.
// NOTE: sdk.Uint64ToBigEndian sets the uint64 to a slice of length 8.
func commitV1Packet(packet Packet) []byte {
	timeoutHeight := packet.GetTimeoutHeight()

	buf := sdk.Uint64ToBigEndian(packet.GetTimeoutTimestamp())

	revisionNumber := sdk.Uint64ToBigEndian(timeoutHeight.GetRevisionNumber())
	buf = append(buf, revisionNumber...)

	revisionHeight := sdk.Uint64ToBigEndian(timeoutHeight.GetRevisionHeight())
	buf = append(buf, revisionHeight...)

	dataHash := sha256.Sum256(packet.GetData())
	buf = append(buf, dataHash[:]...)

	hash := sha256.Sum256(buf)
	return hash[:]
}

// commitV2Packet returns the V2 packet commitment bytes. The commitment consists of:
// sha256_hash(timeout_timestamp + timeout_height.RevisionNumber + timeout_height.RevisionHeight)
// + sha256_hash(destPort) + sha256_hash(destChannel) + sha256_hash(version) + sha256_hash(data))
// from a given packet. This results in a fixed length preimage.
// NOTE: A fixed length preimage is ESSENTIAL to prevent relayers from being able
// to malleate the packet fields and create a commitment hash that matches the original packet.
// NOTE: sdk.Uint64ToBigEndian sets the uint64 to a slice of length 8.
func commitV2Packet(packet Packet) []byte {
	timeoutHeight := packet.GetTimeoutHeight()

	timeoutBuf := sdk.Uint64ToBigEndian(packet.GetTimeoutTimestamp())

	// only hash the timeout height if it is non-zero. This will allow us to remove it entirely in the future
	if !timeoutHeight.EQ(clienttypes.ZeroHeight()) {
		revisionNumber := sdk.Uint64ToBigEndian(timeoutHeight.GetRevisionNumber())
		timeoutBuf = append(timeoutBuf, revisionNumber...)

		revisionHeight := sdk.Uint64ToBigEndian(timeoutHeight.GetRevisionHeight())
		timeoutBuf = append(timeoutBuf, revisionHeight...)
	}

	// hash the timeout rather than using a fixed-size preimage directly
	// this will allow more flexibility in the future with how timeouts are defined
	timeoutHash := sha256.Sum256(timeoutBuf)
	buf := timeoutHash[:]

	// hash the destination identifiers since we can no longer retrieve them from the channelEnd
	portHash := sha256.Sum256([]byte(packet.GetDestPort()))
	buf = append(buf, portHash[:]...)
	destinationHash := sha256.Sum256([]byte(packet.GetDestChannel()))
	buf = append(buf, destinationHash[:]...)

	// hash the version only if it is nonempty
	if packet.AppVersion != "" {
		versionHash := sha256.Sum256([]byte(packet.AppVersion))
		buf = append(buf, versionHash[:]...)
	}

	// hash the data
	dataHash := sha256.Sum256(packet.GetData())
	buf = append(buf, dataHash[:]...)

	hash := sha256.Sum256(buf)
	return hash[:]
}

// CommitAcknowledgement returns the hash of commitment bytes
func CommitAcknowledgement(data []byte) []byte {
	hash := sha256.Sum256(data)
	return hash[:]
}

// NewPacket creates a new Packet instance. It panics if the provided
// packet data interface is not registered.
func NewPacket(
	data []byte,
	sequence uint64, sourcePort, sourceChannel,
	destinationPort, destinationChannel string,
	timeoutHeight clienttypes.Height, timeoutTimestamp uint64,
) Packet {
	return Packet{
		Data:               data,
		Sequence:           sequence,
		SourcePort:         sourcePort,
		SourceChannel:      sourceChannel,
		DestinationPort:    destinationPort,
		DestinationChannel: destinationChannel,
		TimeoutHeight:      timeoutHeight,
		TimeoutTimestamp:   timeoutTimestamp,
		ProtocolVersion:    IBC_VERSION_1,
	}
}

func NewPacketWithVersion(
	data []byte,
	sequence uint64, sourcePort, sourceChannel,
	destinationPort, destinationChannel string,
	timeoutHeight clienttypes.Height, timeoutTimestamp uint64,
	appVersion string,
) Packet {
	return Packet{
		Data:               data,
		Sequence:           sequence,
		SourcePort:         sourcePort,
		SourceChannel:      sourceChannel,
		DestinationPort:    destinationPort,
		DestinationChannel: destinationChannel,
		TimeoutHeight:      timeoutHeight,
		TimeoutTimestamp:   timeoutTimestamp,
		ProtocolVersion:    IBC_VERSION_2,
		AppVersion:         appVersion,
	}
}

// GetSequence implements PacketI interface
func (p Packet) GetSequence() uint64 { return p.Sequence }

// GetSourcePort implements PacketI interface
func (p Packet) GetSourcePort() string { return p.SourcePort }

// GetSourceChannel implements PacketI interface
func (p Packet) GetSourceChannel() string { return p.SourceChannel }

// GetDestPort implements PacketI interface
func (p Packet) GetDestPort() string { return p.DestinationPort }

// GetDestChannel implements PacketI interface
func (p Packet) GetDestChannel() string { return p.DestinationChannel }

// GetData implements PacketI interface
func (p Packet) GetData() []byte { return p.Data }

// GetTimeoutHeight implements PacketI interface
func (p Packet) GetTimeoutHeight() exported.Height { return p.TimeoutHeight }

// GetTimeoutTimestamp implements PacketI interface
func (p Packet) GetTimeoutTimestamp() uint64 { return p.TimeoutTimestamp }

// ValidateBasic implements PacketI interface
func (p Packet) ValidateBasic() error {
	if err := host.PortIdentifierValidator(p.SourcePort); err != nil {
		return errorsmod.Wrap(err, "invalid source port ID")
	}
	if err := host.PortIdentifierValidator(p.DestinationPort); err != nil {
		return errorsmod.Wrap(err, "invalid destination port ID")
	}
	if err := host.ChannelIdentifierValidator(p.SourceChannel); err != nil {
		return errorsmod.Wrap(err, "invalid source channel ID")
	}
	if err := host.ChannelIdentifierValidator(p.DestinationChannel); err != nil {
		return errorsmod.Wrap(err, "invalid destination channel ID")
	}
	if p.Sequence == 0 {
		return errorsmod.Wrap(ErrInvalidPacket, "packet sequence cannot be 0")
	}
	if p.TimeoutHeight.IsZero() && p.TimeoutTimestamp == 0 {
		return errorsmod.Wrap(ErrInvalidPacket, "packet timeout height and packet timeout timestamp cannot both be 0")
	}
	if len(p.Data) == 0 {
		return errorsmod.Wrap(ErrInvalidPacket, "packet data bytes cannot be empty")
	}
	if p.AppVersion != "" && slices.Contains([]IBCVersion{IBC_VERSION_UNSPECIFIED, IBC_VERSION_1}, p.ProtocolVersion) {
		return errorsmod.Wrapf(ErrInvalidPacket, "app version cannot be specified when packet does not use protocol %s", IBC_VERSION_2)
	}
	return nil
}

// Validates a PacketId
func (p PacketId) Validate() error {
	if err := host.PortIdentifierValidator(p.PortId); err != nil {
		return errorsmod.Wrap(err, "invalid source port ID")
	}

	if err := host.ChannelIdentifierValidator(p.ChannelId); err != nil {
		return errorsmod.Wrap(err, "invalid source channel ID")
	}

	if p.Sequence == 0 {
		return errorsmod.Wrap(ErrInvalidPacket, "packet sequence cannot be 0")
	}

	return nil
}

// NewPacketID returns a new instance of PacketId
func NewPacketID(portID, channelID string, seq uint64) PacketId {
	return PacketId{PortId: portID, ChannelId: channelID, Sequence: seq}
}
