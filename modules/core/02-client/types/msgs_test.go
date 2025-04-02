package types_test

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/require"
	testifysuite "github.com/stretchr/testify/suite"

	errorsmod "cosmossdk.io/errors"
	upgradetypes "cosmossdk.io/x/upgrade/types"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"

	ibc "github.com/cosmos/ibc-go/v10/modules/core"
	"github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	commitmenttypes "github.com/cosmos/ibc-go/v10/modules/core/23-commitment/types"
	host "github.com/cosmos/ibc-go/v10/modules/core/24-host"
	ibcerrors "github.com/cosmos/ibc-go/v10/modules/core/errors"
	"github.com/cosmos/ibc-go/v10/modules/core/exported"
	solomachine "github.com/cosmos/ibc-go/v10/modules/light-clients/06-solomachine"
	ibctm "github.com/cosmos/ibc-go/v10/modules/light-clients/07-tendermint"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
)

type TypesTestSuite struct {
	testifysuite.Suite

	coordinator *ibctesting.Coordinator

	chainA      *ibctesting.TestChain
	chainB      *ibctesting.TestChain
	solomachine *ibctesting.Solomachine
}

func (suite *TypesTestSuite) SetupTest() {
	suite.coordinator = ibctesting.NewCoordinator(suite.T(), 2)
	suite.chainA = suite.coordinator.GetChain(ibctesting.GetChainID(1))
	suite.chainB = suite.coordinator.GetChain(ibctesting.GetChainID(2))
	suite.solomachine = ibctesting.NewSolomachine(suite.T(), suite.chainA.Codec, "solomachinesingle", "testing", 1)
}

func TestTypesTestSuite(t *testing.T) {
	testifysuite.Run(t, new(TypesTestSuite))
}

// tests that different clients within MsgCreateClient can be marshaled
// and unmarshaled.
func (suite *TypesTestSuite) TestMarshalMsgCreateClient() {
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
				soloMachine := ibctesting.NewSolomachine(suite.T(), suite.chainA.Codec, "solomachine", "", 2)
				msg, err = types.NewMsgCreateClient(soloMachine.ClientState(), soloMachine.ConsensusState(), suite.chainA.SenderAccount.GetAddress().String())
				suite.Require().NoError(err)
			},
		},
		{
			"tendermint client", func() {
				tendermintClient := ibctm.NewClientState(suite.chainA.ChainID, ibctesting.DefaultTrustLevel, ibctesting.TrustingPeriod, ibctesting.UnbondingPeriod, ibctesting.MaxClockDrift, clientHeight, commitmenttypes.GetSDKSpecs(), ibctesting.UpgradePath)
				msg, err = types.NewMsgCreateClient(tendermintClient, suite.chainA.CurrentTMClientHeader().ConsensusState(), suite.chainA.SenderAccount.GetAddress().String())
				suite.Require().NoError(err)
			},
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupTest()

			tc.malleate()

			cdc := suite.chainA.App.AppCodec()

			// marshal message
			bz, err := cdc.MarshalJSON(msg)
			suite.Require().NoError(err)

			// unmarshal message
			newMsg := &types.MsgCreateClient{}
			err = cdc.UnmarshalJSON(bz, newMsg)
			suite.Require().NoError(err)

			suite.Require().True(proto.Equal(msg, newMsg))
		})
	}
}

