package types_test

import (
	"testing"
	"time"

	"github.com/golang/protobuf/proto" //nolint:staticcheck
	"github.com/stretchr/testify/require"
	testifysuite "github.com/stretchr/testify/suite"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	commitmenttypes "github.com/cosmos/ibc-go/v7/modules/core/23-commitment/types"
	solomachine "github.com/cosmos/ibc-go/v7/modules/light-clients/06-solomachine"
	ibctm "github.com/cosmos/ibc-go/v7/modules/light-clients/07-tendermint"
	ibctesting "github.com/cosmos/ibc-go/v7/testing"
)

type TypesTestSuite struct {
	testifysuite.Suite

	coordinator *ibctesting.Coordinator

	chainA      *ibctesting.TestChain
	chainB      *ibctesting.TestChain
	solomachine *ibctesting.Solomachine
}

func (s *TypesTestSuite) SetupTest() {
	s.coordinator = ibctesting.NewCoordinator(s.T(), 2)
	s.chainA = s.coordinator.GetChain(ibctesting.GetChainID(1))
	s.chainB = s.coordinator.GetChain(ibctesting.GetChainID(2))
	s.solomachine = ibctesting.NewSolomachine(s.T(), s.chainA.Codec, "solomachinesingle", "testing", 1)
}

func TestTypesTestSuite(t *testing.T) {
	testifysuite.Run(t, new(TypesTestSuite))
}

// tests that different clients within MsgCreateClient can be marshaled
// and unmarshaled.
func (s *TypesTestSuite) TestMarshalMsgCreateClient() {
	var (
		msg *types.MsgCreateClient
		err error
	)

	testCases := []struct {
		name     string
		malleate func()
	}{
		{
			"solo machine client", func() {
				soloMachine := ibctesting.NewSolomachine(s.T(), s.chainA.Codec, "solomachine", "", 2)
				msg, err = types.NewMsgCreateClient(soloMachine.ClientState(), soloMachine.ConsensusState(), s.chainA.SenderAccount.GetAddress().String())
				s.Require().NoError(err)
			},
		},
		{
			"tendermint client", func() {
				tendermintClient := ibctm.NewClientState(s.chainA.ChainID, ibctesting.DefaultTrustLevel, ibctesting.TrustingPeriod, ibctesting.UnbondingPeriod, ibctesting.MaxClockDrift, clientHeight, commitmenttypes.GetSDKSpecs(), ibctesting.UpgradePath)
				msg, err = types.NewMsgCreateClient(tendermintClient, s.chainA.CurrentTMClientHeader().ConsensusState(), s.chainA.SenderAccount.GetAddress().String())
				s.Require().NoError(err)
			},
		},
	}

	for _, tc := range testCases {
		tc := tc

		s.Run(tc.name, func() {
			s.SetupTest()

			tc.malleate()

			cdc := s.chainA.App.AppCodec()

			// marshal message
			bz, err := cdc.MarshalJSON(msg)
			s.Require().NoError(err)

			// unmarshal message
			newMsg := &types.MsgCreateClient{}
			err = cdc.UnmarshalJSON(bz, newMsg)
			s.Require().NoError(err)

			s.Require().True(proto.Equal(msg, newMsg))
		})
	}
}

