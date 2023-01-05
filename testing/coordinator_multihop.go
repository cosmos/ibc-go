package ibctesting

import "github.com/stretchr/testify/require"

// CoordinatorM coordinates testing set up for multihop channels.
//
// method naming conventions following Coordinate:
//   - SetupXXX: create prerequisites of XXX and then create XXX. Eg. SetupConnections creates clients (prereq) and connections.
//   - CreateXXX: create XXX only without its preprequisites. Eg. CreateChannels creates channels, which fails if no connections exist.
type CoordinatorM struct {
	// Coordinator is the underlying coordinator used to create clients, connections, and channels for single-hop IBC.
	*Coordinator
}

// SetupChannels constructs TM clients, connections, and a multihop channel for a given multihop channel path.
// Fail test if any error occurs.
func (coord *CoordinatorM) SetupChannels(path *PathM) {
	coord.SetupConnections(path)

	// channels can also be referenced through the returned connections
	coord.CreateChannels(path)
}

// SetupConnections creates clients and then connections for each pair of chains in a multihop path.
// Fail test if any error occurs.
// Prerequisite: none of clients or connections has been created.
func (coord *CoordinatorM) SetupConnections(path *PathM) {
	// EndpointA and EndpointZ keeps opposite views of the same connections. So it's sufficient to just create
	// connections on EndpointA.paths.
	for _, path := range path.EndpointA.paths {
		path := path
		require.Empty(coord.T, path.EndpointA.ClientID)
		require.Empty(coord.T, path.EndpointB.ClientID)
		require.Empty(coord.T, path.EndpointA.ConnectionID)
		require.Empty(coord.T, path.EndpointB.ConnectionID)
		coord.Coordinator.SetupConnections(path)
	}
}

// CreateChannels constructs and executes channel handshake messages to create OPEN channels.
// Fail test if any error occurs.
// Prerequisite: clients and connections have been created.
func (coord *CoordinatorM) CreateChannels(path *PathM) {
	// create channels on path
	err := path.EndpointA.ChanOpenInit()
	require.NoError(coord.T, err)

	err = path.EndpointZ.ChanOpenTry()
	require.NoError(coord.T, err)

	err = path.EndpointA.ChanOpenAck()
	require.NoError(coord.T, err)

	err = path.EndpointZ.ChanOpenConfirm()
	require.NoError(coord.T, err)
}
