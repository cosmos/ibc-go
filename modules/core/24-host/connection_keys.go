package host

import "fmt"

const KeyConnectionPrefix = "connections"

// ICS03
// The following paths are the keys to the store as defined in https://github.com/cosmos/ibc/blob/master/spec/core/ics-003-connection-semantics#store-paths

// ClientConnectionsKey returns the store key for the connections of a given client
func ClientConnectionsKey(clientID string) []byte {
	return FullClientKey(clientID, []byte(KeyConnectionPrefix))
}

// ConnectionKey returns the store key for a particular connection
func ConnectionKey(connectionID string) []byte {
	return []byte(fmt.Sprintf("%s/%s", KeyConnectionPrefix, connectionID))
}
