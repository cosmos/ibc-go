package types

import codectypes "github.com/cosmos/cosmos-sdk/codec/types"

// GetClientState returns the byte slice representation of the client state included in the connection open try message.
// Returns nil if the client state is not set.
func (m *MsgConnectionOpenTry) GetClientState() []byte {
	if m.ClientState == nil {
		return nil
	}
	return m.ClientState.Value
}

// SetClientState sets the client state in the connection open try message to the given value.
func (m *MsgConnectionOpenTry) SetClientState(state *codectypes.Any) {
	m.ClientState = state
}

// GetClientState returns the byte slice representation of the client state included in the connection open acknowledgement message.
// Returns nil if the client state is not set.
func (m *MsgConnectionOpenAck) GetClientState() []byte {
	if m.ClientState == nil {
		return nil
	}
	return m.ClientState.Value
}

// SetClientState sets the client state in the connection open acknowledgement message to the given value.
func (m *MsgConnectionOpenAck) SetClientState(state *codectypes.Any) {
	m.ClientState = state
}
