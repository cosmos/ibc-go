package types

import (
	"fmt"

	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	clienttypes "github.com/cosmos/ibc-go/v5/modules/core/02-client/types"
	"github.com/cosmos/ibc-go/v5/modules/core/exported"
)

var _ exported.ConsensusState = (*ConsensusState)(nil)

func (m *ConsensusState) ClientType() string {
	return exported.Wasm
}

func (m *ConsensusState) GetRoot() exported.Root {
	return m.Root
}

func (m *ConsensusState) GetTimestamp() uint64 {
	return m.Timestamp
}

func (m *ConsensusState) ValidateBasic() error {
	if m.Root.Empty() {
		return sdkerrors.Wrap(clienttypes.ErrInvalidConsensus, "root cannot be empty")
	}
	if m.Timestamp == 0 {
		return sdkerrors.Wrap(clienttypes.ErrInvalidConsensus, "timestamp cannot be zero Unix time")
	}

	if m.Data == nil || len(m.Data) == 0 {
		return fmt.Errorf("data cannot be empty")
	}

	if m.CodeId == nil || len(m.CodeId) == 0 {
		return fmt.Errorf("codeid cannot be empty")
	}

	return nil
}
