package testing

import (
	"errors"
	"time"

	"github.com/cosmos/cosmos-sdk/codec"

	"github.com/cosmos/ibc-go/modules/light-clients/08-wasm/types"
	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	commitmenttypes "github.com/cosmos/ibc-go/v8/modules/core/23-commitment/types"
	ibctm "github.com/cosmos/ibc-go/v8/modules/light-clients/07-tendermint"
	testing "github.com/cosmos/ibc-go/v8/testing"
)

var (
	// Represents the code of the wasm contract used in the tests with a mock vm.
	WasmMagicNumber                   = []byte("\x00\x61\x73\x6D")
	Code                              = append(WasmMagicNumber, []byte("0123456780123456780123456780")...)
	MockClientStateBz                 = []byte("client-state-data")
	MockConsensusStateBz              = []byte("consensus-state-data")
	MockWrappedClientState            = CreateMockWrappedClientState(clienttypes.NewHeight(1, 10))
	MockWrappedClientHeader           = &ibctm.Header{}
	MockWrappedClientMisbehaviour     = ibctm.NewMisbehaviour("client-id", MockWrappedClientHeader, MockWrappedClientHeader)
	MockWrappedClientConsensusState   = ibctm.NewConsensusState(time.Now(), commitmenttypes.NewMerkleRoot([]byte("hash")), []byte("nextValsHash"))
	MockValidProofBz                  = []byte("valid proof")
	MockInvalidProofBz                = []byte("invalid proof")
	MockUpgradedClientStateProofBz    = []byte("upgraded client state proof")
	MockUpgradedConsensusStateProofBz = []byte("upgraded consensus state proof")

	ErrMockContract = errors.New("mock contract error")
	ErrMockVM       = errors.New("mock vm error")
)

// CreateMockWrappedClientState returns valid wrapped client state for use in tests.
func CreateMockWrappedClientState(height clienttypes.Height) *ibctm.ClientState {
	return ibctm.NewClientState(
		"chain-id",
		ibctm.DefaultTrustLevel,
		testing.TrustingPeriod,
		testing.UnbondingPeriod,
		testing.MaxClockDrift,
		height,
		commitmenttypes.GetSDKSpecs(),
		testing.UpgradePath,
	)
}

// CreateMockClientStateBz returns valid client state bytes for use in tests.
func CreateMockClientStateBz(cdc codec.BinaryCodec, checksum types.Checksum) []byte {
	wrappedClientStateBz := clienttypes.MustMarshalClientState(cdc, MockWrappedClientState)
	mockClientSate := types.NewClientState(wrappedClientStateBz, checksum, MockWrappedClientState.GetLatestHeight().(clienttypes.Height))
	return clienttypes.MustMarshalClientState(cdc, mockClientSate)
}