func (s *TypesTestSuite) TestMsgCreateClient_ValidateBasic() {
	var (
		msg = &types.MsgCreateClient{}
		err error
	)

	cases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{
			"valid - tendermint client",
			func() {
				tendermintClient := ibctm.NewClientState(s.chainA.ChainID, ibctesting.DefaultTrustLevel, ibctesting.TrustingPeriod, ibctesting.UnbondingPeriod, ibctesting.MaxClockDrift, clientHeight, commitmenttypes.GetSDKSpecs(), ibctesting.UpgradePath)
				msg, err = types.NewMsgCreateClient(tendermintClient, s.chainA.CurrentTMClientHeader().ConsensusState(), s.chainA.SenderAccount.GetAddress().String())
				s.Require().NoError(err)
			},
			true,
		},
		{
			"invalid tendermint client",
			func() {
				msg, err = types.NewMsgCreateClient(&ibctm.ClientState{}, s.chainA.CurrentTMClientHeader().ConsensusState(), s.chainA.SenderAccount.GetAddress().String())
				s.Require().NoError(err)
			},
			false,
		},
		{
			"failed to unpack client",
			func() {
				msg.ClientState = nil
			},
			false,
		},
		{
			"failed to unpack consensus state",
			func() {
				tendermintClient := ibctm.NewClientState(s.chainA.ChainID, ibctesting.DefaultTrustLevel, ibctesting.TrustingPeriod, ibctesting.UnbondingPeriod, ibctesting.MaxClockDrift, clientHeight, commitmenttypes.GetSDKSpecs(), ibctesting.UpgradePath)
				msg, err = types.NewMsgCreateClient(tendermintClient, s.chainA.CurrentTMClientHeader().ConsensusState(), s.chainA.SenderAccount.GetAddress().String())
				s.Require().NoError(err)
				msg.ConsensusState = nil
			},
			false,
		},
		{
			"invalid signer",
			func() {
				msg.Signer = ""
			},
			false,
		},
		{
			"valid - solomachine client",
			func() {
				soloMachine := ibctesting.NewSolomachine(s.T(), s.chainA.Codec, "solomachine", "", 2)
				msg, err = types.NewMsgCreateClient(soloMachine.ClientState(), soloMachine.ConsensusState(), s.chainA.SenderAccount.GetAddress().String())
				s.Require().NoError(err)
			},
			true,
		},
		{
			"invalid solomachine client",
			func() {
				soloMachine := ibctesting.NewSolomachine(s.T(), s.chainA.Codec, "solomachine", "", 2)
				msg, err = types.NewMsgCreateClient(&solomachine.ClientState{}, soloMachine.ConsensusState(), s.chainA.SenderAccount.GetAddress().String())
				s.Require().NoError(err)
			},
			false,
		},
		{
			"invalid solomachine consensus state",
			func() {
				soloMachine := ibctesting.NewSolomachine(s.T(), s.chainA.Codec, "solomachine", "", 2)
				msg, err = types.NewMsgCreateClient(soloMachine.ClientState(), &solomachine.ConsensusState{}, s.chainA.SenderAccount.GetAddress().String())
				s.Require().NoError(err)
			},
			false,
		},
		{
			"invalid - client state and consensus state client types do not match",
			func() {
				tendermintClient := ibctm.NewClientState(s.chainA.ChainID, ibctesting.DefaultTrustLevel, ibctesting.TrustingPeriod, ibctesting.UnbondingPeriod, ibctesting.MaxClockDrift, clientHeight, commitmenttypes.GetSDKSpecs(), ibctesting.UpgradePath)
				soloMachine := ibctesting.NewSolomachine(s.T(), s.chainA.Codec, "solomachine", "", 2)
				msg, err = types.NewMsgCreateClient(tendermintClient, soloMachine.ConsensusState(), s.chainA.SenderAccount.GetAddress().String())
				s.Require().NoError(err)
			},
			false,
		},
	}

	for _, tc := range cases {
		tc.malleate()
		err = msg.ValidateBasic()
		if tc.expPass {
			s.Require().NoError(err, tc.name)
		} else {
			s.Require().Error(err, tc.name)
		}
	}
}