func (suite *TypesTestSuite) TestMsgCreateClient_ValidateBasic() {
	var (
		msg = &types.MsgCreateClient{}
		err error
	)

	cases := []struct {
		name     string
		malleate func()
		expErr   error
	}{
		{
			"valid - tendermint client",
			func() {
				tendermintClient := ibctm.NewClientState(suite.chainA.ChainID, ibctesting.DefaultTrustLevel, ibctesting.TrustingPeriod, ibctesting.UnbondingPeriod, ibctesting.MaxClockDrift, clientHeight, commitmenttypes.GetSDKSpecs(), ibctesting.UpgradePath)
				msg, err = types.NewMsgCreateClient(tendermintClient, suite.chainA.CurrentTMClientHeader().ConsensusState(), suite.chainA.SenderAccount.GetAddress().String())
				suite.Require().NoError(err)
			},
			nil,
		},
		{
			"invalid tendermint client",
			func() {
				msg, err = types.NewMsgCreateClient(&ibctm.ClientState{}, suite.chainA.CurrentTMClientHeader().ConsensusState(), suite.chainA.SenderAccount.GetAddress().String())
				suite.Require().NoError(err)
			},
			errorsmod.Wrap(ibctm.ErrInvalidChainID, "chain id cannot be empty string"),
		},
		{
			"failed to unpack client",
			func() {
				msg.ClientState = nil
			},
			errorsmod.Wrap(ibcerrors.ErrUnpackAny, "protobuf Any message cannot be nil"),
		},
		{
			"failed to unpack consensus state",
			func() {
				tendermintClient := ibctm.NewClientState(suite.chainA.ChainID, ibctesting.DefaultTrustLevel, ibctesting.TrustingPeriod, ibctesting.UnbondingPeriod, ibctesting.MaxClockDrift, clientHeight, commitmenttypes.GetSDKSpecs(), ibctesting.UpgradePath)
				msg, err = types.NewMsgCreateClient(tendermintClient, suite.chainA.CurrentTMClientHeader().ConsensusState(), suite.chainA.SenderAccount.GetAddress().String())
				suite.Require().NoError(err)
				msg.ConsensusState = nil
			},
			errorsmod.Wrap(ibcerrors.ErrUnpackAny, "protobuf Any message cannot be nil"),
		},
		{
			"invalid signer",
			func() {
				msg.Signer = ""
			},
			errorsmod.Wrapf(ibcerrors.ErrInvalidAddress, "string could not be parsed as address: empty address string is not allowed"),
		},
		{
			"valid - solomachine client",
			func() {
				soloMachine := ibctesting.NewSolomachine(suite.T(), suite.chainA.Codec, "solomachine", "", 2)
				msg, err = types.NewMsgCreateClient(soloMachine.ClientState(), soloMachine.ConsensusState(), suite.chainA.SenderAccount.GetAddress().String())
				suite.Require().NoError(err)
			},
			nil,
		},
		{
			"invalid solomachine client",
			func() {
				soloMachine := ibctesting.NewSolomachine(suite.T(), suite.chainA.Codec, "solomachine", "", 2)
				msg, err = types.NewMsgCreateClient(&solomachine.ClientState{}, soloMachine.ConsensusState(), suite.chainA.SenderAccount.GetAddress().String())
				suite.Require().NoError(err)
			},
			errorsmod.Wrap(types.ErrInvalidClient, "sequence cannot be 0"),
		},
		{
			"invalid solomachine consensus state",
			func() {
				soloMachine := ibctesting.NewSolomachine(suite.T(), suite.chainA.Codec, "solomachine", "", 2)
				msg, err = types.NewMsgCreateClient(soloMachine.ClientState(), &solomachine.ConsensusState{}, suite.chainA.SenderAccount.GetAddress().String())
				suite.Require().NoError(err)
			},
			errorsmod.Wrap(types.ErrInvalidConsensus, "timestamp cannot be 0"),
		},
		{
			"invalid - client state and consensus state client types do not match",
			func() {
				tendermintClient := ibctm.NewClientState(suite.chainA.ChainID, ibctesting.DefaultTrustLevel, ibctesting.TrustingPeriod, ibctesting.UnbondingPeriod, ibctesting.MaxClockDrift, clientHeight, commitmenttypes.GetSDKSpecs(), ibctesting.UpgradePath)
				soloMachine := ibctesting.NewSolomachine(suite.T(), suite.chainA.Codec, "solomachine", "", 2)
				msg, err = types.NewMsgCreateClient(tendermintClient, soloMachine.ConsensusState(), suite.chainA.SenderAccount.GetAddress().String())
				suite.Require().NoError(err)
			},
			errorsmod.Wrap(types.ErrInvalidClientType, "client type for client state and consensus state do not match"),
		},
	}

	for _, tc := range cases {
		tc.malleate()
		err = msg.ValidateBasic()
		if tc.expErr == nil {
			suite.Require().NoError(err, tc.name)
		} else {
			suite.Require().Error(err, tc.name)
			suite.Require().ErrorIs(err, tc.expErr)
		}
	}
}

