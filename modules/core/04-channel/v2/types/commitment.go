package types

import (
	"crypto/sha256"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// CommitPacket returns the V2 packet commitment bytes. The commitment consists of:
// sha256_hash(timeout) + sha256_hash(destinationChannel) + sha256_hash(packetData) from a given packet.
// This results in a fixed length preimage.
// NOTE: A fixed length preimage is ESSENTIAL to prevent relayers from being able
// to malleate the packet fields and create a commitment hash that matches the original packet.
func CommitPacket(packet Packet) []byte {
	buf := sdk.Uint64ToBigEndian(packet.GetTimeoutTimestamp())

	destIDHash := sha256.Sum256([]byte(packet.DestinationChannel))
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

// CommitAcknowledgement returns the hash of the acknowledgement data.
func CommitAcknowledgement(acknowledgement Acknowledgement) []byte {
	var buf []byte
	for _, ack := range acknowledgement.GetAcknowledgementResults() {
		hash := sha256.Sum256(ack.RecvPacketResult.GetAcknowledgement())
		buf = append(buf, hash[:]...)
	}

	hash := sha256.Sum256(buf)
	return hash[:]
}
