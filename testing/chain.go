package ibctesting

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"cosmossdk.io/core/header"
	errorsmod "cosmossdk.io/errors"
	sdkmath "cosmossdk.io/math"
	banktypes "cosmossdk.io/x/bank/types"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	abci "github.com/cometbft/cometbft/api/cometbft/abci/v1"
	cmtproto "github.com/cometbft/cometbft/api/cometbft/types/v1"
	cmtprotoversion "github.com/cometbft/cometbft/api/cometbft/version/v1"
	"github.com/cometbft/cometbft/crypto/tmhash"
	cmttypes "github.com/cometbft/cometbft/types"
	cmtversion "github.com/cometbft/cometbft/version"

	clienttypes "github.com/cosmos/ibc-go/v9/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v9/modules/core/04-channel/types"
	commitmenttypes "github.com/cosmos/ibc-go/v9/modules/core/23-commitment/types"
	host "github.com/cosmos/ibc-go/v9/modules/core/24-host"
	"github.com/cosmos/ibc-go/v9/modules/core/exported"
	ibctm "github.com/cosmos/ibc-go/v9/modules/light-clients/07-tendermint"
	"github.com/cosmos/ibc-go/v9/testing/simapp"
)

var MaxAccounts = 10

type SenderAccount struct {
	SenderPrivKey cryptotypes.PrivKey
	SenderAccount sdk.AccountI
}

const (
	DefaultGenesisAccBalance = "10000000000000000000"
)

// TestChain is a testing struct that wraps a simapp with the last TM Header, the current ABCI
// header and the validators of the TestChain. It also contains a field called ChainID. This
// is the clientID that *other* chains use to refer to this TestChain. The SenderAccount
// is used for delivering transactions through the application state.
// NOTE: the actual application uses an empty chain-id for ease of testing.
type TestChain struct {
	testing.TB

	Coordinator           *Coordinator
	App                   TestingApp
	ChainID               string
	LatestCommittedHeader *ibctm.Header   // header for last block height committed
	ProposedHeader        cmtproto.Header // proposed (uncommitted) header for current block height
	TxConfig              client.TxConfig
	Codec                 codec.Codec

	Vals     *cmttypes.ValidatorSet
	NextVals *cmttypes.ValidatorSet

	// Signers is a map from validator address to the PrivValidator
	// The map is converted into an array that is the same order as the validators right before signing commit
	// This ensures that signers will always be in correct order even as validator powers change.
	// If a test adds a new validator after chain creation, then the signer map must be updated to include
	// the new PrivValidator entry.
	Signers map[string]cmttypes.PrivValidator

	// TrustedValidators is a mapping used to obtain the validator set from which we can prove a header update.
	// It maps from a header height to the next validator set associated with that header.
	TrustedValidators map[uint64]*cmttypes.ValidatorSet

	// autogenerated sender private key
	SenderPrivKey cryptotypes.PrivKey
	SenderAccount sdk.AccountI

	SenderAccounts []SenderAccount

	// Short-term solution to override the logic of the standard SendMsgs function.
	// See issue https://github.com/cosmos/ibc-go/issues/3123 for more information.
	SendMsgsOverride func(msgs ...sdk.Msg) (*abci.ExecTxResult, error)
}

