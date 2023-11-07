package testing

import "errors"

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
