package host

// ClientConnectionsKey returns the store key for the connections of a given client
func ClientConnectionsKey(clientID string) []byte {
	return []byte(ClientConnectionsPath(clientID))
}

// ConnectionKey returns the store key for a particular connection
func ConnectionKey(connectionID string) []byte {
	return []byte(ConnectionPath(connectionID))
}
