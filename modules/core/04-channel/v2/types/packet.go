package types

import (
	"strings"

	errorsmod "cosmossdk.io/errors"

	channeltypesv1 "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	host "github.com/cosmos/ibc-go/v10/modules/core/24-host"
)

// NewPacket constructs a new packet.
func NewPacket(sequence uint64, sourceClient, destinationClient string, timeoutTimestamp uint64, payloads ...Payload) Packet {
	return Packet{
		Sequence:          sequence,
		SourceClient:      sourceClient,
		DestinationClient: destinationClient,
		TimeoutTimestamp:  timeoutTimestamp,
		Payloads:          payloads,
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
		return errorsmod.Wrap(ErrInvalidPacket, "payloads must contain exactly one payload")
	}

	totalPayloadsSize := 0
	for _, pd := range p.Payloads {
		if err := pd.ValidateBasic(); err != nil {
			return errorsmod.Wrap(err, "invalid Payload")
		}
		totalPayloadsSize += len(pd.Value)
	}

	if totalPayloadsSize > channeltypesv1.MaximumPayloadsSize {
		return errorsmod.Wrapf(ErrInvalidPacket, "packet data bytes cannot exceed %d bytes", channeltypesv1.MaximumPayloadsSize)
	}

	if err := host.ChannelIdentifierValidator(p.SourceClient); err != nil {
		return errorsmod.Wrap(err, "invalid source ID")
	}
	if err := host.ChannelIdentifierValidator(p.DestinationClient); err != nil {
		return errorsmod.Wrap(err, "invalid destination ID")
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
