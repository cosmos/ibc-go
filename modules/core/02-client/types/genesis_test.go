package types_test

import (
	"errors"
	"time"

	cmttypes "github.com/cometbft/cometbft/types"

	client "github.com/cosmos/ibc-go/v9/modules/core/02-client"
	"github.com/cosmos/ibc-go/v9/modules/core/02-client/types"
	commitmenttypes "github.com/cosmos/ibc-go/v9/modules/core/23-commitment/types"
	"github.com/cosmos/ibc-go/v9/modules/core/exported"
	solomachine "github.com/cosmos/ibc-go/v9/modules/light-clients/06-solomachine"
	ibctm "github.com/cosmos/ibc-go/v9/modules/light-clients/07-tendermint"
	ibctesting "github.com/cosmos/ibc-go/v9/testing"
)

const (
	tmClientID0     = "07-tendermint-0"
	tmClientID1     = "07-tendermint-1"
	invalidClientID = "myclient-0"
	clientID        = tmClientID0

	height = 10
)

var clientHeight = types.NewHeight(1, 10)

func (suite *TypesTestSuite) TestMarshalGenesisState() {
	cdc := suite.chainA.App.AppCodec()
	path := ibctesting.NewPath(suite.chainA, suite.chainB)
	path.Setup()
	err := path.EndpointA.UpdateClient()
	suite.Require().NoError(err)

	genesis, err := client.ExportGenesis(suite.chainA.GetContext(), suite.chainA.App.GetIBCKeeper().ClientKeeper)
	suite.Require().NoError(err)

	bz, err := cdc.MarshalJSON(&genesis)
	suite.Require().NoError(err)
	suite.Require().NotNil(bz)

	var gs types.GenesisState
	err = cdc.UnmarshalJSON(bz, &gs)
	suite.Require().NoError(err)
}

