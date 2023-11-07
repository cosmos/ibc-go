package testing

import (
	"errors"

	"github.com/cosmos/cosmos-sdk/codec"

	"github.com/cosmos/ibc-go/modules/light-clients/08-wasm/types"
	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
)

var (
	// Represents the code of the wasm contract used in the tests with a mock vm.
	Code                              = []byte("01234567012345670123456701234567")
	contractClientState               = []byte{1}
	contractConsensusState            = []byte{2}
	ErrMockContract                   = errors.New("mock contract error")
	MockClientStateBz                 = []byte("client-state-data")
	MockConsensusStateBz              = []byte("consensus-state-data")
	MockValidProofBz                  = []byte("valid proof")
	MockInvalidProofBz                = []byte("invalid proof")
	MockUpgradedClientStateProofBz    = []byte("upgraded client state proof")
	MockUpgradedConsensusStateProofBz = []byte("upgraded consensus state proof")
)

func CreateMockClientStateBz(cdc codec.BinaryCodec, codeHash []byte) []byte {
	mockClientSate := types.NewClientState([]byte{1}, codeHash, clienttypes.NewHeight(2000, 2))
	return clienttypes.MustMarshalClientState(cdc, mockClientSate)
}