// tests that different header within MsgUpdateClient can be marshaled
// and unmarshaled.
func (suite *TypesTestSuite) TestMarshalMsgUpdateClient() {
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
				soloMachine := ibctesting.NewSolomachine(suite.T(), suite.chainA.Codec, "solomachine", "", 2)
				msg, err = types.NewMsgUpdateClient(soloMachine.ClientID, soloMachine.CreateHeader(soloMachine.Diversifier), suite.chainA.SenderAccount.GetAddress().String())
				suite.Require().NoError(err)
			},
		},
		{
			"tendermint client", func() {
				msg, err = types.NewMsgUpdateClient("tendermint", suite.chainA.CurrentTMClientHeader(), suite.chainA.SenderAccount.GetAddress().String())
				suite.Require().NoError(err)
			},
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupTest()

			tc.malleate()

			cdc := suite.chainA.App.AppCodec()

			// marshal message
			bz, err := cdc.MarshalJSON(msg)
			suite.Require().NoError(err)

			// unmarshal message
			newMsg := &types.MsgUpdateClient{}
			err = cdc.UnmarshalJSON(bz, newMsg)
			suite.Require().NoError(err)

			suite.Require().True(proto.Equal(msg, newMsg))
		})
	}
}

func (suite *TypesTestSuite) TestMsgUpdateClient_ValidateBasic() {
	var (
		msg = &types.MsgUpdateClient{}
		err error
	)

	cases := []struct {
		name     string
		malleate func()
		expErr   error
	}{
		{
			"invalid client-id",
			func() {
				msg.ClientId = ""
			},
			errorsmod.Wrapf(ibcerrors.ErrInvalidAddress, "string could not be parsed as address:  empty address string is not allowed"),
		},
		{
			"valid - tendermint header",
			func() {
				msg, err = types.NewMsgUpdateClient("tendermint", suite.chainA.CurrentTMClientHeader(), suite.chainA.SenderAccount.GetAddress().String())
				suite.Require().NoError(err)
			},
			nil,
		},
		{
			"invalid tendermint header",
			func() {
				msg, err = types.NewMsgUpdateClient("tendermint", &ibctm.Header{}, suite.chainA.SenderAccount.GetAddress().String())
				suite.Require().NoError(err)
			},
			errorsmod.Wrap(types.ErrInvalidHeader, "tendermint signed header cannot be nil"),
		},
		{
			"failed to unpack header",
			func() {
				msg.ClientMessage = nil
			},
			errorsmod.Wrap(ibcerrors.ErrUnpackAny, "protobuf Any message cannot be nil"),
		},
		{
			"invalid signer",
			func() {
				msg.Signer = ""
			},
			errorsmod.Wrapf(ibcerrors.ErrInvalidAddress, "string could not be parsed as address:  empty address string is not allowed"),
		},
		{
			"valid - solomachine header",
			func() {
				soloMachine := ibctesting.NewSolomachine(suite.T(), suite.chainA.Codec, "solomachine", "", 2)
				msg, err = types.NewMsgUpdateClient(soloMachine.ClientID, soloMachine.CreateHeader(soloMachine.Diversifier), suite.chainA.SenderAccount.GetAddress().String())

				suite.Require().NoError(err)
			},
			nil,
		},
		{
			"invalid solomachine header",
			func() {
				msg, err = types.NewMsgUpdateClient("solomachine", &solomachine.Header{}, suite.chainA.SenderAccount.GetAddress().String())
				suite.Require().NoError(err)
			},
			errorsmod.Wrap(types.ErrInvalidHeader, "timestamp cannot be zero"),
		},
	}

	for _, tc := range cases {
		tc.malleate()
		err = msg.ValidateBasic()
		if tc.expErr == nil {
			suite.Require().NoError(err, tc.name)
		} else {
			suite.Require().Error(err, tc.name)
			suite.Require().ErrorIs(err, tc.expErr)
		}
	}
}

func (suite *TypesTestSuite) TestMarshalMsgUpgradeClient() {
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
				tendermintClient := ibctm.NewClientState(suite.chainA.ChainID, ibctesting.DefaultTrustLevel, ibctesting.TrustingPeriod, ibctesting.UnbondingPeriod, ibctesting.MaxClockDrift, clientHeight, commitmenttypes.GetSDKSpecs(), ibctesting.UpgradePath)
				tendermintConsState := &ibctm.ConsensusState{NextValidatorsHash: []byte("nextValsHash")}
				msg, err = types.NewMsgUpgradeClient("clientid", tendermintClient, tendermintConsState, []byte("proofUpgradeClient"), []byte("proofUpgradeConsState"), suite.chainA.SenderAccount.GetAddress().String())
				suite.Require().NoError(err)
			},
		},
		{
			"client upgrades to new solomachine client",
			func() {
				soloMachine := ibctesting.NewSolomachine(suite.T(), suite.chainA.Codec, "solomachine", "", 1)
				msg, err = types.NewMsgUpgradeClient("clientid", soloMachine.ClientState(), soloMachine.ConsensusState(), []byte("proofUpgradeClient"), []byte("proofUpgradeConsState"), suite.chainA.SenderAccount.GetAddress().String())
				suite.Require().NoError(err)
			},
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupTest()

			tc.malleate()

			cdc := suite.chainA.App.AppCodec()

			// marshal message
			bz, err := cdc.MarshalJSON(msg)
			suite.Require().NoError(err)

			// unmarshal message
			newMsg := &types.MsgUpgradeClient{}
			err = cdc.UnmarshalJSON(bz, newMsg)
			suite.Require().NoError(err)
		})
	}
}

