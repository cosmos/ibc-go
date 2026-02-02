package types

import (
	"crypto/sha256"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// CommitPacket returns the V2 packet commitment bytes. The commitment consists of:
// ha256_hash(0x02 + sha256_hash(destinationClient) + sha256_hash(timeout) + sha256_hash(payload)) from a given packet.
// This results in a fixed length preimage of 32 bytes.
// NOTE: A fixed length preimage is ESSENTIAL to prevent relayers from being able
// to malleate the packet fields and create a commitment hash that matches the original packet.
func CommitPacket(packet Packet) []byte {
	buf := make([]byte, 0, 1+32*3)
	buf = append(buf, byte(2))

	destIDHash := sha256.Sum256([]byte(packet.DestinationClient))
	buf = append(buf, destIDHash[:]...)

	timeoutBytes := sdk.Uint64ToBigEndian(packet.GetTimeoutTimestamp())
	timeoutHash := sha256.Sum256(timeoutBytes)
	buf = append(buf, timeoutHash[:]...)

	appH := sha256.New()
	for _, payload := range packet.Payloads {
		_, _ = appH.Write(hashPayload(payload)) // 32 bytes each
	}
	var appHash [32]byte
	copy(appHash[:], appH.Sum(nil))

	buf = append(buf, appHash[:]...)

	sum := sha256.Sum256(buf)
	return sum[:]
}

// hashPayload returns the hash of the payload.
func hashPayload(data Payload) []byte {
	buf := make([]byte, 0, 32*5)

	sourceHash := sha256.Sum256([]byte(data.SourcePort))
	buf = append(buf, sourceHash[:]...)
	destHash := sha256.Sum256([]byte(data.DestinationPort))
	buf = append(buf, destHash[:]...)
	payloadVersionHash := sha256.Sum256([]byte(data.Version))
	buf = append(buf, payloadVersionHash[:]...)
	payloadEncodingHash := sha256.Sum256([]byte(data.Encoding))
	buf = append(buf, payloadEncodingHash[:]...)
	payloadValueHash := sha256.Sum256(data.Value)
	buf = append(buf, payloadValueHash[:]...)
	hash := sha256.Sum256(buf)
	return hash[:]
}

// CommitAcknowledgement returns the V2 acknowledgement commitment bytes. The commitment consists of:
// sha256_hash(0x02 + sha256_hash(ack1) + sha256_hash(ack2) + ...) from a given acknowledgement.
func CommitAcknowledgement(acknowledgement Acknowledgement) []byte {
	h := sha256.New()
	_, _ = h.Write([]byte{2})

	for _, ack := range acknowledgement.GetAppAcknowledgements() {
		sum := sha256.Sum256(ack)
		_, _ = h.Write(sum[:])
	}

	return h.Sum(nil)
}