// tests that different header within MsgUpdateClient can be marshaled
// and unmarshaled.
func (s *TypesTestSuite) TestMarshalMsgUpdateClient() {
	var (
		msg *types.MsgUpdateClient
		err error
	)

	testCases := []struct {
		name     string
		malleate func()
	}{
		{
			"solo machine client", func() {
				soloMachine := ibctesting.NewSolomachine(s.T(), s.chainA.Codec, "solomachine", "", 2)
				msg, err = types.NewMsgUpdateClient(soloMachine.ClientID, soloMachine.CreateHeader(soloMachine.Diversifier), s.chainA.SenderAccount.GetAddress().String())
				s.Require().NoError(err)
			},
		},
		{
			"tendermint client", func() {
				msg, err = types.NewMsgUpdateClient("tendermint", s.chainA.CurrentTMClientHeader(), s.chainA.SenderAccount.GetAddress().String())
				s.Require().NoError(err)
			},
		},
	}

	for _, tc := range testCases {
		tc := tc

		s.Run(tc.name, func() {
			s.SetupTest()

			tc.malleate()

			cdc := s.chainA.App.AppCodec()

			// marshal message
			bz, err := cdc.MarshalJSON(msg)
			s.Require().NoError(err)

			// unmarshal message
			newMsg := &types.MsgUpdateClient{}
			err = cdc.UnmarshalJSON(bz, newMsg)
			s.Require().NoError(err)

			s.Require().True(proto.Equal(msg, newMsg))
		})
	}
}

func (s *TypesTestSuite) TestMsgUpdateClient_ValidateBasic() {
	var (
		msg = &types.MsgUpdateClient{}
		err error
	)

	cases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{
			"invalid client-id",
			func() {
				msg.ClientId = ""
			},
			false,
		},
		{
			"valid - tendermint header",
			func() {
				msg, err = types.NewMsgUpdateClient("tendermint", s.chainA.CurrentTMClientHeader(), s.chainA.SenderAccount.GetAddress().String())
				s.Require().NoError(err)
			},
			true,
		},
		{
			"invalid tendermint header",
			func() {
				msg, err = types.NewMsgUpdateClient("tendermint", &ibctm.Header{}, s.chainA.SenderAccount.GetAddress().String())
				s.Require().NoError(err)
			},
			false,
		},
		{
			"failed to unpack header",
			func() {
				msg.ClientMessage = nil
			},
			false,
		},
		{
			"invalid signer",
			func() {
				msg.Signer = ""
			},
			false,
		},
		{
			"valid - solomachine header",
			func() {
				soloMachine := ibctesting.NewSolomachine(s.T(), s.chainA.Codec, "solomachine", "", 2)
				msg, err = types.NewMsgUpdateClient(soloMachine.ClientID, soloMachine.CreateHeader(soloMachine.Diversifier), s.chainA.SenderAccount.GetAddress().String())

				s.Require().NoError(err)
			},
			true,
		},
		{
			"invalid solomachine header",
			func() {
				msg, err = types.NewMsgUpdateClient("solomachine", &solomachine.Header{}, s.chainA.SenderAccount.GetAddress().String())
				s.Require().NoError(err)
			},
			false,
		},
	}

	for _, tc := range cases {
		tc.malleate()
		err = msg.ValidateBasic()
		if tc.expPass {
			s.Require().NoError(err, tc.name)
		} else {
			s.Require().Error(err, tc.name)
		}
	}
}