func (suite *TypesTestSuite) TestMsgUpgradeClient_ValidateBasic() {
	cases := []struct {
		name     string
		malleate func(*types.MsgUpgradeClient)
		expErr   error
	}{
		{
			name:     "success",
			malleate: func(msg *types.MsgUpgradeClient) {},
			expErr:   nil,
		},
		{
			name: "client id empty",
			malleate: func(msg *types.MsgUpgradeClient) {
				msg.ClientId = ""
			},
			expErr: errorsmod.Wrap(host.ErrInvalidID, "protobuf Any message cannot be nil"),
		},
		{
			name: "invalid client id",
			malleate: func(msg *types.MsgUpgradeClient) {
				msg.ClientId = "invalid~chain/id"
			},
			expErr: errorsmod.Wrap(host.ErrInvalidID, "identifier invalid~chain/id cannot contain separator '/'"),
		},
		{
			name: "unpacking clientstate fails",
			malleate: func(msg *types.MsgUpgradeClient) {
				msg.ClientState = nil
			},
			expErr: errorsmod.Wrap(ibcerrors.ErrUnpackAny, "protobuf Any message cannot be nil"),
		},
		{
			name: "unpacking consensus state fails",
			malleate: func(msg *types.MsgUpgradeClient) {
				msg.ConsensusState = nil
			},
			expErr: errorsmod.Wrap(ibcerrors.ErrUnpackAny, "protobuf Any message cannot be nil"),
		},
		{
			name: "client and consensus type does not match",
			malleate: func(msg *types.MsgUpgradeClient) {
				soloMachine := ibctesting.NewSolomachine(suite.T(), suite.chainA.Codec, "solomachine", "", 2)
				soloConsensus, err := types.PackConsensusState(soloMachine.ConsensusState())
				suite.Require().NoError(err)
				msg.ConsensusState = soloConsensus
			},
			expErr: errorsmod.Wrap(types.ErrInvalidUpgradeClient, "consensus state's client-type does not match client. expected: 07-tendermint, got: 06-solomachine"),
		},
		{
			name: "empty client proof",
			malleate: func(msg *types.MsgUpgradeClient) {
				msg.ProofUpgradeClient = nil
			},
			expErr: errorsmod.Wrap(types.ErrInvalidUpgradeClient, "proof of upgrade client cannot be empty"),
		},
		{
			name: "empty consensus state proof",
			malleate: func(msg *types.MsgUpgradeClient) {
				msg.ProofUpgradeConsensusState = nil
			},
			expErr: errorsmod.Wrap(types.ErrInvalidUpgradeClient, "proof of upgrade consensus state cannot be empty"),
		},
		{
			name: "empty signer",
			malleate: func(msg *types.MsgUpgradeClient) {
				msg.Signer = "  "
			},
			expErr: errorsmod.Wrap(ibcerrors.ErrInvalidAddress, "string could not be parsed as address: empty address string is not allowed"),
		},
	}

	for _, tc := range cases {
		tc := tc

		clientState := ibctm.NewClientState(suite.chainA.ChainID, ibctesting.DefaultTrustLevel, ibctesting.TrustingPeriod, ibctesting.UnbondingPeriod, ibctesting.MaxClockDrift, clientHeight, commitmenttypes.GetSDKSpecs(), ibctesting.UpgradePath)
		consState := &ibctm.ConsensusState{NextValidatorsHash: []byte("nextValsHash")}
		msg, err := types.NewMsgUpgradeClient("testclientid", clientState, consState, []byte("proofUpgradeClient"), []byte("proofUpgradeConsState"), suite.chainA.SenderAccount.GetAddress().String())
		suite.Require().NoError(err)

		tc.malleate(msg)
		err = msg.ValidateBasic()
		if tc.expErr == nil {
			suite.Require().NoError(err, "valid case %s failed", tc.name)
		} else {
			suite.Require().Error(err, "invalid case %s passed", tc.name)
			suite.Require().ErrorIs(err, tc.expErr)
		}
	}
}

