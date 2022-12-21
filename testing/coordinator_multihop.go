package ibctesting

import "github.com/stretchr/testify/require"

// CoordinatorM coordinates testing set up for multihop channels.
type CoordinatorM struct {
	*Coordinator
}

// Setup constructs TM clients, connections, and a multihop channel for a given multihop channel path.
// Fail test if any error occurs.
func (coord *CoordinatorM) Setup(path *PathM) {
	// coord.SetupConnections(path)

	// channels can also be referenced through the returned connections
	coord.CreateChannels(path)
}

// CreateChannels constructs and executes channel handshake messages to create OPEN channels.
// Fail test if any error occurs.
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
