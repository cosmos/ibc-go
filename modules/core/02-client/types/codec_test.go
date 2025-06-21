package types_test

import (
	"errors"

	errorsmod "cosmossdk.io/errors"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	commitmenttypes "github.com/cosmos/ibc-go/v10/modules/core/23-commitment/types"
	ibcerrors "github.com/cosmos/ibc-go/v10/modules/core/errors"
	"github.com/cosmos/ibc-go/v10/modules/core/exported"
	ibctm "github.com/cosmos/ibc-go/v10/modules/light-clients/07-tendermint"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
)

type caseAny struct {
	name   string
	any    *codectypes.Any
	expErr error
}

func (s *TypesTestSuite) TestPackClientState() {
	testCases := []struct {
		name        string
		clientState exported.ClientState
		expErr      error
	}{
		{
			"solo machine client",
			ibctesting.NewSolomachine(s.T(), s.chainA.Codec, "solomachine", "", 2).ClientState(),
			nil,
		},
		{
			"tendermint client",
			ibctm.NewClientState(s.chainA.ChainID, ibctesting.DefaultTrustLevel, ibctesting.TrustingPeriod, ibctesting.UnbondingPeriod, ibctesting.MaxClockDrift, clientHeight, commitmenttypes.GetSDKSpecs(), ibctesting.UpgradePath),
			nil,
		},
		{
			"nil",
			nil,
			errorsmod.Wrap(ibcerrors.ErrUnpackAny, "protobuf Any message cannot be nil"),
		},
	}

	testCasesAny := []caseAny{}

	for _, tc := range testCases {
		protoAny, err := types.PackClientState(tc.clientState)
		if tc.expErr == nil {
			s.Require().NoError(err, tc.name)
		} else {
			s.Require().Error(err, tc.name)
		}

		testCasesAny = append(testCasesAny, caseAny{tc.name, protoAny, tc.expErr})
	}

	for i, tc := range testCasesAny {
		cs, err := types.UnpackClientState(tc.any)
		if tc.expErr == nil {
			s.Require().NoError(err, tc.name)
			s.Require().Equal(testCases[i].clientState, cs, tc.name)
		} else {
			s.Require().Error(err, tc.name)
			s.Require().ErrorIs(err, tc.expErr)
		}
	}
}

func (s *TypesTestSuite) TestPackConsensusState() {
	testCases := []struct {
		name           string
		consensusState exported.ConsensusState
		expErr         error
	}{
		{
			"solo machine consensus",
			ibctesting.NewSolomachine(s.T(), s.chainA.Codec, "solomachine", "", 2).ConsensusState(),
			nil,
		},
		{
			"tendermint consensus",
			s.chainA.LatestCommittedHeader.ConsensusState(),
			nil,
		},
		{
			"nil",
			nil,
			errorsmod.Wrap(ibcerrors.ErrUnpackAny, "protobuf Any message cannot be nil"),
		},
	}

	testCasesAny := []caseAny{}

	for _, tc := range testCases {
		protoAny, err := types.PackConsensusState(tc.consensusState)
		if tc.expErr == nil {
			s.Require().NoError(err, tc.name)
		} else {
			s.Require().Error(err, tc.name)
		}
		testCasesAny = append(testCasesAny, caseAny{tc.name, protoAny, tc.expErr})
	}

	for i, tc := range testCasesAny {
		cs, err := types.UnpackConsensusState(tc.any)
		if tc.expErr == nil {
			s.Require().NoError(err, tc.name)
			s.Require().Equal(testCases[i].consensusState, cs, tc.name)
		} else {
			s.Require().Error(err, tc.name)
			s.Require().ErrorIs(err, tc.expErr)
		}
	}
}

func (s *TypesTestSuite) TestPackClientMessage() {
	testCases := []struct {
		name          string
		clientMessage exported.ClientMessage
		expErr        error
	}{
		{
			"solo machine header",
			ibctesting.NewSolomachine(s.T(), s.chainA.Codec, "solomachine", "", 2).CreateHeader("solomachine"),
			nil,
		},
		{
			"tendermint header",
			s.chainA.LatestCommittedHeader,
			nil,
		},
		{
			"nil",
			nil,
			errorsmod.Wrap(ibcerrors.ErrUnpackAny, "protobuf Any message cannot be nil"),
		},
	}

	testCasesAny := []caseAny{}

	for _, tc := range testCases {
		protoAny, err := types.PackClientMessage(tc.clientMessage)
		if tc.expErr == nil {
			s.Require().NoError(err, tc.name)
		} else {
			s.Require().Error(err, tc.name)
		}

		testCasesAny = append(testCasesAny, caseAny{tc.name, protoAny, tc.expErr})
	}

	for i, tc := range testCasesAny {
		cs, err := types.UnpackClientMessage(tc.any)
		if tc.expErr == nil {
			s.Require().NoError(err, tc.name)
			s.Require().Equal(testCases[i].clientMessage, cs, tc.name)
		} else {
			s.Require().Error(err, tc.name)
			s.Require().ErrorIs(err, tc.expErr)
		}
	}
}

func (s *TypesTestSuite) TestCodecTypeRegistration() {
	testCases := []struct {
		name    string
		typeURL string
		expErr  error
	}{
		{
			"success: MsgCreateClient",
			sdk.MsgTypeURL(&types.MsgCreateClient{}),
			nil,
		},
		{
			"success: MsgUpdateClient",
			sdk.MsgTypeURL(&types.MsgUpdateClient{}),
			nil,
		},
		{
			"success: MsgUpgradeClient",
			sdk.MsgTypeURL(&types.MsgUpgradeClient{}),
			nil,
		},
		{
			"success: MsgRecoverClient",
			sdk.MsgTypeURL(&types.MsgRecoverClient{}),
			nil,
		},
		{
			"success: MsgIBCSoftwareUpgrade",
			sdk.MsgTypeURL(&types.MsgIBCSoftwareUpgrade{}),
			nil,
		},
		{
			"success: MsgUpdateParams",
			sdk.MsgTypeURL(&types.MsgUpdateParams{}),
			nil,
		},
		{
			"success: ClientUpdateProposal",
			sdk.MsgTypeURL(&types.ClientUpdateProposal{}),
			nil,
		},
		{
			"success: UpgradeProposal",
			sdk.MsgTypeURL(&types.UpgradeProposal{}),
			nil,
		},
		{
			"type not registered on codec",
			"ibc.invalid.MsgTypeURL",
			errors.New("unable to resolve type URL ibc.invalid.MsgTypeURL"),
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			msg, err := s.chainA.GetSimApp().AppCodec().InterfaceRegistry().Resolve(tc.typeURL)

			if tc.expErr == nil {
				s.Require().NotNil(msg)
				s.Require().NoError(err)
			} else {
				s.Require().Nil(msg)
				s.Require().Error(err)
				s.Require().Equal(err.Error(), tc.expErr.Error())
			}
		})
	}
}
