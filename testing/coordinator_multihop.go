package ibctesting

import (
	"fmt"
	"math/rand"

	"github.com/stretchr/testify/require"
)

// CoordinatorM coordinates testing set up for multihop channels.
//
// method naming conventions following Coordinate:
//   - SetupXXX: create prerequisites of XXX and then create XXX. Eg. SetupConnections creates clients (prereq) and connections.
//   - CreateXXX: create XXX only without its preprequisites. Eg. CreateChannels creates channels, which fails if no connections exist.
type CoordinatorM struct {
	// Coordinator is the underlying coordinator used to create clients, connections, and channels for single-hop IBC.
	*Coordinator
}

// SetupClients is a helper function to create clients on both chains. It assumes the
// caller does not anticipate any errors.
func (coord *CoordinatorM) SetupClients(path *PathM) {
	for _, path := range path.EndpointA.Paths {
		require.Empty(coord.T, path.EndpointA.ClientID)
		require.Empty(coord.T, path.EndpointB.ClientID)
		require.Empty(coord.T, path.EndpointA.ConnectionID)
		require.Empty(coord.T, path.EndpointB.ConnectionID)
		// add variability to the clientIDs
		N := rand.Int() % 5
		for n := 0; n <= N; n++ {
			coord.Coordinator.SetupClients(path)
		}
	}
}

// SetupClientConnections is a helper function to create clients and the appropriate
// connections on both the source and counterparty chain. It assumes the caller does not
// anticipate any errors.
func (coord *CoordinatorM) SetupConnections(path *PathM) {
	coord.SetupClients(path)
	coord.CreateConnections(path)
}

func (coord *CoordinatorM) SetupAllButTheSpecifiedConnection(path *PathM, index int) error {
	if index >= len(path.EndpointA.Paths) {
		return fmt.Errorf("SetupAllButTheSpecifiedConnection(): invalid index parameter %d", index)
	}

	for _, path := range path.EndpointA.Paths[:index] {
		// add variability to the clientIDs
		N := rand.Int() % 5
		for n := 0; n <= N; n++ {
			coord.Coordinator.SetupClients(path)
		}
		coord.Coordinator.CreateConnections(path)
	}

	for _, path := range path.EndpointA.Paths[index+1:] {
		// add variability to the clientIDs
		N := rand.Int() % 5
		for n := 0; n <= N; n++ {
			coord.Coordinator.SetupClients(path)
		}
		coord.Coordinator.CreateConnections(path)
	}

	return nil
}

// CreateConnection constructs and executes connection handshake messages in order to create
// OPEN channels on chainA and chainB. The connection information of for chainA and chainB
// are returned within a TestConnection struct. The function expects the connections to be
// successfully opened otherwise testing will fail.
func (coord *CoordinatorM) CreateConnections(path *PathM) {
	for _, path := range path.EndpointA.Paths {
		path := path
		coord.Coordinator.CreateConnections(path)
	}
}

// SetupChannels constructs TM clients, connections, and a multihop channel for a given multihop channel path.
// Fail test if any error occurs.
func (coord *CoordinatorM) SetupChannels(path *PathM) {
	coord.SetupConnections(path)

	// channels can also be referenced through the returned connections
	coord.CreateChannels(path)
}

// CreateChannels constructs and executes channel handshake messages to create OPEN channels.
// Fail test if any error occurs.
// Prerequisite: clients and connections have been created.
func (coord *CoordinatorM) CreateChannels(path *PathM) {
	// create channels on path
	err := path.EndpointA.ChanOpenInit()
	require.NoError(coord.T, err)

	err = path.EndpointZ.ChanOpenTry(path.EndpointA.Chain.LastHeader.GetHeight())
	require.NoError(coord.T, err)

	err = path.EndpointA.ChanOpenAck(path.EndpointZ.Chain.LastHeader.GetHeight())
	require.NoError(coord.T, err)

	err = path.EndpointZ.ChanOpenConfirm(path.EndpointA.Chain.LastHeader.GetHeight())
	require.NoError(coord.T, err)
}