func (suite *TypesTestSuite) TestValidateGenesis() {
	privVal := cmttypes.NewMockPV()
	pubKey, err := privVal.GetPubKey()
	suite.Require().NoError(err)

	now := time.Now().UTC()

	val := cmttypes.NewValidator(pubKey, 10)
	valSet := cmttypes.NewValidatorSet([]*cmttypes.Validator{val})

	signers := make(map[string]cmttypes.PrivValidator)
	signers[val.Address.String()] = privVal

	heightMinus1 := types.NewHeight(1, height-1)
	header := suite.chainA.CreateTMClientHeader(suite.chainA.ChainID, int64(clientHeight.RevisionHeight), heightMinus1, now, valSet, valSet, valSet, signers)

	testCases := []struct {
		name     string
		genState types.GenesisState
		expError error
	}{
		{
			name:     "default",
			genState: types.DefaultGenesisState(),
			expError: nil,
		},
		{
			name: "valid custom genesis",
			genState: types.NewGenesisState(
				[]types.IdentifiedClientState{
					types.NewIdentifiedClientState(
						tmClientID0, ibctm.NewClientState(suite.chainA.ChainID, ibctesting.DefaultTrustLevel, ibctesting.TrustingPeriod, ibctesting.UnbondingPeriod, ibctesting.MaxClockDrift, clientHeight, commitmenttypes.GetSDKSpecs(), ibctesting.UpgradePath),
					),
				},
				[]types.ClientConsensusStates{
					types.NewClientConsensusStates(
						tmClientID0,
						[]types.ConsensusStateWithHeight{
							types.NewConsensusStateWithHeight(
								header.GetHeight().(types.Height),
								ibctm.NewConsensusState(
									header.GetTime(), commitmenttypes.NewMerkleRoot(header.Header.GetAppHash()), header.Header.NextValidatorsHash,
								),
							),
						},
					),
				},
				[]types.IdentifiedGenesisMetadata{
					types.NewIdentifiedGenesisMetadata(
						clientID,
						[]types.GenesisMetadata{
							types.NewGenesisMetadata([]byte("key1"), []byte("val1")),
							types.NewGenesisMetadata([]byte("key2"), []byte("val2")),
						},
					),
				},
				types.NewParams(exported.Tendermint),
				false,
				2,
			),
			expError: nil,
		},
		{
			name: "invalid client type",
			genState: types.NewGenesisState(
				[]types.IdentifiedClientState{
					types.NewIdentifiedClientState(
						ibctesting.DefaultSolomachineClientID, ibctm.NewClientState(suite.chainA.ChainID, ibctm.DefaultTrustLevel, ibctesting.TrustingPeriod, ibctesting.UnbondingPeriod, ibctesting.MaxClockDrift, clientHeight, commitmenttypes.GetSDKSpecs(), ibctesting.UpgradePath),
					),
					types.NewIdentifiedClientState(tmClientID0, solomachine.NewClientState(0, &solomachine.ConsensusState{PublicKey: suite.solomachine.ConsensusState().PublicKey, Diversifier: suite.solomachine.Diversifier, Timestamp: suite.solomachine.Time})),
				},
				nil,
				nil,
				types.NewParams(exported.Tendermint),
				false,
				0,
			),
			expError: errors.New("client state type 07-tendermint does not equal client type in client identifier 06-solomachine"),
		},
		{
			name: "invalid clientid",
			genState: types.NewGenesisState(
				[]types.IdentifiedClientState{
					types.NewIdentifiedClientState(
						invalidClientID, ibctm.NewClientState(suite.chainA.ChainID, ibctm.DefaultTrustLevel, ibctesting.TrustingPeriod, ibctesting.UnbondingPeriod, ibctesting.MaxClockDrift, clientHeight, commitmenttypes.GetSDKSpecs(), ibctesting.UpgradePath),
					),
				},
				[]types.ClientConsensusStates{
					types.NewClientConsensusStates(
						invalidClientID,
						[]types.ConsensusStateWithHeight{
							types.NewConsensusStateWithHeight(
								header.GetHeight().(types.Height),
								ibctm.NewConsensusState(
									header.GetTime(), commitmenttypes.NewMerkleRoot(header.Header.GetAppHash()), header.Header.NextValidatorsHash,
								),
							),
						},
					),
				},
				nil,
				types.NewParams(exported.Tendermint),
				false,
				0,
			),
			expError: errors.New("client state type 07-tendermint does not equal client type in client identifier myclient"),
		},
		{
			name: "consensus state client id does not match client id in genesis clients",
			genState: types.NewGenesisState(
				[]types.IdentifiedClientState{
					types.NewIdentifiedClientState(
						tmClientID0, ibctm.NewClientState(suite.chainA.ChainID, ibctm.DefaultTrustLevel, ibctesting.TrustingPeriod, ibctesting.UnbondingPeriod, ibctesting.MaxClockDrift, clientHeight, commitmenttypes.GetSDKSpecs(), ibctesting.UpgradePath),
					),
				},
				[]types.ClientConsensusStates{
					types.NewClientConsensusStates(
						tmClientID1,
						[]types.ConsensusStateWithHeight{
							types.NewConsensusStateWithHeight(
								types.NewHeight(1, 1),
								ibctm.NewConsensusState(
									header.GetTime(), commitmenttypes.NewMerkleRoot(header.Header.GetAppHash()), header.Header.NextValidatorsHash,
								),
							),
						},
					),
				},
				nil,
				types.NewParams(exported.Tendermint),
				false,
				0,
			),
			expError: errors.New("consensus state in genesis has a client id 07-tendermint-1 that does not map to a genesis client"),
		},
		{
			name: "invalid consensus state height",
			genState: types.NewGenesisState(
				[]types.IdentifiedClientState{
					types.NewIdentifiedClientState(
						tmClientID0, ibctm.NewClientState(suite.chainA.ChainID, ibctm.DefaultTrustLevel, ibctesting.TrustingPeriod, ibctesting.UnbondingPeriod, ibctesting.MaxClockDrift, clientHeight, commitmenttypes.GetSDKSpecs(), ibctesting.UpgradePath),
					),
				},
				[]types.ClientConsensusStates{
					types.NewClientConsensusStates(
						tmClientID0,
						[]types.ConsensusStateWithHeight{
							types.NewConsensusStateWithHeight(
								types.ZeroHeight(),
								ibctm.NewConsensusState(
									header.GetTime(), commitmenttypes.NewMerkleRoot(header.Header.GetAppHash()), header.Header.NextValidatorsHash,
								),
							),
						},
					),
				},
				nil,
				types.NewParams(exported.Tendermint),
				false,
				0,
			),
			expError: errors.New("consensus state height cannot be zero"),
		},
		{
			name: "invalid consensus state",
			genState: types.NewGenesisState(
				[]types.IdentifiedClientState{
					types.NewIdentifiedClientState(
						tmClientID0, ibctm.NewClientState(suite.chainA.ChainID, ibctm.DefaultTrustLevel, ibctesting.TrustingPeriod, ibctesting.UnbondingPeriod, ibctesting.MaxClockDrift, clientHeight, commitmenttypes.GetSDKSpecs(), ibctesting.UpgradePath),
					),
				},
				[]types.ClientConsensusStates{
					types.NewClientConsensusStates(
						tmClientID0,
						[]types.ConsensusStateWithHeight{
							types.NewConsensusStateWithHeight(
								types.NewHeight(1, 1),
								ibctm.NewConsensusState(
									time.Time{}, commitmenttypes.NewMerkleRoot(header.Header.GetAppHash()), header.Header.NextValidatorsHash,
								),
							),
						},
					),
				},
				nil,
				types.NewParams(exported.Tendermint),
				false,
				0,
			),
			expError: errors.New("invalid client consensus state timestamp"),
		},
		{
			name: "client in genesis clients is disallowed by params",
			genState: types.NewGenesisState(
				[]types.IdentifiedClientState{
					types.NewIdentifiedClientState(
						tmClientID0, ibctm.NewClientState(suite.chainA.ChainID, ibctesting.DefaultTrustLevel, ibctesting.TrustingPeriod, ibctesting.UnbondingPeriod, ibctesting.MaxClockDrift, clientHeight, commitmenttypes.GetSDKSpecs(), ibctesting.UpgradePath),
					),
				},
				[]types.ClientConsensusStates{
					types.NewClientConsensusStates(
						tmClientID0,
						[]types.ConsensusStateWithHeight{
							types.NewConsensusStateWithHeight(
								header.GetHeight().(types.Height),
								ibctm.NewConsensusState(
									header.GetTime(), commitmenttypes.NewMerkleRoot(header.Header.GetAppHash()), header.Header.NextValidatorsHash,
								),
							),
						},
					),
				},
				nil,
				types.NewParams(exported.Solomachine),
				false,
				0,
			),
			expError: errors.New("client type 07-tendermint not allowed by genesis params"),
		},
		{
			name: "metadata client-id does not match a genesis client",
			genState: types.NewGenesisState(
				[]types.IdentifiedClientState{
					types.NewIdentifiedClientState(
						clientID, ibctm.NewClientState(suite.chainA.ChainID, ibctesting.DefaultTrustLevel, ibctesting.TrustingPeriod, ibctesting.UnbondingPeriod, ibctesting.MaxClockDrift, clientHeight, commitmenttypes.GetSDKSpecs(), ibctesting.UpgradePath),
					),
				},
				[]types.ClientConsensusStates{
					types.NewClientConsensusStates(
						clientID,
						[]types.ConsensusStateWithHeight{
							types.NewConsensusStateWithHeight(
								header.GetHeight().(types.Height),
								ibctm.NewConsensusState(
									header.GetTime(), commitmenttypes.NewMerkleRoot(header.Header.GetAppHash()), header.Header.NextValidatorsHash,
								),
							),
						},
					),
				},
				[]types.IdentifiedGenesisMetadata{
					types.NewIdentifiedGenesisMetadata(
						"wrongclientid",
						[]types.GenesisMetadata{
							types.NewGenesisMetadata([]byte("key1"), []byte("val1")),
							types.NewGenesisMetadata([]byte("key2"), []byte("val2")),
						},
					),
				},
				types.NewParams(exported.Tendermint),
				false,
				0,
			),
			expError: errors.New("metadata in genesis has a client id wrongclientid that does not map to a genesis client"),
		},
		{
			name: "invalid metadata",
			genState: types.NewGenesisState(
				[]types.IdentifiedClientState{
					types.NewIdentifiedClientState(
						clientID, ibctm.NewClientState(suite.chainA.ChainID, ibctm.DefaultTrustLevel, ibctesting.TrustingPeriod, ibctesting.UnbondingPeriod, ibctesting.MaxClockDrift, clientHeight, commitmenttypes.GetSDKSpecs(), ibctesting.UpgradePath),
					),
				},
				[]types.ClientConsensusStates{
					types.NewClientConsensusStates(
						clientID,
						[]types.ConsensusStateWithHeight{
							types.NewConsensusStateWithHeight(
								header.GetHeight().(types.Height),
								ibctm.NewConsensusState(
									header.GetTime(), commitmenttypes.NewMerkleRoot(header.Header.GetAppHash()), header.Header.NextValidatorsHash,
								),
							),
						},
					),
				},
				[]types.IdentifiedGenesisMetadata{
					types.NewIdentifiedGenesisMetadata(
						clientID,
						[]types.GenesisMetadata{
							types.NewGenesisMetadata([]byte(""), []byte("val1")),
							types.NewGenesisMetadata([]byte("key2"), []byte("val2")),
						},
					),
				},
				types.NewParams(exported.Tendermint),
				false,
				0,
			),
			expError: errors.New("invalid client metadata"),
		},
		{
			name: "invalid params",
			genState: types.NewGenesisState(
				[]types.IdentifiedClientState{
					types.NewIdentifiedClientState(
						tmClientID0, ibctm.NewClientState(suite.chainA.ChainID, ibctesting.DefaultTrustLevel, ibctesting.TrustingPeriod, ibctesting.UnbondingPeriod, ibctesting.MaxClockDrift, clientHeight, commitmenttypes.GetSDKSpecs(), ibctesting.UpgradePath),
					),
				},
				[]types.ClientConsensusStates{
					types.NewClientConsensusStates(
						tmClientID0,
						[]types.ConsensusStateWithHeight{
							types.NewConsensusStateWithHeight(
								header.GetHeight().(types.Height),
								ibctm.NewConsensusState(
									header.GetTime(), commitmenttypes.NewMerkleRoot(header.Header.GetAppHash()), header.Header.NextValidatorsHash,
								),
							),
						},
					),
				},
				nil,
				types.NewParams(" "),
				false,
				0,
			),
			expError: errors.New("client type 0 cannot be blank"),
		},
		{
			name: "invalid param",
			genState: types.NewGenesisState(
				[]types.IdentifiedClientState{
					types.NewIdentifiedClientState(
						tmClientID0, ibctm.NewClientState(suite.chainA.ChainID, ibctesting.DefaultTrustLevel, ibctesting.TrustingPeriod, ibctesting.UnbondingPeriod, ibctesting.MaxClockDrift, clientHeight, commitmenttypes.GetSDKSpecs(), ibctesting.UpgradePath),
					),
				},
				[]types.ClientConsensusStates{
					types.NewClientConsensusStates(
						tmClientID0,
						[]types.ConsensusStateWithHeight{
							types.NewConsensusStateWithHeight(
								header.GetHeight().(types.Height),
								ibctm.NewConsensusState(
									header.GetTime(), commitmenttypes.NewMerkleRoot(header.Header.GetAppHash()), header.Header.NextValidatorsHash,
								),
							),
						},
					),
				},
				nil,
				types.NewParams(" "),
				false,
				0,
			),
			expError: errors.New("client type 0 cannot be blank"),
		},
		{
			name: "next sequence too small",
			genState: types.NewGenesisState(
				[]types.IdentifiedClientState{
					types.NewIdentifiedClientState(
						tmClientID0, ibctm.NewClientState(suite.chainA.ChainID, ibctesting.DefaultTrustLevel, ibctesting.TrustingPeriod, ibctesting.UnbondingPeriod, ibctesting.MaxClockDrift, clientHeight, commitmenttypes.GetSDKSpecs(), ibctesting.UpgradePath),
					),
					types.NewIdentifiedClientState(
						tmClientID1, ibctm.NewClientState(suite.chainA.ChainID, ibctesting.DefaultTrustLevel, ibctesting.TrustingPeriod, ibctesting.UnbondingPeriod, ibctesting.MaxClockDrift, clientHeight, commitmenttypes.GetSDKSpecs(), ibctesting.UpgradePath),
					),
				},
				[]types.ClientConsensusStates{
					types.NewClientConsensusStates(
						tmClientID0,
						[]types.ConsensusStateWithHeight{
							types.NewConsensusStateWithHeight(
								header.GetHeight().(types.Height),
								ibctm.NewConsensusState(
									header.GetTime(), commitmenttypes.NewMerkleRoot(header.Header.GetAppHash()), header.Header.NextValidatorsHash,
								),
							),
						},
					),
				},
				nil,
				types.NewParams(exported.Tendermint),
				false,
				0,
			),
			expError: errors.New("next client identifier sequence 0 must be greater than the maximum sequence used in the provided client identifiers 1"),
		},
		{
			name: "failed to parse client identifier in client state loop",
			genState: types.NewGenesisState(
				[]types.IdentifiedClientState{
					types.NewIdentifiedClientState(
						"my-client", ibctm.NewClientState(suite.chainA.ChainID, ibctesting.DefaultTrustLevel, ibctesting.TrustingPeriod, ibctesting.UnbondingPeriod, ibctesting.MaxClockDrift, clientHeight, commitmenttypes.GetSDKSpecs(), ibctesting.UpgradePath),
					),
				},
				[]types.ClientConsensusStates{
					types.NewClientConsensusStates(
						tmClientID0,
						[]types.ConsensusStateWithHeight{
							types.NewConsensusStateWithHeight(
								header.GetHeight().(types.Height),
								ibctm.NewConsensusState(
									header.GetTime(), commitmenttypes.NewMerkleRoot(header.Header.GetAppHash()), header.Header.NextValidatorsHash,
								),
							),
						},
					),
				},
				nil,
				types.NewParams(exported.Tendermint),
				false,
				5,
			),
			expError: errors.New("invalid client identifier my-client is not in format"),
		},
		{
			name: "consensus state different than client state type",
			genState: types.NewGenesisState(
				[]types.IdentifiedClientState{},
				[]types.ClientConsensusStates{
					types.NewClientConsensusStates(
						tmClientID0,
						[]types.ConsensusStateWithHeight{
							types.NewConsensusStateWithHeight(
								header.GetHeight().(types.Height),
								ibctm.NewConsensusState(
									header.GetTime(), commitmenttypes.NewMerkleRoot(header.Header.GetAppHash()), header.Header.NextValidatorsHash,
								),
							),
						},
					),
				},
				nil,
				types.NewParams(exported.Tendermint),
				false,
				5,
			),
			expError: errors.New("consensus state in genesis has a client id 07-tendermint-0 that does not map to a genesis client"),
		},
	}

	for _, tc := range testCases {
		tc := tc
		err := tc.genState.Validate()
		if tc.expError == nil {
			suite.Require().NoError(err, tc.name)
		} else {
			suite.Require().ErrorContains(err, tc.expError.Error())
		}
	}
}