func (s *TypesTestSuite) TestMarshalMsgUpgradeClient() {
	var (
		msg *types.MsgUpgradeClient
		err error
	)

	testCases := []struct {
		name     string
		malleate func()
	}{
		{
			"client upgrades to new tendermint client",
			func() {
				tendermintClient := ibctm.NewClientState(s.chainA.ChainID, ibctesting.DefaultTrustLevel, ibctesting.TrustingPeriod, ibctesting.UnbondingPeriod, ibctesting.MaxClockDrift, clientHeight, commitmenttypes.GetSDKSpecs(), ibctesting.UpgradePath)
				tendermintConsState := &ibctm.ConsensusState{NextValidatorsHash: []byte("nextValsHash")}
				msg, err = types.NewMsgUpgradeClient("clientid", tendermintClient, tendermintConsState, []byte("proofUpgradeClient"), []byte("proofUpgradeConsState"), s.chainA.SenderAccount.GetAddress().String())
				s.Require().NoError(err)
			},
		},
		{
			"client upgrades to new solomachine client",
			func() {
				soloMachine := ibctesting.NewSolomachine(s.T(), s.chainA.Codec, "solomachine", "", 1)
				msg, err = types.NewMsgUpgradeClient("clientid", soloMachine.ClientState(), soloMachine.ConsensusState(), []byte("proofUpgradeClient"), []byte("proofUpgradeConsState"), s.chainA.SenderAccount.GetAddress().String())
				s.Require().NoError(err)
			},
		},
	}

	for _, tc := range testCases {
		tc := tc

		s.Run(tc.name, func() {
			s.SetupTest()

			tc.malleate()

			cdc := s.chainA.App.AppCodec()

			// marshal message
			bz, err := cdc.MarshalJSON(msg)
			s.Require().NoError(err)

			// unmarshal message
			newMsg := &types.MsgUpgradeClient{}
			err = cdc.UnmarshalJSON(bz, newMsg)
			s.Require().NoError(err)
		})
	}
}

func (s *TypesTestSuite) TestMsgUpgradeClient_ValidateBasic() {
	cases := []struct {
		name     string
		malleate func(*types.MsgUpgradeClient)
		expPass  bool
	}{
		{
			name:     "success",
			malleate: func(msg *types.MsgUpgradeClient) {},
			expPass:  true,
		},
		{
			name: "client id empty",
			malleate: func(msg *types.MsgUpgradeClient) {
				msg.ClientId = ""
			},
			expPass: false,
		},
		{
			name: "invalid client id",
			malleate: func(msg *types.MsgUpgradeClient) {
				msg.ClientId = "invalid~chain/id"
			},
			expPass: false,
		},
		{
			name: "unpacking clientstate fails",
			malleate: func(msg *types.MsgUpgradeClient) {
				msg.ClientState = nil
			},
			expPass: false,
		},
		{
			name: "unpacking consensus state fails",
			malleate: func(msg *types.MsgUpgradeClient) {
				msg.ConsensusState = nil
			},
			expPass: false,
		},
		{
			name: "client and consensus type does not match",
			malleate: func(msg *types.MsgUpgradeClient) {
				soloMachine := ibctesting.NewSolomachine(s.T(), s.chainA.Codec, "solomachine", "", 2)
				soloConsensus, err := types.PackConsensusState(soloMachine.ConsensusState())
				s.Require().NoError(err)
				msg.ConsensusState = soloConsensus
			},
			expPass: false,
		},
		{
			name: "empty client proof",
			malleate: func(msg *types.MsgUpgradeClient) {
				msg.ProofUpgradeClient = nil
			},
			expPass: false,
		},
		{
			name: "empty consensus state proof",
			malleate: func(msg *types.MsgUpgradeClient) {
				msg.ProofUpgradeConsensusState = nil
			},
			expPass: false,
		},
		{
			name: "empty signer",
			malleate: func(msg *types.MsgUpgradeClient) {
				msg.Signer = "  "
			},
			expPass: false,
		},
	}

	for _, tc := range cases {
		tc := tc

		clientState := ibctm.NewClientState(s.chainA.ChainID, ibctesting.DefaultTrustLevel, ibctesting.TrustingPeriod, ibctesting.UnbondingPeriod, ibctesting.MaxClockDrift, clientHeight, commitmenttypes.GetSDKSpecs(), ibctesting.UpgradePath)
		consState := &ibctm.ConsensusState{NextValidatorsHash: []byte("nextValsHash")}
		msg, err := types.NewMsgUpgradeClient("testclientid", clientState, consState, []byte("proofUpgradeClient"), []byte("proofUpgradeConsState"), s.chainA.SenderAccount.GetAddress().String())
		s.Require().NoError(err)

		tc.malleate(msg)
		err = msg.ValidateBasic()
		if tc.expPass {
			s.Require().NoError(err, "valid case %s failed", tc.name)
		} else {
			s.Require().Error(err, "invalid case %s passed", tc.name)
		}
	}
}