// tests that different misbehaviours within MsgSubmitMisbehaviour can be marshaled
// and unmarshaled.
func (suite *TypesTestSuite) TestMarshalMsgSubmitMisbehaviour() {
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
				soloMachine := ibctesting.NewSolomachine(suite.T(), suite.chainA.Codec, "solomachine", "", 2)
				msg, err = types.NewMsgSubmitMisbehaviour(soloMachine.ClientID, soloMachine.CreateMisbehaviour(), suite.chainA.SenderAccount.GetAddress().String())
				suite.Require().NoError(err)
			},
		},
		{
			"tendermint client", func() {
				height := types.NewHeight(0, uint64(suite.chainA.ProposedHeader.Height))
				heightMinus1 := types.NewHeight(0, uint64(suite.chainA.ProposedHeader.Height)-1)
				header1 := suite.chainA.CreateTMClientHeader(suite.chainA.ChainID, int64(height.RevisionHeight), heightMinus1, suite.chainA.ProposedHeader.Time, suite.chainA.Vals, suite.chainA.Vals, suite.chainA.Vals, suite.chainA.Signers)
				header2 := suite.chainA.CreateTMClientHeader(suite.chainA.ChainID, int64(height.RevisionHeight), heightMinus1, suite.chainA.ProposedHeader.Time.Add(time.Minute), suite.chainA.Vals, suite.chainA.Vals, suite.chainA.Vals, suite.chainA.Signers)

				misbehaviour := ibctm.NewMisbehaviour("tendermint", header1, header2)
				msg, err = types.NewMsgSubmitMisbehaviour("tendermint", misbehaviour, suite.chainA.SenderAccount.GetAddress().String())
				suite.Require().NoError(err)
			},
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupTest()

			tc.malleate()

			cdc := suite.chainA.App.AppCodec()

			// marshal message
			bz, err := cdc.MarshalJSON(msg)
			suite.Require().NoError(err)

			// unmarshal message
			newMsg := &types.MsgSubmitMisbehaviour{}
			err = cdc.UnmarshalJSON(bz, newMsg)
			suite.Require().NoError(err)

			suite.Require().True(proto.Equal(msg, newMsg))
		})
	}
}

func (suite *TypesTestSuite) TestMsgSubmitMisbehaviour_ValidateBasic() {
	var (
		msg = &types.MsgSubmitMisbehaviour{}
		err error
	)

	cases := []struct {
		name     string
		malleate func()
		expErr   error
	}{
		{
			"invalid client-id",
			func() {
				msg.ClientId = ""
			},
			errorsmod.Wrapf(ibcerrors.ErrInvalidAddress, "string could not be parsed as address: empty address string is not allowed"),
		},
		{
			"valid - tendermint misbehaviour",
			func() {
				height := types.NewHeight(0, uint64(suite.chainA.ProposedHeader.Height))
				heightMinus1 := types.NewHeight(0, uint64(suite.chainA.ProposedHeader.Height)-1)
				header1 := suite.chainA.CreateTMClientHeader(suite.chainA.ChainID, int64(height.RevisionHeight), heightMinus1, suite.chainA.ProposedHeader.Time, suite.chainA.Vals, suite.chainA.Vals, suite.chainA.Vals, suite.chainA.Signers)
				header2 := suite.chainA.CreateTMClientHeader(suite.chainA.ChainID, int64(height.RevisionHeight), heightMinus1, suite.chainA.ProposedHeader.Time.Add(time.Minute), suite.chainA.Vals, suite.chainA.Vals, suite.chainA.Vals, suite.chainA.Signers)

				misbehaviour := ibctm.NewMisbehaviour("tendermint", header1, header2)
				msg, err = types.NewMsgSubmitMisbehaviour("tendermint", misbehaviour, suite.chainA.SenderAccount.GetAddress().String())
				suite.Require().NoError(err)
			},
			nil,
		},
		{
			"invalid tendermint misbehaviour",
			func() {
				msg, err = types.NewMsgSubmitMisbehaviour("tendermint", &ibctm.Misbehaviour{}, suite.chainA.SenderAccount.GetAddress().String())
				suite.Require().NoError(err)
			},
			errorsmod.Wrap(ibctm.ErrInvalidHeader, "misbehaviour Header1 cannot be nil"),
		},
		{
			"failed to unpack misbehaviourt",
			func() {
				msg.Misbehaviour = nil
			},
			errorsmod.Wrap(ibcerrors.ErrUnpackAny, "protobuf Any message cannot be nil"),
		},
		{
			"invalid signer",
			func() {
				msg.Signer = ""
			},
			errorsmod.Wrapf(ibcerrors.ErrInvalidAddress, "string could not be parsed as address:  empty address string is not allowed"),
		},
		{
			"valid - solomachine misbehaviour",
			func() {
				soloMachine := ibctesting.NewSolomachine(suite.T(), suite.chainA.Codec, "solomachine", "", 2)
				msg, err = types.NewMsgSubmitMisbehaviour(soloMachine.ClientID, soloMachine.CreateMisbehaviour(), suite.chainA.SenderAccount.GetAddress().String())
				suite.Require().NoError(err)
			},
			nil,
		},
		{
			"invalid solomachine misbehaviour",
			func() {
				msg, err = types.NewMsgSubmitMisbehaviour("solomachine", &solomachine.Misbehaviour{}, suite.chainA.SenderAccount.GetAddress().String())
				suite.Require().NoError(err)
			},
			errorsmod.Wrapf(types.ErrInvalidMisbehaviour, "sequence cannot be 0"),
		},
		{
			"client-id too short",
			func() {
				soloMachineMisbehaviour := ibctesting.NewSolomachine(suite.T(), suite.chainA.Codec, "solomachine", "", 2).CreateMisbehaviour()
				msg, err = types.NewMsgSubmitMisbehaviour("ext", soloMachineMisbehaviour, suite.chainA.SenderAccount.GetAddress().String())
				suite.Require().NoError(err)
			},
			errorsmod.Wrapf(host.ErrInvalidID, "identifier external has invalid length: 3, must be between 4-64 characters"),
		},
	}

	for _, tc := range cases {
		tc.malleate()
		err = msg.ValidateBasic()
		if tc.expErr == nil {
			suite.Require().NoError(err, tc.name)
		} else {
			suite.Require().Error(err, tc.name)
			suite.Require().ErrorIs(err, tc.expErr)
		}
	}
}

