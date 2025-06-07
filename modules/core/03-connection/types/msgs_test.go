package types_test

import (
	"errors"
	"fmt"
	"testing"

	dbm "github.com/cosmos/cosmos-db"
	"github.com/stretchr/testify/require"
	testifysuite "github.com/stretchr/testify/suite"

	"cosmossdk.io/log"
	"cosmossdk.io/store/iavl"
	"cosmossdk.io/store/metrics"
	"cosmossdk.io/store/rootmulti"
	storetypes "cosmossdk.io/store/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"

	ibc "github.com/cosmos/ibc-go/v10/modules/core"
	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	"github.com/cosmos/ibc-go/v10/modules/core/03-connection/types"
	commitmenttypes "github.com/cosmos/ibc-go/v10/modules/core/23-commitment/types"
	host "github.com/cosmos/ibc-go/v10/modules/core/24-host"
	ibcerrors "github.com/cosmos/ibc-go/v10/modules/core/errors"
	"github.com/cosmos/ibc-go/v10/modules/core/exported"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
	"github.com/cosmos/ibc-go/v10/testing/simapp"
)

var (
	signer = "cosmos1ckgw5d7jfj7wwxjzs9fdrdev9vc8dzcw3n2lht"

	emptyPrefix = commitmenttypes.MerklePrefix{}
	emptyProof  = []byte{}
)

type MsgTestSuite struct {
	testifysuite.Suite

	coordinator *ibctesting.Coordinator

	chainA *ibctesting.TestChain
	chainB *ibctesting.TestChain

	proof []byte
}

func TestMsgTestSuite(t *testing.T) {
	testifysuite.Run(t, new(MsgTestSuite))
}

func (s *MsgTestSuite) SetupTest() {
	s.coordinator = ibctesting.NewCoordinator(s.T(), 2)

	s.chainA = s.coordinator.GetChain(ibctesting.GetChainID(1))
	s.chainB = s.coordinator.GetChain(ibctesting.GetChainID(2))

	app := simapp.Setup(s.T(), false)
	db := dbm.NewMemDB()
	store := rootmulti.NewStore(db, log.NewNopLogger(), metrics.NewNoOpMetrics())
	storeKey := storetypes.NewKVStoreKey("iavlStoreKey")

	store.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, nil)
	err := store.LoadVersion(0)
	s.Require().NoError(err)
	iavlStore, ok := store.GetCommitStore(storeKey).(*iavl.Store)
	s.Require().True(ok)

	iavlStore.Set([]byte("KEY"), []byte("VALUE"))
	_ = store.Commit()

	res, err := store.Query(&storetypes.RequestQuery{
		Data:  []byte("KEY"),
		Path:  fmt.Sprintf("/%s/key", storeKey.Name()), // required path to get key/value+proof
		Prove: true,
	})
	s.Require().NoError(err)

	merkleProof, err := commitmenttypes.ConvertProofs(res.ProofOps)
	s.Require().NoError(err)
	proof, err := app.AppCodec().Marshal(&merkleProof)
	s.Require().NoError(err)

	s.proof = proof
}

func (s *MsgTestSuite) TestNewMsgConnectionOpenInit() {
	prefix := commitmenttypes.NewMerklePrefix([]byte("storePrefixKey"))
	// empty versions are considered valid, the default compatible versions
	// will be used in protocol.
	var version *types.Version

	testCases := []struct {
		name     string
		msg      *types.MsgConnectionOpenInit
		expError error
	}{
		{"localhost client ID", types.NewMsgConnectionOpenInit(exported.LocalhostClientID, "clienttotest", prefix, version, 500, signer), clienttypes.ErrInvalidClientType},
		{"invalid client ID", types.NewMsgConnectionOpenInit("test/iris", "clienttotest", prefix, version, 500, signer), host.ErrInvalidID},
		{"invalid counterparty client ID", types.NewMsgConnectionOpenInit("clienttotest", "(clienttotest)", prefix, version, 500, signer), host.ErrInvalidID},
		{"invalid counterparty connection ID", &types.MsgConnectionOpenInit{connectionID, types.NewCounterparty("clienttotest", "connectiontotest", prefix), version, 500, signer}, types.ErrInvalidCounterparty},
		{"empty counterparty prefix", types.NewMsgConnectionOpenInit("clienttotest", "clienttotest", emptyPrefix, version, 500, signer), types.ErrInvalidCounterparty},
		{"supplied version fails basic validation", types.NewMsgConnectionOpenInit("clienttotest", "clienttotest", prefix, &types.Version{}, 500, signer), types.ErrInvalidVersion},
		{"empty singer", types.NewMsgConnectionOpenInit("clienttotest", "clienttotest", prefix, version, 500, ""), ibcerrors.ErrInvalidAddress},
		{"success", types.NewMsgConnectionOpenInit("clienttotest", "clienttotest", prefix, version, 500, signer), nil},
	}

	for _, tc := range testCases {
		err := tc.msg.ValidateBasic()

		if tc.expError == nil {
			s.Require().NoError(err, tc.name)
		} else {
			s.Require().ErrorIs(err, tc.expError)
		}
	}
}