// NewTestChainWithValSet initializes a new TestChain instance with the given validator set
// and signer array. It also initializes 10 Sender accounts with a balance of 10000000000000000000 coins of
// bond denom to use for tests.
//
// The first block height is committed to state in order to allow for client creations on
// counterparty chains. The TestChain will return with a block height starting at 2.
//
// Time management is handled by the Coordinator in order to ensure synchrony between chains.
// Each update of any chain increments the block header time for all chains by 5 seconds.
//
// NOTE: to use a custom sender privkey and account for testing purposes, replace and modify this
// constructor function.
//
// CONTRACT: Validator array must be provided in the order expected by Tendermint.
// i.e. sorted first by power and then lexicographically by address.
func NewTestChainWithValSet(tb testing.TB, coord *Coordinator, chainID string, valSet *cmttypes.ValidatorSet, signers map[string]cmttypes.PrivValidator) *TestChain {
	tb.Helper()
	genAccs := []authtypes.GenesisAccount{}
	genBals := []banktypes.Balance{}
	senderAccs := []SenderAccount{}

	// generate genesis accounts
	for i := 0; i < MaxAccounts; i++ {
		senderPrivKey := secp256k1.GenPrivKey()
		acc := authtypes.NewBaseAccount(senderPrivKey.PubKey().Address().Bytes(), senderPrivKey.PubKey(), uint64(i), 0)
		amount, ok := sdkmath.NewIntFromString(DefaultGenesisAccBalance)
		require.True(tb, ok)

		// add sender account
		balance := banktypes.Balance{
			Address: acc.GetAddress().String(),
			Coins: sdk.NewCoins(
				sdk.NewCoin(sdk.DefaultBondDenom, amount),
				sdk.NewCoin(SecondaryDenom, amount),
			),
		}

		genAccs = append(genAccs, acc)
		genBals = append(genBals, balance)

		senderAcc := SenderAccount{
			SenderAccount: acc,
			SenderPrivKey: senderPrivKey,
		}

		senderAccs = append(senderAccs, senderAcc)
	}

	app := SetupWithGenesisValSet(tb, valSet, genAccs, chainID, sdk.DefaultPowerReduction, genBals...)

	// create current header and call begin block
	header := cmtproto.Header{
		ChainID: chainID,
		Height:  1,
		Time:    coord.CurrentTime.UTC(),
	}

	txConfig := app.GetTxConfig()

	// create an account to send transactions from
	chain := &TestChain{
		TB:                tb,
		Coordinator:       coord,
		ChainID:           chainID,
		App:               app,
		ProposedHeader:    header,
		TxConfig:          txConfig,
		Codec:             app.AppCodec(),
		Vals:              valSet,
		NextVals:          valSet,
		Signers:           signers,
		TrustedValidators: make(map[uint64]*cmttypes.ValidatorSet, 0),
		SenderPrivKey:     senderAccs[0].SenderPrivKey,
		SenderAccount:     senderAccs[0].SenderAccount,
		SenderAccounts:    senderAccs,
	}

	// commit genesis block
	chain.NextBlock()

	return chain
}

// NewTestChain initializes a new test chain with a default of 4 validators
// Use this function if the tests do not need custom control over the validator set
func NewTestChain(t *testing.T, coord *Coordinator, chainID string) *TestChain {
	t.Helper()
	// generate validators private/public key
	var (
		validatorsPerChain = 4
		validators         []*cmttypes.Validator
		signersByAddress   = make(map[string]cmttypes.PrivValidator, validatorsPerChain)
	)

	for i := 0; i < validatorsPerChain; i++ {
		_, privVal := cmttypes.RandValidator(false, 100)
		pubKey, err := privVal.GetPubKey()
		require.NoError(t, err)
		validators = append(validators, cmttypes.NewValidator(pubKey, 1))
		signersByAddress[pubKey.Address().String()] = privVal
	}

	// construct validator set;
	// Note that the validators are sorted by voting power
	// or, if equal, by address lexical order
	valSet := cmttypes.NewValidatorSet(validators)

	return NewTestChainWithValSet(t, coord, chainID, valSet, signersByAddress)
}

// GetContext returns the current context for the application.
func (chain *TestChain) GetContext() sdk.Context {
	ctx := chain.App.GetBaseApp().NewUncachedContext(false, chain.ProposedHeader)

	// since:cosmos-sdk/v0.52 when fetching time from context, it now returns from HeaderInfo
	headerInfo := header.Info{
		Time:    chain.ProposedHeader.Time,
		ChainID: chain.ProposedHeader.ChainID,
	}

	return ctx.WithHeaderInfo(headerInfo)
}

// GetSimApp returns the SimApp to allow usage ofnon-interface fields.
// CONTRACT: This function should not be called by third parties implementing
// their own SimApp.
func (chain *TestChain) GetSimApp() *simapp.SimApp {
	app, ok := chain.App.(*simapp.SimApp)
	require.True(chain.TB, ok)

	return app
}

// QueryProof performs an abci query with the given key and returns the proto encoded merkle proof
// for the query and the height at which the proof will succeed on a tendermint verifier.
func (chain *TestChain) QueryProof(key []byte) ([]byte, clienttypes.Height) {
	return chain.QueryProofAtHeight(key, chain.App.LastBlockHeight())
}

// QueryProofAtHeight performs an abci query with the given key and returns the proto encoded merkle proof
// for the query and the height at which the proof will succeed on a tendermint verifier. Only the IBC
// store is supported
func (chain *TestChain) QueryProofAtHeight(key []byte, height int64) ([]byte, clienttypes.Height) {
	return chain.QueryProofForStore(exported.StoreKey, key, height)
}