// tests that different misbehaviours within MsgSubmitMisbehaviour can be marshaled
// and unmarshaled.
func (s *TypesTestSuite) TestMarshalMsgSubmitMisbehaviour() {
	var (
		msg *types.MsgSubmitMisbehaviour
		err error
	)

	testCases := []struct {
		name     string
		malleate func()
	}{
		{
			"solo machine client", func() {
				soloMachine := ibctesting.NewSolomachine(s.T(), s.chainA.Codec, "solomachine", "", 2)
				msg, err = types.NewMsgSubmitMisbehaviour(soloMachine.ClientID, soloMachine.CreateMisbehaviour(), s.chainA.SenderAccount.GetAddress().String())
				s.Require().NoError(err)
			},
		},
		{
			"tendermint client", func() {
				height := types.NewHeight(0, uint64(s.chainA.CurrentHeader.Height))
				heightMinus1 := types.NewHeight(0, uint64(s.chainA.CurrentHeader.Height)-1)
				header1 := s.chainA.CreateTMClientHeader(s.chainA.ChainID, int64(height.RevisionHeight), heightMinus1, s.chainA.CurrentHeader.Time, s.chainA.Vals, s.chainA.Vals, s.chainA.Vals, s.chainA.Signers)
				header2 := s.chainA.CreateTMClientHeader(s.chainA.ChainID, int64(height.RevisionHeight), heightMinus1, s.chainA.CurrentHeader.Time.Add(time.Minute), s.chainA.Vals, s.chainA.Vals, s.chainA.Vals, s.chainA.Signers)

				misbehaviour := ibctm.NewMisbehaviour("tendermint", header1, header2)
				msg, err = types.NewMsgSubmitMisbehaviour("tendermint", misbehaviour, s.chainA.SenderAccount.GetAddress().String())
				s.Require().NoError(err)
			},
		},
	}

	for _, tc := range testCases {
		tc := tc

		s.Run(tc.name, func() {
			s.SetupTest()

			tc.malleate()

			cdc := s.chainA.App.AppCodec()

			// marshal message
			bz, err := cdc.MarshalJSON(msg)
			s.Require().NoError(err)

			// unmarshal message
			newMsg := &types.MsgSubmitMisbehaviour{}
			err = cdc.UnmarshalJSON(bz, newMsg)
			s.Require().NoError(err)

			s.Require().True(proto.Equal(msg, newMsg))
		})
	}
}

