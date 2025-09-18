package ibctesting

import (
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

var (
	ChainIDPrefix = "testchain"
	// to disable revision format, set ChainIDSuffix to ""
	ChainIDSuffix   = "-1"
	globalStartTime = time.Date(2020, 1, 2, 0, 0, 0, 0, time.UTC)
	TimeIncrement   = time.Second * 5
)

// Coordinator is a testing struct which contains N TestChain's. It handles keeping all chains
// in sync with regards to time.
type Coordinator struct {
	*testing.T

	CurrentTime time.Time
	Chains      map[string]*TestChain
}

// NewCoordinator initializes Coordinator with N TestChain's
func NewCoordinator(t *testing.T, n int) *Coordinator {
	t.Helper()
	return NewCustomAppCoordinator(t, n, DefaultTestingAppInit)
}

// NewCustomAppCoordinator initializes a Coordinator with N TestChain's using the given AppCreator function.
func NewCustomAppCoordinator(t *testing.T, n int, appCreator AppCreator) *Coordinator {
	t.Helper()
	chains := make(map[string]*TestChain)
	coord := &Coordinator{
		T:           t,
		CurrentTime: globalStartTime,
	}

	for i := 1; i <= n; i++ {
		chainID := GetChainID(i)
		chains[chainID] = NewCustomAppTestChain(t, coord, chainID, appCreator)
	}
	coord.Chains = chains

	return coord
}

// IncrementTime iterates through all the TestChain's and increments their current header time
// by 5 seconds.
//
// CONTRACT: this function must be called after every Commit on any TestChain.
func (c *Coordinator) IncrementTime() {
	c.IncrementTimeBy(TimeIncrement)
}

// IncrementTimeBy iterates through all the TestChain's and increments their current header time
// by specified time.
func (c *Coordinator) IncrementTimeBy(increment time.Duration) {
	c.CurrentTime = c.CurrentTime.Add(increment).UTC()
	c.UpdateTime()
}

// SetTime sets the coordinator's current time to the specified time and updates
// the proposed header time for all chains.
func (c *Coordinator) SetTime(t time.Time) {
	c.CurrentTime = t.UTC()
	c.UpdateTime()
}

// UpdateTime updates all clocks for the TestChains to the current global time.
func (c *Coordinator) UpdateTime() {
	for _, chain := range c.Chains {
		c.UpdateTimeForChain(chain)
	}
}

// UpdateTimeForChain updates the clock for a specific chain.
func (c *Coordinator) UpdateTimeForChain(chain *TestChain) {
	chain.ProposedHeader.Time = c.CurrentTime.UTC()
}

// Setup constructs a TM client, connection, and channel on both chains provided. It will
// fail if any error occurs.
// Deprecated: please use path.Setup(), this function will be removed in v10
func (*Coordinator) Setup(path *Path) {
	path.Setup()
}

// SetupClients is a helper function to create clients on both chains. It assumes the
// caller does not anticipate any errors.
// Deprecated: please use path.SetupClients(), this function will be removed in v10
func (*Coordinator) SetupClients(path *Path) {
	path.SetupClients()
}

// SetupConnections is a helper function to create clients and the appropriate
// connections on both the source and counterparty chain. It assumes the caller does not
// anticipate any errors.
// Deprecated: please use path.SetupConnections(), this function will be removed in v10
func (*Coordinator) SetupConnections(path *Path) {
	path.SetupConnections()
}

// CreateConnections constructs and executes connection handshake messages in order to create
// OPEN channels on chainA and chainB. The connection information of for chainA and chainB
// are returned within a TestConnection struct. The function expects the connections to be
// successfully opened otherwise testing will fail.
// Deprecated: please use path.CreateConnections(), this function will be removed in v10
func (*Coordinator) CreateConnections(path *Path) {
	path.CreateConnections()
}

// CreateMockChannels constructs and executes channel handshake messages to create OPEN
// channels that use a mock application module that returns nil on all callbacks. This
// function is expects the channels to be successfully opened otherwise testing will
// fail.
func (*Coordinator) CreateMockChannels(path *Path) {
	path.EndpointA.ChannelConfig.PortID = MockPort
	path.EndpointB.ChannelConfig.PortID = MockPort

	path.CreateChannels()
}

// CreateTransferChannels constructs and executes channel handshake messages to create OPEN
// ibc-transfer channels on chainA and chainB. The function expects the channels to be
// successfully opened otherwise testing will fail.
func (*Coordinator) CreateTransferChannels(path *Path) {
	path.EndpointA.ChannelConfig.PortID = TransferPort
	path.EndpointB.ChannelConfig.PortID = TransferPort

	path.CreateChannels()
}

// GetChain returns the TestChain using the given chainID and returns an error if it does
// not exist.
func (c *Coordinator) GetChain(chainID string) *TestChain {
	chain, found := c.Chains[chainID]
	require.True(c.T, found, "%s chain does not exist", chainID)
	return chain
}

// GetChainID returns the chainID used for the provided index.
func GetChainID(index int) string {
	return ChainIDPrefix + strconv.Itoa(index) + ChainIDSuffix
}

// CommitBlock commits a block on the provided indexes and then increments the global time.
//
// CONTRACT: the passed in list of indexes must not contain duplicates
func (c *Coordinator) CommitBlock(chains ...*TestChain) {
	for _, chain := range chains {
		chain.NextBlock()
	}
	c.IncrementTime()
}

// CommitNBlocks commits n blocks to state and updates the block height by 1 for each commit.
func (c *Coordinator) CommitNBlocks(chain *TestChain, n uint64) {
	for range n {
		chain.NextBlock()
		c.IncrementTime()
	}
}