// QueryProofForStore performs an abci query with the given key and returns the proto encoded merkle proof
// for the query and the height at which the proof will succeed on a tendermint verifier.
func (chain *TestChain) QueryProofForStore(storeKey string, key []byte, height int64) ([]byte, clienttypes.Height) {
	res, err := chain.App.Query(
		chain.GetContext().Context(),
		&abci.QueryRequest{
			Path:   fmt.Sprintf("store/%s/key", storeKey),
			Height: height - 1,
			Data:   key,
			Prove:  true,
		})
	require.NoError(chain.TB, err)

	merkleProof, err := commitmenttypes.ConvertProofs(res.ProofOps)
	require.NoError(chain.TB, err)

	proof, err := chain.App.AppCodec().Marshal(&merkleProof)
	require.NoError(chain.TB, err)

	revision := clienttypes.ParseChainID(chain.ChainID)

	// proof height + 1 is returned as the proof created corresponds to the height the proof
	// was created in the IAVL tree. Tendermint and subsequently the clients that rely on it
	// have heights 1 above the IAVL tree. Thus we return proof height + 1
	return proof, clienttypes.NewHeight(revision, uint64(res.Height)+1)
}

// QueryUpgradeProof performs an abci query with the given key and returns the proto encoded merkle proof
// for the query and the height at which the proof will succeed on a tendermint verifier.
func (chain *TestChain) QueryUpgradeProof(key []byte, height uint64) ([]byte, clienttypes.Height) {
	res, err := chain.App.Query(
		chain.GetContext().Context(),
		&abci.QueryRequest{
			Path:   "store/upgrade/key",
			Height: int64(height - 1),
			Data:   key,
			Prove:  true,
		})
	require.NoError(chain.TB, err)

	merkleProof, err := commitmenttypes.ConvertProofs(res.ProofOps)
	require.NoError(chain.TB, err)

	proof, err := chain.App.AppCodec().Marshal(&merkleProof)
	require.NoError(chain.TB, err)

	revision := clienttypes.ParseChainID(chain.ChainID)

	// proof height + 1 is returned as the proof created corresponds to the height the proof
	// was created in the IAVL tree. Tendermint and subsequently the clients that rely on it
	// have heights 1 above the IAVL tree. Thus we return proof height + 1
	return proof, clienttypes.NewHeight(revision, uint64(res.Height+1))
}

// QueryConsensusStateProof performs an abci query for a consensus state
// stored on the given clientID. The proof and consensusHeight are returned.
func (chain *TestChain) QueryConsensusStateProof(clientID string) ([]byte, clienttypes.Height) {
	consensusHeight, ok := chain.GetClientLatestHeight(clientID).(clienttypes.Height)
	require.True(chain.TB, ok)
	consensusKey := host.FullConsensusStateKey(clientID, consensusHeight)
	consensusProof, _ := chain.QueryProof(consensusKey)

	return consensusProof, consensusHeight
}

// NextBlock sets the last header to the current header and increments the current header to be
// at the next block height. It does not update the time as that is handled by the Coordinator.
// It will call FinalizeBlock and Commit and apply the validator set changes to the next validators
// of the next block being created. This follows the Tendermint protocol of applying valset changes
// returned on block `n` to the validators of block `n+2`.
// It calls BeginBlock with the new block created before returning.
func (chain *TestChain) NextBlock() {
	res, err := chain.App.FinalizeBlock(&abci.FinalizeBlockRequest{
		Height:             chain.ProposedHeader.Height,
		Time:               chain.ProposedHeader.GetTime(),
		NextValidatorsHash: chain.NextVals.Hash(),
	})
	require.NoError(chain.TB, err)
	chain.commitBlock(res)
}