func (s *MsgTestSuite) TestNewMsgConnectionOpenTry() {
	prefix := commitmenttypes.NewMerklePrefix([]byte("storePrefixKey"))

	testCases := []struct {
		name     string
		msg      *types.MsgConnectionOpenTry
		expError error
	}{
		{"success", types.NewMsgConnectionOpenTry("clienttotesta", "connectiontotest", "clienttotest", prefix, []*types.Version{ibctesting.ConnectionVersion}, 500, s.proof, clientHeight, signer), nil},
		{"localhost client ID", types.NewMsgConnectionOpenTry(exported.LocalhostClientID, "connectiontotest", "clienttotest", prefix, []*types.Version{ibctesting.ConnectionVersion}, 500, s.proof, clientHeight, signer), clienttypes.ErrInvalidClientType},
		{"invalid client ID", types.NewMsgConnectionOpenTry("test/iris", "connectiontotest", "clienttotest", prefix, []*types.Version{ibctesting.ConnectionVersion}, 500, s.proof, clientHeight, signer), host.ErrInvalidID},
		{"invalid counterparty connection ID", types.NewMsgConnectionOpenTry("clienttotesta", "ibc/test", "clienttotest", prefix, []*types.Version{ibctesting.ConnectionVersion}, 500, s.proof, clientHeight, signer), host.ErrInvalidID},
		{"invalid counterparty client ID", types.NewMsgConnectionOpenTry("clienttotesta", "connectiontotest", "test/conn1", prefix, []*types.Version{ibctesting.ConnectionVersion}, 500, s.proof, clientHeight, signer), host.ErrInvalidID},
		{"empty counterparty prefix", types.NewMsgConnectionOpenTry("clienttotesta", "connectiontotest", "clienttotest", emptyPrefix, []*types.Version{ibctesting.ConnectionVersion}, 500, s.proof, clientHeight, signer), types.ErrInvalidCounterparty},
		{"empty counterpartyVersions", types.NewMsgConnectionOpenTry("clienttotesta", "connectiontotest", "clienttotest", prefix, []*types.Version{}, 500, s.proof, clientHeight, signer), ibcerrors.ErrInvalidVersion},
		{"empty proofInit", types.NewMsgConnectionOpenTry("clienttotesta", "connectiontotest", "clienttotest", prefix, []*types.Version{ibctesting.ConnectionVersion}, 500, emptyProof, clientHeight, signer), commitmenttypes.ErrInvalidProof},
		{"empty singer", types.NewMsgConnectionOpenTry("clienttotesta", "connectiontotest", "clienttotest", prefix, []*types.Version{ibctesting.ConnectionVersion}, 500, s.proof, clientHeight, ""), ibcerrors.ErrInvalidAddress},
		{"invalid version", types.NewMsgConnectionOpenTry("clienttotesta", "connectiontotest", "clienttotest", prefix, []*types.Version{{}}, 500, s.proof, clientHeight, signer), types.ErrInvalidVersion},
		{"too many counterparty versions", types.NewMsgConnectionOpenTry("clienttotesta", "connectiontotest", "clienttotest", prefix, make([]*types.Version, types.MaxCounterpartyVersionsLength+1), 500, s.proof, clientHeight, signer), ibcerrors.ErrInvalidVersion},
		{"too many features in counterparty version", types.NewMsgConnectionOpenTry("clienttotesta", "connectiontotest", "clienttotest", prefix, []*types.Version{{"v1", make([]string, types.MaxFeaturesLength+1)}}, 500, s.proof, clientHeight, signer), types.ErrInvalidVersion},
	}

	for _, tc := range testCases {
		err := tc.msg.ValidateBasic()

		if tc.expError == nil {
			s.Require().NoError(err, tc.name)
		} else {
			s.Require().ErrorIs(err, tc.expError)
		}
	}
}

