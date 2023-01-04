package ibctesting

import (
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
	ibcdmtypes "github.com/cosmos/ibc-go/v3/modules/light-clients/01-dymint/types"
	"github.com/cosmos/ibc-go/v3/testing/mock"
)

type DymintConfig struct {
	TrustingPeriod time.Duration
	MaxClockDrift  time.Duration
}

func (tmcfg *DymintConfig) GetClientType() string {
	return exported.Dymint
}

var _ ClientConfig = &DymintConfig{}

// TestChainDymint is a testing struct that 'wraps' a TestChain with the last DM Header,
// the current ABCI header.
type TestChainDymint struct {
	TC *TestChain

	LastHeader    *ibcdmtypes.Header // header for last block height committed
	CurrentHeader tmproto.Header     // header for current block height

}

var _ TestChainClientI = &TestChainDymint{}

// NewChainDymintClient initializes the consunsus spesisifc pare of the TestChain
func NewChainDymintClient(tc *TestChain) *TestChainDymint {

	// create current header and call begin block
	header := tmproto.Header{
		ChainID: tc.ChainID,
		Height:  1,
		Time:    tc.Coordinator.CurrentTime.UTC(),
	}

	// create an account to send transactions from
	chain := &TestChainDymint{
		tc,
		nil,
		header,
	}

	return chain
}

func (chain *TestChainDymint) GetSelfClientType() string {
	return exported.Dymint
}

func (chain *TestChainDymint) NewConfig() ClientConfig {
	return &DymintConfig{
		TrustingPeriod: TrustingPeriod,
		MaxClockDrift:  MaxClockDrift,
	}
}

// GetContext returns the current context for the application.
func (chain *TestChainDymint) GetContext() sdk.Context {
	return chain.TC.App.GetBaseApp().NewContext(false, chain.CurrentHeader)
}

// NextBlock sets the last header to the current header and increments the current header to be
// at the next block height. It does not update the time as that is handled by the Coordinator.
//
// CONTRACT: this function must only be called after app.Commit() occurs
func (chain *TestChainDymint) NextBlock() {
	// set the last header to the current header
	// use nil trusted fields
	chain.LastHeader = chain.CurrentDMClientHeader()

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

// ConstructUpdateDMClientHeader will construct a valid 01-dymint Header to update the
// light client on the source chain.
func ConstructUpdateDMClientHeaderWithTrustedHeight(counterparty *TestChain, clientID string, trustedHeight clienttypes.Height) (*ibcdmtypes.Header, error) {
	header := counterparty.TestChainClient.GetLastHeader().(*ibcdmtypes.Header)

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
			return nil, sdkerrors.Wrapf(ibcdmtypes.ErrInvalidHeaderHeight, "could not retrieve trusted validators at trustedHeight: %d", trustedHeight)
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

// CurrentDMClientHeader creates a DM header using the current header parameters
// on the chain. The trusted fields in the header are set to nil.
func (chain *TestChainDymint) CurrentDMClientHeader() *ibcdmtypes.Header {
	return chain.CreateDMClientHeader(chain.TC.ChainID, chain.CurrentHeader.Height, clienttypes.Height{}, chain.CurrentHeader.Time, chain.TC.Vals, nil, chain.TC.Signers)
}

// CreateDMClientHeader creates a DM header to update the DM client. Args are passed in to allow
// caller flexibility to use params that differ from the chain.
func (chain *TestChainDymint) CreateDMClientHeader(chainID string, blockHeight int64, trustedHeight clienttypes.Height, timestamp time.Time, tmValSet, tmTrustedVals *tmtypes.ValidatorSet, signers []tmtypes.PrivValidator) *ibcdmtypes.Header {
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

	// only one sequencer can sign
	pv, ok := signers[0].(mock.PV)
	require.True(chain.TC.T, ok)
	headerBytes, err := tmHeader.ToProto().Marshal()
	require.NoError(chain.TC.T, err)
	signedBytes, err := pv.PrivKey.Sign(headerBytes)
	require.NoError(chain.TC.T, err)

	// Dymint check the header bytes signatures
	signedHeader.Commit.Signatures[0].Signature = signedBytes

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
	return &ibcdmtypes.Header{
		SignedHeader:      signedHeader,
		ValidatorSet:      valSet,
		TrustedHeight:     trustedHeight,
		TrustedValidators: trustedVals,
	}
}

// UpdateTimeForChain updates the clock for this chain.
func (chain *TestChainDymint) UpdateCurrentHeaderTime(t time.Time) {
	chain.CurrentHeader.Time = t
}

// BeginBlock signals the beginning of a block with chain.CurrentHeader
func (chain *TestChainDymint) BeginBlock() {
	chain.TC.App.BeginBlock(abci.RequestBeginBlock{Header: chain.CurrentHeader})
}

// ClientConfigToState builds the ClientState based on the clientConfig and last header
func (chain *TestChainDymint) ClientConfigToState(clientConfig ClientConfig) exported.ClientState {
	tmConfig, ok := clientConfig.(*DymintConfig)
	require.True(chain.TC.T, ok)

	height := chain.LastHeader.GetHeight().(clienttypes.Height)
	clientState := ibcdmtypes.NewClientState(
		chain.TC.ChainID, tmConfig.TrustingPeriod, tmConfig.MaxClockDrift,
		height, commitmenttypes.GetSDKSpecs(), UpgradePath,
	)
	return clientState
}

// GetConsensusState returns the consensus state of the last header
func (chain *TestChainDymint) GetConsensusState() exported.ConsensusState {
	return chain.LastHeader.ConsensusState()
}

func (chain *TestChainDymint) GetLastHeader() interface{} {
	return chain.LastHeader
}
