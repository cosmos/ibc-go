package types

import codectypes "github.com/cosmos/cosmos-sdk/codec/types"

// ClientStateMsg is an interface that defines methods for getting and setting the client state of a message.
type ClientStateMsg interface {
	// GetClientState returns the byte slice representation of the client state included in the message.
	// Returns nil if the client state is not set.
	GetClientState() []byte

	// SetClientState sets the client state in the message to the given value.
	SetClientState(state *codectypes.Any)
}

// GetClientState returns the byte slice representation of the client state included in the create client message.
// Returns nil if the client state is not set.
func (m *MsgCreateClient) GetClientState() []byte {
	if m.ClientState == nil {
		return nil
	}
	return m.ClientState.Value
}

// SetClientState sets the client state in the create client message to the given value.
func (m *MsgCreateClient) SetClientState(state *codectypes.Any) {
	m.ClientState = state
}

// GetClientState returns the byte slice representation of the client state included in the upgrade client message.
// Returns nil if the client state is not set.
func (m *MsgUpgradeClient) GetClientState() []byte {
	if m.ClientState == nil {
		return nil
	}
	return m.ClientState.Value
}

// SetClientState sets the client state in the upgrade client message to the given value.
func (m *MsgUpgradeClient) SetClientState(state *codectypes.Any) {
	m.ClientState = state
}