// TestMsgRecoverClientValidateBasic tests ValidateBasic for MsgRecoverClient
func (suite *TypesTestSuite) TestMsgRecoverClientValidateBasic() {
	var msg *types.MsgRecoverClient

	testCases := []struct {
		name     string
		malleate func()
		expError error
	}{
		{
			"success: valid signer and client identifiers",
			func() {},
			nil,
		},
		{
			"failure: invalid signer address",
			func() {
				msg.Signer = "invalid"
			},
			ibcerrors.ErrInvalidAddress,
		},
		{
			"failure: invalid subject client ID",
			func() {
				msg.SubjectClientId = ""
			},
			host.ErrInvalidID,
		},
		{
			"failure: invalid substitute client ID",
			func() {
				msg.SubstituteClientId = ""
			},
			host.ErrInvalidID,
		},
		{
			"failure: subject and substribute client IDs are the same",
			func() {
				msg.SubstituteClientId = ibctesting.FirstClientID
			},
			types.ErrInvalidSubstitute,
		},
	}

	for _, tc := range testCases {
		msg = types.NewMsgRecoverClient(
			ibctesting.TestAccAddress,
			ibctesting.FirstClientID,
			ibctesting.SecondClientID,
		)

		tc.malleate()

		err := msg.ValidateBasic()
		if tc.expError == nil {
			suite.Require().NoError(err, "valid case %s failed", tc.name)
		} else {
			suite.Require().ErrorIs(err, tc.expError, "invalid case %s passed", tc.name)
		}
	}
}

// TestMsgRecoverClientGetSigners tests GetSigners for MsgRecoverClient
func TestMsgRecoverClientGetSigners(t *testing.T) {
	testCases := []struct {
		name     string
		address  sdk.AccAddress
		expError error
	}{
		{"success: valid address", sdk.AccAddress(ibctesting.TestAccAddress), nil},
		{"failure: nil address", nil, fmt.Errorf("empty address string is not allowed")},
	}

	for _, tc := range testCases {
		// Leave subject client ID and substitute client ID as empty strings
		msg := types.MsgRecoverClient{
			Signer: tc.address.String(),
		}
		encodingCfg := moduletestutil.MakeTestEncodingConfig(ibc.AppModuleBasic{})
		signers, _, err := encodingCfg.Codec.GetMsgV1Signers(&msg)
		if tc.expError == nil {
			require.NoError(t, err)
			require.Equal(t, tc.address.Bytes(), signers[0])
		} else {
			require.Error(t, err)
			require.Equal(t, err.Error(), tc.expError.Error())
		}
	}
}

