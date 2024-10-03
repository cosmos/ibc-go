package types

import (
	"crypto/sha256"
	"strings"

	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"
	host "github.com/cosmos/ibc-go/v9/modules/core/24-host"
)

func NewPacket(sequence uint64, sourceID, destinationID string, timeoutTimestamp uint64, data ...PacketData) Packet {
	return Packet{
		Sequence:         sequence,
		SourceId:         sourceID,
		DestinationId:    destinationID,
		TimeoutTimestamp: timeoutTimestamp,
		Data:             data,
	}
}

func (p Packet) ValidateBasic() error {
	if len(p.Data) != 1 {
		return errorsmod.Wrapf(ErrInvalidPacket, "packet data length must be 1, got %d", len(p.Data))
	}

	for _, pd := range p.Data {
		if err := host.PortIdentifierValidator(pd.SourcePort); err != nil {
			return errorsmod.Wrap(err, "invalid source port ID")
		}
		if err := host.PortIdentifierValidator(pd.DestinationPort); err != nil {
			return errorsmod.Wrap(err, "invalid destination port ID")
		}
		if err := pd.Payload.Validate(); err != nil {
			return err
		}
	}

	if err := host.ChannelIdentifierValidator(p.SourceId); err != nil {
		return errorsmod.Wrap(err, "invalid source channel ID")
	}
	if err := host.ChannelIdentifierValidator(p.DestinationId); err != nil {
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

// Validate validates a  Payload.
func (p Payload) Validate() error {
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

// CommitPacket returns the V2 packet commitment bytes. The commitment consists of:
// sha256_hash(timeout) + sha256_hash(destinationID) + sha256_hash(packetData) from a given packet.
// This results in a fixed length preimage.
// NOTE: A fixed length preimage is ESSENTIAL to prevent relayers from being able
// to malleate the packet fields and create a commitment hash that matches the original packet.
func CommitPacket(packet Packet) []byte {
	buf := sdk.Uint64ToBigEndian(packet.GetTimeoutTimestamp())

	destIDHash := sha256.Sum256([]byte(packet.DestinationId))
	buf = append(buf, destIDHash[:]...)

	for _, data := range packet.Data {
		buf = append(buf, hashPacketData(data)...)
	}

	hash := sha256.Sum256(buf)
	return hash[:]
}

// hashPacketData returns the hash of the packet data.
func hashPacketData(data PacketData) []byte {
	var buf []byte
	sourceHash := sha256.Sum256([]byte(data.SourcePort))
	buf = append(buf, sourceHash[:]...)
	destHash := sha256.Sum256([]byte(data.DestinationPort))
	buf = append(buf, destHash[:]...)
	payloadValueHash := sha256.Sum256(data.Payload.Value)
	buf = append(buf, payloadValueHash[:]...)
	payloadEncodingHash := sha256.Sum256([]byte(data.Payload.Encoding))
	buf = append(buf, payloadEncodingHash[:]...)
	payloadVersionHash := sha256.Sum256([]byte(data.Payload.Version))
	buf = append(buf, payloadVersionHash[:]...)
	hash := sha256.Sum256(buf)
	return hash[:]
}
