package host

import (
	"fmt"

	"github.com/cosmos/ibc-go/v10/modules/core/exported"
)

// KeyClientStorePrefix defines the KVStore key prefix for IBC clients
var KeyClientStorePrefix = []byte("clients")

const (
	KeyClientState          = "clientState"
	KeyConsensusStatePrefix = "consensusStates"
)

// FullClientKey returns the full path of specific client path in the format:
// "clients/{clientID}/{path}" as a byte array.
func FullClientKey(clientID string, path []byte) []byte {
	return fmt.Appendf(nil, "%s/%s/%s", KeyClientStorePrefix, clientID, path)
}

// PrefixedClientStoreKey returns a key which can be used for prefixed
// key store iteration. The prefix may be a clientType, clientID, or any
// valid key prefix which may be concatenated with the client store constant.
func PrefixedClientStoreKey(prefix []byte) []byte {
	return fmt.Appendf(nil, "%s/%s", KeyClientStorePrefix, prefix)
}

// FullClientStateKey takes a client identifier and returns a Key under which to store a
// particular client state.
func FullClientStateKey(clientID string) []byte {
	return FullClientKey(clientID, []byte(KeyClientState))
}

// ClientStateKey returns a store key under which a particular client state is stored
// in a client prefixed store
func ClientStateKey() []byte {
	return []byte(KeyClientState)
}

// FullConsensusStateKey returns the store key for the consensus state of a particular
// client.
func FullConsensusStateKey(clientID string, height exported.Height) []byte {
	return FullClientKey(clientID, ConsensusStateKey(height))
}

// ConsensusStateKey returns the store key for a the consensus state of a particular
// client stored in a client prefixed store.
func ConsensusStateKey(height exported.Height) []byte {
	return fmt.Appendf(nil, "%s/%s", KeyConsensusStatePrefix, height)
}