// TestMsgIBCSoftwareUpgrade_NewMsgIBCSoftwareUpgrade tests NewMsgIBCSoftwareUpgrade
func (suite *TypesTestSuite) TestMsgIBCSoftwareUpgrade_NewMsgIBCSoftwareUpgrade() {
	testCases := []struct {
		name                string
		upgradedClientState exported.ClientState
		expErr              error
	}{
		{
			"success",
			ibctm.NewClientState(suite.chainA.ChainID, ibctesting.DefaultTrustLevel, ibctesting.TrustingPeriod, ibctesting.UnbondingPeriod, ibctesting.MaxClockDrift, clientHeight, commitmenttypes.GetSDKSpecs(), ibctesting.UpgradePath),
			nil,
		},
		{
			"fail: failed to pack ClientState",
			nil,
			errorsmod.Wrap(ibcerrors.ErrPackAny, "cannot proto marshal <nil>"),
		},
	}

	for _, tc := range testCases {
		plan := upgradetypes.Plan{
			Name:   "upgrade IBC clients",
			Height: 1000,
		}
		msg, err := types.NewMsgIBCSoftwareUpgrade(
			ibctesting.TestAccAddress,
			plan,
			tc.upgradedClientState,
		)

		if tc.expErr == nil {
			suite.Require().NoError(err)
			suite.Assert().Equal(ibctesting.TestAccAddress, msg.Signer)
			suite.Assert().Equal(plan, msg.Plan)
			unpackedClientState, err := types.UnpackClientState(msg.UpgradedClientState)
			suite.Require().NoError(err)
			suite.Assert().Equal(tc.upgradedClientState, unpackedClientState)
		} else {
			suite.Require().True(errors.Is(err, ibcerrors.ErrPackAny))
			suite.Require().ErrorIs(err, tc.expErr)
		}
	}
}

// TestMsgIBCSoftwareUpgrade_GetSigners tests GetSigners for MsgIBCSoftwareUpgrade
func TestMsgIBCSoftwareUpgrade_GetSigners(t *testing.T) {
	testCases := []struct {
		name    string
		address sdk.AccAddress
		expErr  error
	}{
		{
			"success: valid address",
			sdk.AccAddress(ibctesting.TestAccAddress),
			nil,
		},
		{
			"failure: nil address",
			nil,
			fmt.Errorf("empty address string is not allowed"),
		},
	}

	for _, tc := range testCases {
		clientState := ibctm.NewClientState("testchain1-1", ibctesting.DefaultTrustLevel, ibctesting.TrustingPeriod, ibctesting.UnbondingPeriod, ibctesting.MaxClockDrift, clientHeight, commitmenttypes.GetSDKSpecs(), ibctesting.UpgradePath)
		plan := upgradetypes.Plan{
			Name:   "upgrade IBC clients",
			Height: 1000,
		}
		msg, err := types.NewMsgIBCSoftwareUpgrade(
			tc.address.String(),
			plan,
			clientState,
		)
		require.NoError(t, err)

		encodingCfg := moduletestutil.MakeTestEncodingConfig(ibc.AppModuleBasic{})
		signers, _, err := encodingCfg.Codec.GetMsgV1Signers(msg)
		if tc.expErr == nil {
			require.NoError(t, err)
			require.Equal(t, tc.address.Bytes(), signers[0])
		} else {
			require.Error(t, err)
			require.Equal(t, err.Error(), tc.expErr.Error())
		}
	}
}