func (chain *TestChain) commitBlock(res *abci.FinalizeBlockResponse) {
	_, err := chain.App.Commit()
	require.NoError(chain.TB, err)

	// set the last header to the current header
	// use nil trusted fields
	chain.LatestCommittedHeader = chain.CurrentTMClientHeader()
	// set the trusted validator set to the next validator set
	// The latest trusted validator set is the next validator set
	// associated with the header being committed in storage. This will
	// allow for header updates to be proved against these validators.
	chain.TrustedValidators[uint64(chain.ProposedHeader.Height)] = chain.NextVals

	// val set changes returned from previous block get applied to the next validators
	// of this block. See tendermint spec for details.
	chain.Vals = chain.NextVals
	chain.NextVals = ApplyValSetChanges(chain, chain.Vals, res.ValidatorUpdates)

	// increment the proposer priority of validators
	chain.Vals.IncrementProposerPriority(1)

	// increment the current header
	chain.ProposedHeader = cmtproto.Header{
		ChainID: chain.ChainID,
		Height:  chain.App.LastBlockHeight() + 1,
		AppHash: chain.App.LastCommitID().Hash,
		// NOTE: the time is increased by the coordinator to maintain time synchrony amongst
		// chains.
		Time:               chain.ProposedHeader.Time,
		ValidatorsHash:     chain.Vals.Hash(),
		NextValidatorsHash: chain.NextVals.Hash(),
		ProposerAddress:    chain.Vals.Proposer.Address,
	}
}

// sendMsgs delivers a transaction through the application without returning the result.
func (chain *TestChain) sendMsgs(msgs ...sdk.Msg) error {
	_, err := chain.SendMsgs(msgs...)
	return err
}

// SendMsgs delivers a transaction through the application using a predefined sender.
// It updates the senders sequence number and updates the TestChain's headers.
// It returns the result and error if one occurred.
func (chain *TestChain) SendMsgs(msgs ...sdk.Msg) (*abci.ExecTxResult, error) {
	senderAccount := SenderAccount{
		SenderPrivKey: chain.SenderPrivKey,
		SenderAccount: chain.SenderAccount,
	}

	return chain.SendMsgsWithSender(senderAccount, msgs...)
}

// SendMsgsWithSender delivers a transaction through the application using the provided sender.
func (chain *TestChain) SendMsgsWithSender(sender SenderAccount, msgs ...sdk.Msg) (*abci.ExecTxResult, error) {
	if chain.SendMsgsOverride != nil {
		return chain.SendMsgsOverride(msgs...)
	}
	// ensure the chain has the latest time
	chain.Coordinator.UpdateTimeForChain(chain)

	// increment acc sequence regardless of success or failure tx execution
	defer func() {
		err := sender.SenderAccount.SetSequence(sender.SenderAccount.GetSequence() + 1)
		if err != nil {
			panic(err)
		}
	}()
	resp, err := simapp.SignAndDeliver(
		chain.TB,
		chain.TxConfig,
		chain.App.GetBaseApp(),
		msgs,
		chain.ChainID,
		[]uint64{sender.SenderAccount.GetAccountNumber()},
		[]uint64{sender.SenderAccount.GetSequence()},
		true,
		chain.ProposedHeader.GetTime(),
		chain.NextVals.Hash(),
		sender.SenderPrivKey,
	)
	if err != nil {
		return nil, err
	}

	chain.commitBlock(resp)

	require.Len(chain.TB, resp.TxResults, 1)
	txResult := resp.TxResults[0]

	if txResult.Code != 0 {
		return txResult, fmt.Errorf("%s/%d: %q", txResult.Codespace, txResult.Code, txResult.Log)
	}

	chain.Coordinator.IncrementTime()

	return txResult, nil
}

// GetClientState retrieves the client state for the provided clientID. The client is
// expected to exist otherwise testing will fail.
func (chain *TestChain) GetClientState(clientID string) exported.ClientState {
	clientState, found := chain.App.GetIBCKeeper().ClientKeeper.GetClientState(chain.GetContext(), clientID)
	require.True(chain.TB, found)

	return clientState
}

// GetConsensusState retrieves the consensus state for the provided clientID and height.
// It will return a success boolean depending on if consensus state exists or not.
func (chain *TestChain) GetConsensusState(clientID string, height exported.Height) (exported.ConsensusState, bool) {
	return chain.App.GetIBCKeeper().ClientKeeper.GetClientConsensusState(chain.GetContext(), clientID, height)
}

// GetAcknowledgement retrieves an acknowledgement for the provided packet. If the
// acknowledgement does not exist then testing will fail.
func (chain *TestChain) GetAcknowledgement(packet channeltypes.Packet) []byte {
	ack, found := chain.App.GetIBCKeeper().ChannelKeeper.GetPacketAcknowledgement(chain.GetContext(), packet.GetDestPort(), packet.GetDestChannel(), packet.GetSequence())
	require.True(chain.TB, found)

	return ack
}

