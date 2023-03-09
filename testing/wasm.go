package ibctesting

import (
	"testing"

	"github.com/cosmos/cosmos-sdk/codec"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/cosmos/cosmos-sdk/crypto/types/multisig"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	"github.com/stretchr/testify/require"

	clienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	commitmenttypes "github.com/cosmos/ibc-go/v7/modules/core/23-commitment/types"
	host "github.com/cosmos/ibc-go/v7/modules/core/24-host"
	"github.com/cosmos/ibc-go/v7/modules/core/exported"
	wasmtypes "github.com/cosmos/ibc-go/v7/modules/light-clients/08-wasm/types"
)

// Wasm is a testing helper used to simulate a counterparty
// wasm machine client.
type Wasm struct {
	t *testing.T

	cdc         codec.BinaryCodec
	ClientID    string
	PrivateKeys []cryptotypes.PrivKey // keys used for signing
	PublicKeys  []cryptotypes.PubKey  // keys used for generating wasm machine pub key
	PublicKey   cryptotypes.PubKey    // key used for verification
	Sequence    uint64
	Time        uint64
	Diversifier string
}

// NewWasm returns a new wasm instance with an `nKeys` amount of
// generated private/public key pairs and a sequence starting at 1. If nKeys
// is greater than 1 then a multisig public key is used.
func NewWasm(t *testing.T, cdc codec.BinaryCodec, clientID, diversifier string, nKeys uint64) *Wasm {
	privKeys, pubKeys, pk := GenerateKeys(t, nKeys)

	return &Wasm{
		t:           t,
		cdc:         cdc,
		ClientID:    clientID,
		PrivateKeys: privKeys,
		PublicKeys:  pubKeys,
		PublicKey:   pk,
		Sequence:    1,
		Time:        10,
		Diversifier: diversifier,
	}
}

// ClientState returns a new wasm machine ClientState instance.
func (wasm *Wasm) ClientState() *wasmtypes.ClientState {
	return wasmtypes.NewClientState([]byte{0}, []byte{}, clienttypes.Height{})
}

// ConsensusState returns a new wasm machine ConsensusState instance
func (wasm *Wasm) ConsensusState() *wasmtypes.ConsensusState {
	return &wasmtypes.ConsensusState{
		Timestamp: wasm.Time,
	}
}

// GetHeight returns an exported.Height with Sequence as RevisionHeight
func (wasm *Wasm) GetHeight() exported.Height {
	return clienttypes.NewHeight(0, wasm.Sequence)
}

// CreateHeader generates a new private/public key pair and creates the
// necessary signature to construct a valid wasm machine header.
// A new diversifier will be used as well
func (wasm *Wasm) CreateHeader(newDiversifier string) *wasmtypes.Header {
	header := &wasmtypes.Header{
		Data:   []byte(newDiversifier),
		Height: clienttypes.Height{},
	}

	// assumes successful header update
	wasm.Sequence++
	wasm.Diversifier = newDiversifier

	return header
}

// CreateMisbehaviour constructs testing misbehaviour for the wasm machine client
// by signing over two different data bytes at the same sequence.
func (wasm *Wasm) CreateMisbehaviour() *wasmtypes.Misbehaviour {
	// TODO: Wasm.CreateMisbehaviour
	return nil
}

// GenerateSignature uses the stored private keys to generate a signature
// over the sign bytes with each key. If the amount of keys is greater than
// 1 then a multisig data type is returned.
func (wasm *Wasm) GenerateSignature(signBytes []byte) []byte {
	sigs := make([]signing.SignatureData, len(wasm.PrivateKeys))
	for i, key := range wasm.PrivateKeys {
		sig, err := key.Sign(signBytes)
		require.NoError(wasm.t, err)

		sigs[i] = &signing.SingleSignatureData{
			Signature: sig,
		}
	}

	var sigData signing.SignatureData
	if len(sigs) == 1 {
		// single public key
		sigData = sigs[0]
	} else {
		// generate multi signature data
		multiSigData := multisig.NewMultisig(len(sigs))
		for i, sig := range sigs {
			multisig.AddSignature(multiSigData, sig, i)
		}

		sigData = multiSigData
	}

	protoSigData := signing.SignatureDataToProto(sigData)
	bz, err := wasm.cdc.Marshal(protoSigData)
	require.NoError(wasm.t, err)

	return bz
}

