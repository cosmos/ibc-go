package ibctesting

import (
	"bytes"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/stretchr/testify/require"
	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/crypto/tmhash"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"
	tmprotoversion "github.com/tendermint/tendermint/proto/tendermint/version"
	tmtypes "github.com/tendermint/tendermint/types"
	tmversion "github.com/tendermint/tendermint/version"

	clienttypes "github.com/cosmos/ibc-go/v3/modules/core/02-client/types"
	commitmenttypes "github.com/cosmos/ibc-go/v3/modules/core/23-commitment/types"
	"github.com/cosmos/ibc-go/v3/modules/core/exported"
	ibctmtypes "github.com/cosmos/ibc-go/v3/modules/light-clients/07-tendermint/types"
)

type TendermintConfig struct {
	TrustLevel                   ibctmtypes.Fraction
	TrustingPeriod               time.Duration
	UnbondingPeriod              time.Duration
	MaxClockDrift                time.Duration
	AllowUpdateAfterExpiry       bool
	AllowUpdateAfterMisbehaviour bool
}

func (tmcfg *TendermintConfig) GetClientType() string {
	return exported.Tendermint
}

var _ ClientConfig = &TendermintConfig{}

// TestChainTendermint is a testing struct that 'wraps' a TestChain with the last TM Header,
// the current ABCI header.
type TestChainTendermint struct {
	TC *TestChain

	LastHeader    *ibctmtypes.Header // header for last block height committed
	CurrentHeader tmproto.Header     // header for current block height

}

var _ TestChainClientI = &TestChainTendermint{}

// NewChainTendermintClient initializes the consunsus spesisifc pare of the TestChain
func NewChainTendermintClient(tc *TestChain) *TestChainTendermint {

	// create current header and call begin block
	header := tmproto.Header{
		ChainID: tc.ChainID,
		Height:  1,
		Time:    tc.Coordinator.CurrentTime.UTC(),
	}

	// create an account to send transactions from
	chain := &TestChainTendermint{
		tc,
		nil,
		header,
	}

	return chain
}

func (chain *TestChainTendermint) GetSelfClientType() string {
	return exported.Tendermint
}

func (chain *TestChainTendermint) NewConfig() ClientConfig {
	return &TendermintConfig{
		TrustLevel:                   DefaultTrustLevel,
		TrustingPeriod:               TrustingPeriod,
		UnbondingPeriod:              UnbondingPeriod,
		MaxClockDrift:                MaxClockDrift,
		AllowUpdateAfterExpiry:       false,
		AllowUpdateAfterMisbehaviour: false,
	}
}

// GetContext returns the current context for the application.
func (chain *TestChainTendermint) GetContext() sdk.Context {
	return chain.TC.App.GetBaseApp().NewContext(false, chain.CurrentHeader)
}

// NextBlock sets the last header to the current header and increments the current header to be
// at the next block height. It does not update the time as that is handled by the Coordinator.
//
// CONTRACT: this function must only be called after app.Commit() occurs
func (chain *TestChainTendermint) NextBlock() {
	// set the last header to the current header
	// use nil trusted fields
	chain.LastHeader = chain.CurrentTMClientHeader()

	// increment the current header
	chain.CurrentHeader = tmproto.Header{
		ChainID: chain.TC.ChainID,
		Height:  chain.TC.App.LastBlockHeight() + 1,
		AppHash: chain.TC.App.LastCommitID().Hash,
		// NOTE: the time is increased by the coordinator to maintain time synchrony amongst
		// chains.
		Time:               chain.CurrentHeader.Time,
		ValidatorsHash:     chain.TC.Vals.Hash(),
		NextValidatorsHash: chain.TC.Vals.Hash(),
	}

	chain.BeginBlock()
}

// ConstructUpdateTMClientHeader will construct a valid 07-tendermint Header to update the
// light client on the source chain.
func (chain *TestChainTendermint) ConstructUpdateTMClientHeader(counterparty *TestChain, clientID string) (*ibctmtypes.Header, error) {
	// Relayer must query for LatestHeight on client to get TrustedHeight if the trusted height is not set
	trustedHeight := chain.TC.GetClientState(clientID).GetLatestHeight().(clienttypes.Height)
	return ConstructUpdateTMClientHeaderWithTrustedHeight(counterparty, clientID, trustedHeight)
}