// GetPrefix returns the prefix for used by a chain in connection creation
func (chain *TestChain) GetPrefix() commitmenttypes.MerklePrefix {
	return commitmenttypes.NewMerklePrefix(chain.App.GetIBCKeeper().ConnectionKeeper.GetCommitmentPrefix().Bytes())
}

// ExpireClient fast forwards the chain's block time by the provided amount of time which will
// expire any clients with a trusting period less than or equal to this amount of time.
func (chain *TestChain) ExpireClient(amount time.Duration) {
	chain.Coordinator.IncrementTimeBy(amount)
}

// CurrentTMClientHeader creates a TM header using the current header parameters
// on the chain. The trusted fields in the header are set to nil.
func (chain *TestChain) CurrentTMClientHeader() *ibctm.Header {
	return chain.CreateTMClientHeader(
		chain.ChainID,
		chain.ProposedHeader.Height,
		clienttypes.Height{},
		chain.ProposedHeader.Time,
		chain.Vals,
		chain.NextVals,
		nil,
		chain.Signers,
	)
}

// CommitHeader takes in a proposed header and returns a signed cometbft header.
// The signers passed in must match the validator set provided. The signers will
// be used to sign over the proposed header.
func CommitHeader(proposedHeader cmttypes.Header, valSet *cmttypes.ValidatorSet, signers map[string]cmttypes.PrivValidator) (*cmtproto.SignedHeader, error) {
	hhash := proposedHeader.Hash()
	blockID := MakeBlockID(hhash, 3, unusedHash)
	voteSet := cmttypes.NewVoteSet(proposedHeader.ChainID, proposedHeader.Height, 1, cmtproto.PrecommitType, valSet)

	// MakeExtCommit expects a signer array in the same order as the validator array.
	// Thus we iterate over the ordered validator set and construct a signer array
	// from the signer map in the same order.
	signerArr := make([]cmttypes.PrivValidator, len(valSet.Validators))
	for i, v := range valSet.Validators { //nolint:staticcheck // need to check for nil validator set
		signerArr[i] = signers[v.Address.String()]
	}

	extCommit, err := cmttypes.MakeExtCommit(blockID, proposedHeader.Height, 1, voteSet, signerArr, proposedHeader.Time, false)
	if err != nil {
		return nil, err
	}

	signedHeader := &cmtproto.SignedHeader{
		Header: proposedHeader.ToProto(),
		Commit: extCommit.ToCommit().ToProto(),
	}

	return signedHeader, nil
}

// CreateTMClientHeader creates a TM header to update the TM client. Args are passed in to allow
// caller flexibility to use params that differ from the chain.
func (chain *TestChain) CreateTMClientHeader(chainID string, blockHeight int64, trustedHeight clienttypes.Height, timestamp time.Time, cmtValSet, nextVals, cmtTrustedVals *cmttypes.ValidatorSet, signers map[string]cmttypes.PrivValidator) *ibctm.Header {
	var (
		valSet      *cmtproto.ValidatorSet
		trustedVals *cmtproto.ValidatorSet
	)
	require.NotNil(chain.TB, cmtValSet)

	proposedHeader := cmttypes.Header{
		Version:            cmtprotoversion.Consensus{Block: cmtversion.BlockProtocol, App: 2},
		ChainID:            chainID,
		Height:             blockHeight,
		Time:               timestamp,
		LastBlockID:        MakeBlockID(make([]byte, tmhash.Size), 10_000, make([]byte, tmhash.Size)),
		LastCommitHash:     chain.App.LastCommitID().Hash,
		DataHash:           unusedHash,
		ValidatorsHash:     cmtValSet.Hash(),
		NextValidatorsHash: nextVals.Hash(),
		ConsensusHash:      unusedHash,
		AppHash:            chain.ProposedHeader.AppHash,
		LastResultsHash:    unusedHash,
		EvidenceHash:       unusedHash,
		ProposerAddress:    cmtValSet.Proposer.Address, //nolint:staticcheck
	}

	signedHeader, err := CommitHeader(proposedHeader, cmtValSet, signers)
	require.NoError(chain.TB, err)

	if cmtValSet != nil { //nolint:staticcheck
		valSet, err = cmtValSet.ToProto()
		require.NoError(chain.TB, err)
		valSet.TotalVotingPower = cmtValSet.TotalVotingPower()
	}

	if cmtTrustedVals != nil {
		trustedVals, err = cmtTrustedVals.ToProto()
		require.NoError(chain.TB, err)
		trustedVals.TotalVotingPower = cmtTrustedVals.TotalVotingPower()
	}

	// The trusted fields may be nil. They may be filled before relaying messages to a client.
	// The relayer is responsible for querying client and injecting appropriate trusted fields.
	return &ibctm.Header{
		SignedHeader:      signedHeader,
		ValidatorSet:      valSet,
		TrustedHeight:     trustedHeight,
		TrustedValidators: trustedVals,
	}
}

