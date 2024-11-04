package types

import (
	"strings"
	"time"

	errorsmod "cosmossdk.io/errors"

	host "github.com/cosmos/ibc-go/v9/modules/core/24-host"
)

// NewPacket constructs a new packet.
func NewPacket(sequence uint64, sourceChannel, destinationChannel string, timeoutTimestamp uint64, payloads ...Payload) Packet {
	return Packet{
		Sequence:           sequence,
		SourceChannel:      sourceChannel,
		DestinationChannel: destinationChannel,
		TimeoutTimestamp:   timeoutTimestamp,
		Payloads:           payloads,
	}
}

// NewPayload constructs a new Payload
func NewPayload(sourcePort, destPort, version, encoding string, value []byte) Payload {
	return Payload{
		SourcePort:      sourcePort,
		DestinationPort: destPort,
		Version:         version,
		Encoding:        encoding,
		Value:           value,
	}
}

// ValidateBasic validates that a Packet satisfies the basic requirements.
func (p Packet) ValidateBasic() error {
	if len(p.Payloads) != 1 {
		return errorsmod.Wrap(ErrInvalidPacket, "payloads must not be empty")
	}

	for _, pd := range p.Payloads {
		if err := pd.ValidateBasic(); err != nil {
			return errorsmod.Wrap(err, "invalid Payload")
		}
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
	if p.TimeoutTimestamp == 0 {
		return errorsmod.Wrap(ErrInvalidPacket, "packet timeout timestamp cannot be 0")
	}

	return nil
}

// ValidateBasic validates a Payload.
func (p Payload) ValidateBasic() error {
	if err := host.PortIdentifierValidator(p.SourcePort); err != nil {
		return errorsmod.Wrap(err, "invalid source port ID")
	}
	if err := host.PortIdentifierValidator(p.DestinationPort); err != nil {
		return errorsmod.Wrap(err, "invalid destination port ID")
	}
	if strings.TrimSpace(p.Version) == "" {
		return errorsmod.Wrap(ErrInvalidPayload, "payload version cannot be empty")
	}
	if strings.TrimSpace(p.Encoding) == "" {
		return errorsmod.Wrap(ErrInvalidPayload, "payload encoding cannot be empty")
	}
	if len(p.Value) == 0 {
		return errorsmod.Wrap(ErrInvalidPayload, "payload value cannot be empty")
	}
	return nil
}

// TimeoutTimestampToNanos takes seconds as a uint64 and returns Unix nanoseconds as uint64.
func TimeoutTimestampToNanos(seconds uint64) uint64 {
	return uint64(time.Unix(int64(seconds), 0).UnixNano())
}

// NewPacketState creates and returns a new PacketState envelope type to encapsulate packet commitments, acks or receipts.
func NewPacketState(channelID string, sequence uint64, data []byte) PacketState {
	return PacketState{
		ChannelId: channelID,
		Sequence:  sequence,
		Data:      data,
	}
}