// ConstructUpdateTMClientHeader will construct a valid 07-tendermint Header to update the
// light client on the source chain.
func ConstructUpdateTMClientHeaderWithTrustedHeight(counterparty *TestChain, clientID string, trustedHeight clienttypes.Height) (*ibctmtypes.Header, error) {
	header := counterparty.TestChainClient.GetLastHeader().(*ibctmtypes.Header)
	// Relayer must query for LatestHeight on client to get TrustedHeight if the trusted height is not set
	require.False(counterparty.T, trustedHeight.IsZero())

	var (
		tmTrustedVals *tmtypes.ValidatorSet
		ok            bool
	)
	// Once we get TrustedHeight from client, we must query the validators from the counterparty chain
	// If the LatestHeight == LastHeader.Height, then TrustedValidators are current validators
	// If LatestHeight < LastHeader.Height, we can query the historical validator set from HistoricalInfo
	if trustedHeight == header.GetHeight() {
		tmTrustedVals = counterparty.Vals
	} else {
		// NOTE: We need to get validators from counterparty at height: trustedHeight+1
		// since the last trusted validators for a header at height h
		// is the NextValidators at h+1 committed to in header h by
		// NextValidatorsHash
		tmTrustedVals, ok = counterparty.GetValsAtHeight(int64(trustedHeight.RevisionHeight + 1))
		if !ok {
			return nil, sdkerrors.Wrapf(ibctmtypes.ErrInvalidHeaderHeight, "could not retrieve trusted validators at trustedHeight: %d", trustedHeight)
		}
	}
	// inject trusted fields into last header
	// for now assume revision number is 0
	header.TrustedHeight = trustedHeight

	trustedVals, err := tmTrustedVals.ToProto()
	if err != nil {
		return nil, err
	}
	header.TrustedValidators = trustedVals

	return header, nil
}

// CurrentTMClientHeader creates a TM header using the current header parameters
// on the chain. The trusted fields in the header are set to nil.
func (chain *TestChainTendermint) CurrentTMClientHeader() *ibctmtypes.Header {
	return chain.CreateTMClientHeader(chain.TC.ChainID, chain.CurrentHeader.Height, clienttypes.Height{}, chain.CurrentHeader.Time, chain.TC.Vals, nil, chain.TC.Signers)
}

// CreateTMClientHeader creates a TM header to update the TM client. Args are passed in to allow
// caller flexibility to use params that differ from the chain.
func (chain *TestChainTendermint) CreateTMClientHeader(chainID string, blockHeight int64, trustedHeight clienttypes.Height, timestamp time.Time, tmValSet, tmTrustedVals *tmtypes.ValidatorSet, signers []tmtypes.PrivValidator) *ibctmtypes.Header {
	var (
		valSet      *tmproto.ValidatorSet
		trustedVals *tmproto.ValidatorSet
	)
	require.NotNil(chain.TC.T, tmValSet)

	vsetHash := tmValSet.Hash()

	tmHeader := tmtypes.Header{
		Version:            tmprotoversion.Consensus{Block: tmversion.BlockProtocol, App: 2},
		ChainID:            chainID,
		Height:             blockHeight,
		Time:               timestamp,
		LastBlockID:        MakeBlockID(make([]byte, tmhash.Size), 10_000, make([]byte, tmhash.Size)),
		LastCommitHash:     chain.TC.App.LastCommitID().Hash,
		DataHash:           tmhash.Sum([]byte("data_hash")),
		ValidatorsHash:     vsetHash,
		NextValidatorsHash: vsetHash,
		ConsensusHash:      tmhash.Sum([]byte("consensus_hash")),
		AppHash:            chain.CurrentHeader.AppHash,
		LastResultsHash:    tmhash.Sum([]byte("last_results_hash")),
		EvidenceHash:       tmhash.Sum([]byte("evidence_hash")),
		ProposerAddress:    tmValSet.Proposer.Address, //nolint:staticcheck
	}

	hhash := tmHeader.Hash()
	blockID := MakeBlockID(hhash, 3, tmhash.Sum([]byte("part_set")))
	voteSet := tmtypes.NewVoteSet(chainID, blockHeight, 1, tmproto.PrecommitType, tmValSet)

	commit, err := tmtypes.MakeCommit(blockID, blockHeight, 1, voteSet, signers, timestamp)
	require.NoError(chain.TC.T, err)

	signedHeader := &tmproto.SignedHeader{
		Header: tmHeader.ToProto(),
		Commit: commit.ToProto(),
	}

	if tmValSet != nil {
		valSet, err = tmValSet.ToProto()
		require.NoError(chain.TC.T, err)
	}

	if tmTrustedVals != nil {
		trustedVals, err = tmTrustedVals.ToProto()
		require.NoError(chain.TC.T, err)
	}

	// The trusted fields may be nil. They may be filled before relaying messages to a client.
	// The relayer is responsible for querying client and injecting appropriate trusted fields.
	return &ibctmtypes.Header{
		SignedHeader:      signedHeader,
		ValidatorSet:      valSet,
		TrustedHeight:     trustedHeight,
		TrustedValidators: trustedVals,
	}
}