// MakeBlockID copied unimported test functions from cmttypes to use them here
func MakeBlockID(hash []byte, partSetSize uint32, partSetHash []byte) cmttypes.BlockID {
	return cmttypes.BlockID{
		Hash: hash,
		PartSetHeader: cmttypes.PartSetHeader{
			Total: partSetSize,
			Hash:  partSetHash,
		},
	}
}

// GetClientLatestHeight returns the latest height for the client state with the given client identifier.
// If an invalid client identifier is provided then a zero value height will be returned and testing will fail.
func (chain *TestChain) GetClientLatestHeight(clientID string) exported.Height {
	latestHeight := chain.App.GetIBCKeeper().ClientKeeper.GetClientLatestHeight(chain.GetContext(), clientID)
	require.False(chain.TB, latestHeight.IsZero())
	return latestHeight
}

// GetTimeoutHeight is a convenience function which returns a IBC packet timeout height
// to be used for testing. It returns the current IBC height + 100 blocks
func (chain *TestChain) GetTimeoutHeight() clienttypes.Height {
	return clienttypes.NewHeight(clienttypes.ParseChainID(chain.ChainID), uint64(chain.GetContext().BlockHeight())+100)
}

// GetTimeoutTimestamp is a convenience function which returns a IBC packet timeout timestamp
// to be used for testing. It returns the current block timestamp + default timestamp delta (1 hour).
func (chain *TestChain) GetTimeoutTimestamp() uint64 {
	return uint64(chain.GetContext().BlockTime().UnixNano()) + DefaultTimeoutTimestampDelta
}

// GetTimeoutTimestampSecs is a convenience function which returns a IBC packet timeout timestamp in seconds
// to be used for testing. It returns the current block timestamp + default timestamp delta (1 hour).
func (chain *TestChain) GetTimeoutTimestampSecs() uint64 {
	return uint64(chain.GetContext().BlockTime().Unix()) + uint64(time.Hour.Seconds())
}

// DeleteKey deletes the specified key from the ibc store.
func (chain *TestChain) DeleteKey(key []byte) {
	storeKey := chain.GetSimApp().GetKey(exported.StoreKey)
	kvStore := chain.GetContext().KVStore(storeKey)
	kvStore.Delete(key)
}

// IBCClientHeader will construct a 07-tendermint Header to update the light client
// on the counterparty chain. The trustedHeight must be passed in as a non-zero height.
func (chain *TestChain) IBCClientHeader(header *ibctm.Header, trustedHeight clienttypes.Height) (*ibctm.Header, error) {
	if trustedHeight.IsZero() {
		return nil, errorsmod.Wrap(ibctm.ErrInvalidHeaderHeight, "trustedHeight must be a non-zero height")
	}

	cmtTrustedVals, ok := chain.TrustedValidators[trustedHeight.RevisionHeight]
	if !ok {
		return nil, fmt.Errorf("unable to find trusted validators at height %d", trustedHeight.RevisionHeight)
	}

	trustedVals, err := cmtTrustedVals.ToProto()
	if err != nil {
		return nil, err
	}

	header.TrustedHeight = trustedHeight
	trustedVals.TotalVotingPower = cmtTrustedVals.TotalVotingPower()
	header.TrustedValidators = trustedVals

	return header, nil
}

// GetSenderAccount returns the sender account associated with the provided private key.
func (chain *TestChain) GetSenderAccount(privKey cryptotypes.PrivKey) SenderAccount {
	account := chain.GetSimApp().AuthKeeper.GetAccount(chain.GetContext(), sdk.AccAddress(privKey.PubKey().Address()))

	return SenderAccount{
		SenderPrivKey: privKey,
		SenderAccount: account,
	}
}
