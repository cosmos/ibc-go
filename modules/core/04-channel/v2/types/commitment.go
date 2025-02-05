package types

import (
	"crypto/sha256"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

var FailedAcknowledgementCommitment = sha256.Sum256([]byte{byte(2), byte(0)})

// CommitPacket returns the V2 packet commitment bytes. The commitment consists of:
// ha256_hash(0x02 + sha256_hash(destinationClient) + sha256_hash(timeout) + sha256_hash(payload)) from a given packet.
// This results in a fixed length preimage of 32 bytes.
// NOTE: A fixed length preimage is ESSENTIAL to prevent relayers from being able
// to malleate the packet fields and create a commitment hash that matches the original packet.
func CommitPacket(packet Packet) []byte {
	var buf []byte
	destIDHash := sha256.Sum256([]byte(packet.DestinationClient))
	buf = append(buf, destIDHash[:]...)

	timeoutBytes := sdk.Uint64ToBigEndian(packet.GetTimeoutTimestamp())
	timeoutHash := sha256.Sum256(timeoutBytes)
	buf = append(buf, timeoutHash[:]...)

	var appBytes []byte
	for _, payload := range packet.Payloads {
		appBytes = append(appBytes, hashPayload(payload)...)
	}
	appHash := sha256.Sum256(appBytes)
	buf = append(buf, appHash[:]...)

	buf = append([]byte{byte(2)}, buf...)

	hash := sha256.Sum256(buf)
	return hash[:]
}

// hashPayload returns the hash of the payload.
func hashPayload(data Payload) []byte {
	var buf []byte
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
	// return constant failed acknowledgement commitment if the acknowledgement is unsuccessful
	// in this case, application modules are expected to only use recv success value
	// and app acknowledgement will be nil
	if !acknowledgement.RecvSuccess {
		return FailedAcknowledgementCommitment[:]
	}
	var buf []byte
	for _, ack := range acknowledgement.GetAppAcknowledgements() {
		hash := sha256.Sum256(ack)
		buf = append(buf, hash[:]...)
	}

	// prepend the preimage with the IBC Version and the recv success
	// to ensure there is no possibility of hash collisions
	buf = append([]byte{byte(2), byte(1)}, buf...)

	hash := sha256.Sum256(buf)
	return hash[:]
}