// MakeBlockID copied unimported test functions from tmtypes to use them here
func MakeBlockID(hash []byte, partSetSize uint32, partSetHash []byte) tmtypes.BlockID {
	return tmtypes.BlockID{
		Hash: hash,
		PartSetHeader: tmtypes.PartSetHeader{
			Total: partSetSize,
			Hash:  partSetHash,
		},
	}
}

// CreateSortedSignerArray takes two PrivValidators, and the corresponding Validator structs
// (including voting power). It returns a signer array of PrivValidators that matches the
// sorting of ValidatorSet.
// The sorting is first by .VotingPower (descending), with secondary index of .Address (ascending).
func CreateSortedSignerArray(altPrivVal, suitePrivVal tmtypes.PrivValidator,
	altVal, suiteVal *tmtypes.Validator,
) []tmtypes.PrivValidator {
	switch {
	case altVal.VotingPower > suiteVal.VotingPower:
		return []tmtypes.PrivValidator{altPrivVal, suitePrivVal}
	case altVal.VotingPower < suiteVal.VotingPower:
		return []tmtypes.PrivValidator{suitePrivVal, altPrivVal}
	default:
		if bytes.Compare(altVal.Address, suiteVal.Address) == -1 {
			return []tmtypes.PrivValidator{altPrivVal, suitePrivVal}
		}
		return []tmtypes.PrivValidator{suitePrivVal, altPrivVal}
	}
}

// UpdateTimeForChain updates the clock for this chain.
func (chain *TestChainTendermint) UpdateCurrentHeaderTime(t time.Time) {
	chain.CurrentHeader.Time = t
}

// BeginBlock signals the beginning of a block with chain.CurrentHeader
func (chain *TestChainTendermint) BeginBlock() {
	chain.TC.App.BeginBlock(abci.RequestBeginBlock{Header: chain.CurrentHeader})
}

// ClientConfigToState builds the ClientState based on the clientConfig and last header
func (chain *TestChainTendermint) ClientConfigToState(clientConfig ClientConfig) exported.ClientState {
	tmConfig, ok := clientConfig.(*TendermintConfig)
	require.True(chain.TC.T, ok)

	height := chain.LastHeader.GetHeight().(clienttypes.Height)
	clientState := ibctmtypes.NewClientState(
		chain.TC.ChainID, tmConfig.TrustLevel, tmConfig.TrustingPeriod, tmConfig.UnbondingPeriod, tmConfig.MaxClockDrift,
		height, commitmenttypes.GetSDKSpecs(), UpgradePath, tmConfig.AllowUpdateAfterExpiry, tmConfig.AllowUpdateAfterMisbehaviour,
	)
	return clientState
}

// GetConsensusState returns the consensus state of the last header
func (chain *TestChainTendermint) GetConsensusState() exported.ConsensusState {
	return chain.LastHeader.ConsensusState()
}

func (chain *TestChainTendermint) GetLastHeader() interface{} {
	return chain.LastHeader
}
