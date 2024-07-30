package host

import (
	"github.com/cosmos/ibc-go/v9/modules/core/exported"
)

// FullClientKey returns the full path of specific client path in the format:
// "clients/{clientID}/{path}" as a byte array.
func FullClientKey(clientID string, path []byte) []byte {
	return []byte(FullClientPath(clientID, string(path)))
}

// PrefixedClientStoreKey returns a key which can be used for prefixed
// key store iteration. The prefix may be a clientType, clientID, or any
// valid key prefix which may be concatenated with the client store constant.
func PrefixedClientStoreKey(prefix []byte) []byte {
	return []byte(PrefixedClientStorePath(prefix))
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
	return []byte(FullConsensusStatePath(clientID, height))
}

// ConsensusStateKey returns the store key for a the consensus state of a particular
// client stored in a client prefixed store.
func ConsensusStateKey(height exported.Height) []byte {
	return []byte(ConsensusStatePath(height))
}