// GetClientStatePath returns the commitment path for the client state.
func (wasm *Wasm) GetClientStatePath(counterpartyClientIdentifier string) commitmenttypes.MerklePath {
	path, err := commitmenttypes.ApplyPrefix(prefix, commitmenttypes.NewMerklePath(host.FullClientStatePath(counterpartyClientIdentifier)))
	require.NoError(wasm.t, err)

	return path
}

// GetConsensusStatePath returns the commitment path for the consensus state.
func (wasm *Wasm) GetConsensusStatePath(counterpartyClientIdentifier string, consensusHeight exported.Height) commitmenttypes.MerklePath {
	path, err := commitmenttypes.ApplyPrefix(prefix, commitmenttypes.NewMerklePath(host.FullConsensusStatePath(counterpartyClientIdentifier, consensusHeight)))
	require.NoError(wasm.t, err)

	return path
}

// GetConnectionStatePath returns the commitment path for the connection state.
func (wasm *Wasm) GetConnectionStatePath(connID string) commitmenttypes.MerklePath {
	connectionPath := commitmenttypes.NewMerklePath(host.ConnectionPath(connID))
	path, err := commitmenttypes.ApplyPrefix(prefix, connectionPath)
	require.NoError(wasm.t, err)

	return path
}

// GetChannelStatePath returns the commitment path for that channel state.
func (wasm *Wasm) GetChannelStatePath(portID, channelID string) commitmenttypes.MerklePath {
	channelPath := commitmenttypes.NewMerklePath(host.ChannelPath(portID, channelID))
	path, err := commitmenttypes.ApplyPrefix(prefix, channelPath)
	require.NoError(wasm.t, err)

	return path
}

// GetPacketCommitmentPath returns the commitment path for a packet commitment.
func (wasm *Wasm) GetPacketCommitmentPath(portID, channelID string) commitmenttypes.MerklePath {
	commitmentPath := commitmenttypes.NewMerklePath(host.PacketCommitmentPath(portID, channelID, wasm.Sequence))
	path, err := commitmenttypes.ApplyPrefix(prefix, commitmentPath)
	require.NoError(wasm.t, err)

	return path
}

// GetPacketAcknowledgementPath returns the commitment path for a packet acknowledgement.
func (wasm *Wasm) GetPacketAcknowledgementPath(portID, channelID string) commitmenttypes.MerklePath {
	ackPath := commitmenttypes.NewMerklePath(host.PacketAcknowledgementPath(portID, channelID, wasm.Sequence))
	path, err := commitmenttypes.ApplyPrefix(prefix, ackPath)
	require.NoError(wasm.t, err)

	return path
}

// GetPacketReceiptPath returns the commitment path for a packet receipt
// and an absent receipts.
func (wasm *Wasm) GetPacketReceiptPath(portID, channelID string) commitmenttypes.MerklePath {
	receiptPath := commitmenttypes.NewMerklePath(host.PacketReceiptPath(portID, channelID, wasm.Sequence))
	path, err := commitmenttypes.ApplyPrefix(prefix, receiptPath)
	require.NoError(wasm.t, err)

	return path
}

// GetNextSequenceRecvPath returns the commitment path for the next sequence recv counter.
func (wasm *Wasm) GetNextSequenceRecvPath(portID, channelID string) commitmenttypes.MerklePath {
	nextSequenceRecvPath := commitmenttypes.NewMerklePath(host.NextSequenceRecvPath(portID, channelID))
	path, err := commitmenttypes.ApplyPrefix(prefix, nextSequenceRecvPath)
	require.NoError(wasm.t, err)

	return path
}