// TestMsgIBCSoftwareUpgrade_ValidateBasic tests ValidateBasic for MsgIBCSoftwareUpgrade
func (suite *TypesTestSuite) TestMsgIBCSoftwareUpgrade_ValidateBasic() {
	var (
		signer    string
		plan      upgradetypes.Plan
		anyClient *codectypes.Any
	)
	testCases := []struct {
		name     string
		malleate func()
		expError error
	}{
		{
			"success",
			func() {},
			nil,
		},
		{
			"failure: invalid authority address",
			func() {
				signer = "invalid"
			},
			ibcerrors.ErrInvalidAddress,
		},
		{
			"failure: error unpacking client state",
			func() {
				anyClient = &codectypes.Any{}
			},
			ibcerrors.ErrUnpackAny,
		},
		{
			"failure: error validating upgrade plan, height is not greater than zero",
			func() {
				plan.Height = 0
			},
			sdkerrors.ErrInvalidRequest,
		},
	}

	for _, tc := range testCases {
		signer = ibctesting.TestAccAddress
		plan = upgradetypes.Plan{
			Name:   "upgrade IBC clients",
			Height: 1000,
		}
		upgradedClientState := ibctm.NewClientState(suite.chainA.ChainID, ibctesting.DefaultTrustLevel, ibctesting.TrustingPeriod, ibctesting.UnbondingPeriod, ibctesting.MaxClockDrift, clientHeight, commitmenttypes.GetSDKSpecs(), ibctesting.UpgradePath)
		var err error
		anyClient, err = types.PackClientState(upgradedClientState)
		suite.Require().NoError(err)

		tc.malleate()

		msg := types.MsgIBCSoftwareUpgrade{
			plan,
			anyClient,
			signer,
		}

		err = msg.ValidateBasic()

		if tc.expError == nil {
			suite.Require().NoError(err)
		}
		if tc.expError != nil {
			suite.Require().True(errors.Is(err, tc.expError))
		}
	}
}

// tests a MsgIBCSoftwareUpgrade can be marshaled and unmarshaled, and the
// client state can be unpacked
func (suite *TypesTestSuite) TestMarshalMsgIBCSoftwareUpgrade() {
	cdc := suite.chainA.App.AppCodec()

	// create proposal
	plan := upgradetypes.Plan{
		Name:   "upgrade ibc",
		Height: 1000,
	}

	msg, err := types.NewMsgIBCSoftwareUpgrade(ibctesting.TestAccAddress, plan, &ibctm.ClientState{})
	suite.Require().NoError(err)

	// marshal message
	bz, err := cdc.MarshalJSON(msg)
	suite.Require().NoError(err)

	// unmarshal proposal
	newMsg := &types.MsgIBCSoftwareUpgrade{}
	err = cdc.UnmarshalJSON(bz, newMsg)
	suite.Require().NoError(err)

	// unpack client state
	_, err = types.UnpackClientState(newMsg.UpgradedClientState)
	suite.Require().NoError(err)
}

// TestMsgUpdateParamsValidateBasic tests ValidateBasic for MsgUpdateParams
func (suite *TypesTestSuite) TestMsgUpdateParamsValidateBasic() {
	signer := suite.chainA.App.GetIBCKeeper().GetAuthority()
	testCases := []struct {
		name   string
		msg    *types.MsgUpdateParams
		expErr error
	}{
		{
			"success: valid signer and params",
			types.NewMsgUpdateParams(signer, types.DefaultParams()),
			nil,
		},
		{
			"success: valid signer empty params",
			types.NewMsgUpdateParams(signer, types.Params{}),
			nil,
		},
		{
			"failure: invalid signer address",
			types.NewMsgUpdateParams("invalid", types.DefaultParams()),
			errorsmod.Wrapf(ibcerrors.ErrInvalidAddress, "string could not be parsed as address: decoding bech32 failed: invalid bech32 string length 7"),
		},
		{
			"failure: invalid allowed client",
			types.NewMsgUpdateParams(signer, types.NewParams("")),
			errors.New("client type 0 cannot be blank"),
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			err := tc.msg.ValidateBasic()
			if tc.expErr == nil {
				suite.Require().NoError(err, "valid case %s failed", tc.name)
			} else {
				suite.Require().Error(err, "invalid case %s passed", tc.name)
				suite.Require().Equal(tc.expErr.Error(), err.Error())
			}
		})
	}
}

// TestMsgUpdateParamsGetSigners tests GetSigners for MsgUpdateParams
func TestMsgUpdateParamsGetSigners(t *testing.T) {
	testCases := []struct {
		name    string
		address sdk.AccAddress
		expErr  error
	}{
		{"success: valid address", sdk.AccAddress(ibctesting.TestAccAddress), nil},
		{"failure: nil address", nil, errors.New("empty address string is not allowed")},
	}

	for _, tc := range testCases {
		tc := tc

		msg := types.MsgUpdateParams{
			Signer: tc.address.String(),
			Params: types.DefaultParams(),
		}
		encodingCfg := moduletestutil.MakeTestEncodingConfig(ibc.AppModuleBasic{})
		signers, _, err := encodingCfg.Codec.GetMsgV1Signers(&msg)
		if tc.expErr == nil {
			require.NoError(t, err)
			require.Equal(t, tc.address.Bytes(), signers[0])
		} else {
			require.Error(t, err)
			require.Equal(t, err.Error(), tc.expErr.Error())
		}
	}
}