func (s *TypesTestSuite) TestMsgSubmitMisbehaviour_ValidateBasic() {
	var (
		msg = &types.MsgSubmitMisbehaviour{}
		err error
	)

	cases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{
			"invalid client-id",
			func() {
				msg.ClientId = ""
			},
			false,
		},
		{
			"valid - tendermint misbehaviour",
			func() {
				height := types.NewHeight(0, uint64(s.chainA.CurrentHeader.Height))
				heightMinus1 := types.NewHeight(0, uint64(s.chainA.CurrentHeader.Height)-1)
				header1 := s.chainA.CreateTMClientHeader(s.chainA.ChainID, int64(height.RevisionHeight), heightMinus1, s.chainA.CurrentHeader.Time, s.chainA.Vals, s.chainA.Vals, s.chainA.Vals, s.chainA.Signers)
				header2 := s.chainA.CreateTMClientHeader(s.chainA.ChainID, int64(height.RevisionHeight), heightMinus1, s.chainA.CurrentHeader.Time.Add(time.Minute), s.chainA.Vals, s.chainA.Vals, s.chainA.Vals, s.chainA.Signers)

				misbehaviour := ibctm.NewMisbehaviour("tendermint", header1, header2)
				msg, err = types.NewMsgSubmitMisbehaviour("tendermint", misbehaviour, s.chainA.SenderAccount.GetAddress().String())
				s.Require().NoError(err)
			},
			true,
		},
		{
			"invalid tendermint misbehaviour",
			func() {
				msg, err = types.NewMsgSubmitMisbehaviour("tendermint", &ibctm.Misbehaviour{}, s.chainA.SenderAccount.GetAddress().String())
				s.Require().NoError(err)
			},
			false,
		},
		{
			"failed to unpack misbehaviourt",
			func() {
				msg.Misbehaviour = nil
			},
			false,
		},
		{
			"invalid signer",
			func() {
				msg.Signer = ""
			},
			false,
		},
		{
			"valid - solomachine misbehaviour",
			func() {
				soloMachine := ibctesting.NewSolomachine(s.T(), s.chainA.Codec, "solomachine", "", 2)
				msg, err = types.NewMsgSubmitMisbehaviour(soloMachine.ClientID, soloMachine.CreateMisbehaviour(), s.chainA.SenderAccount.GetAddress().String())
				s.Require().NoError(err)
			},
			true,
		},
		{
			"invalid solomachine misbehaviour",
			func() {
				msg, err = types.NewMsgSubmitMisbehaviour("solomachine", &solomachine.Misbehaviour{}, s.chainA.SenderAccount.GetAddress().String())
				s.Require().NoError(err)
			},
			false,
		},
		{
			"client-id mismatch",
			func() {
				soloMachineMisbehaviour := ibctesting.NewSolomachine(s.T(), s.chainA.Codec, "solomachine", "", 2).CreateMisbehaviour()
				msg, err = types.NewMsgSubmitMisbehaviour("external", soloMachineMisbehaviour, s.chainA.SenderAccount.GetAddress().String())
				s.Require().NoError(err)
			},
			false,
		},
	}

	for _, tc := range cases {
		tc.malleate()
		err = msg.ValidateBasic()
		if tc.expPass {
			s.Require().NoError(err, tc.name)
		} else {
			s.Require().Error(err, tc.name)
		}
	}
}

// TestMsgUpdateParamsValidateBasic tests ValidateBasic for MsgUpdateParams
func (s *TypesTestSuite) TestMsgUpdateParamsValidateBasic() {
	authority := s.chainA.App.GetIBCKeeper().GetAuthority()
	testCases := []struct {
		name    string
		msg     *types.MsgUpdateParams
		expPass bool
	}{
		{
			"success: valid authority and params",
			types.NewMsgUpdateParams(authority, types.DefaultParams()),
			true,
		},
		{
			"success: valid authority empty params",
			types.NewMsgUpdateParams(authority, types.Params{}),
			true,
		},
		{
			"failure: invalid authority address",
			types.NewMsgUpdateParams("invalid", types.DefaultParams()),
			false,
		},
		{
			"failure: invalid allowed client",
			types.NewMsgUpdateParams(authority, types.NewParams("")),
			false,
		},
	}

	for _, tc := range testCases {
		err := tc.msg.ValidateBasic()
		if tc.expPass {
			s.Require().NoError(err, "valid case %s failed", tc.name)
		} else {
			s.Require().Error(err, "invalid case %s passed", tc.name)
		}
	}
}

// TestMsgUpdateParamsGetSigners tests GetSigners for MsgUpdateParams
func TestMsgUpdateParamsGetSigners(t *testing.T) {
	testCases := []struct {
		name    string
		address sdk.AccAddress
		expPass bool
	}{
		{"success: valid address", sdk.AccAddress(ibctesting.TestAccAddress), true},
		{"failure: nil address", nil, false},
	}

	for _, tc := range testCases {
		msg := types.MsgUpdateParams{
			Authority: tc.address.String(),
			Params:    types.DefaultParams(),
		}
		if tc.expPass {
			require.Equal(t, []sdk.AccAddress{tc.address}, msg.GetSigners())
		} else {
			require.Panics(t, func() {
				msg.GetSigners()
			})
		}
	}
}