func (s *MsgTestSuite) TestNewMsgConnectionOpenAck() {
	testCases := []struct {
		name     string
		msg      *types.MsgConnectionOpenAck
		expError error
	}{
		{"success", types.NewMsgConnectionOpenAck(connectionID, connectionID, s.proof, clientHeight, ibctesting.ConnectionVersion, signer), nil},
		{"invalid connection ID", types.NewMsgConnectionOpenAck("test/conn1", connectionID, s.proof, clientHeight, ibctesting.ConnectionVersion, signer), types.ErrInvalidConnectionIdentifier},
		{"invalid counterparty connection ID", types.NewMsgConnectionOpenAck(connectionID, "test/conn1", s.proof, clientHeight, ibctesting.ConnectionVersion, signer), host.ErrInvalidID},
		{"empty proofTry", types.NewMsgConnectionOpenAck(connectionID, connectionID, emptyProof, clientHeight, ibctesting.ConnectionVersion, signer), commitmenttypes.ErrInvalidProof},
		{"invalid version", types.NewMsgConnectionOpenAck(connectionID, connectionID, s.proof, clientHeight, &types.Version{}, signer), types.ErrInvalidVersion},
		{"empty signer", types.NewMsgConnectionOpenAck(connectionID, connectionID, s.proof, clientHeight, ibctesting.ConnectionVersion, ""), ibcerrors.ErrInvalidAddress},
	}

	for _, tc := range testCases {
		err := tc.msg.ValidateBasic()

		if tc.expError == nil {
			s.Require().NoError(err, tc.name)
		} else {
			s.Require().ErrorIs(err, tc.expError)
		}
	}
}

func (s *MsgTestSuite) TestNewMsgConnectionOpenConfirm() {
	testCases := []struct {
		name     string
		msg      *types.MsgConnectionOpenConfirm
		expError error
	}{
		{"invalid connection ID", types.NewMsgConnectionOpenConfirm("test/conn1", s.proof, clientHeight, signer), types.ErrInvalidConnectionIdentifier},
		{"empty proofTry", types.NewMsgConnectionOpenConfirm(connectionID, emptyProof, clientHeight, signer), commitmenttypes.ErrInvalidProof},
		{"empty signer", types.NewMsgConnectionOpenConfirm(connectionID, s.proof, clientHeight, ""), ibcerrors.ErrInvalidAddress},
		{"success", types.NewMsgConnectionOpenConfirm(connectionID, s.proof, clientHeight, signer), nil},
	}

	for _, tc := range testCases {
		err := tc.msg.ValidateBasic()

		if tc.expError == nil {
			s.Require().NoError(err, tc.name)
		} else {
			s.Require().ErrorIs(err, tc.expError)
		}
	}
}

// TestMsgUpdateParamsValidateBasic tests ValidateBasic for MsgUpdateParams
func (s *MsgTestSuite) TestMsgUpdateParamsValidateBasic() {
	signer := s.chainA.App.GetIBCKeeper().GetAuthority()
	testCases := []struct {
		name     string
		msg      *types.MsgUpdateParams
		expError error
	}{
		{
			"success: valid signer and params",
			types.NewMsgUpdateParams(signer, types.DefaultParams()),
			nil,
		},
		{
			"failure: invalid signer address",
			types.NewMsgUpdateParams("invalid", types.DefaultParams()),
			ibcerrors.ErrInvalidAddress,
		},
		{
			"failure: invalid time per block",
			types.NewMsgUpdateParams(signer, types.NewParams(0)),
			errors.New("MaxExpectedTimePerBlock cannot be zero"),
		},
	}

	for _, tc := range testCases {
		err := tc.msg.ValidateBasic()
		if tc.expError == nil {
			s.Require().NoError(err, tc.name)
		} else {
			s.Require().ErrorContains(err, tc.expError.Error())
		}
	}
}

// TestMsgUpdateParamsGetSigners tests GetSigners for MsgUpdateParams
func TestMsgUpdateParamsGetSigners(t *testing.T) {
	testCases := []struct {
		name    string
		address sdk.AccAddress
		errMsg  string
	}{
		{"success: valid address", sdk.AccAddress(ibctesting.TestAccAddress), ""},
		{"failure: nil address", nil, "empty address string is not allowed"},
	}

	for _, tc := range testCases {
		msg := types.MsgUpdateParams{
			Signer: tc.address.String(),
			Params: types.DefaultParams(),
		}

		encodingCfg := moduletestutil.MakeTestEncodingConfig(ibc.AppModuleBasic{})
		signers, _, err := encodingCfg.Codec.GetMsgV1Signers(&msg)
		if tc.errMsg == "" {
			require.NoError(t, err)
			require.Equal(t, tc.address.Bytes(), signers[0])
		} else {
			require.ErrorContains(t, err, tc.errMsg)
		}
	}
}
