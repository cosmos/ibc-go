package exported

import (
	"context"

	"github.com/cosmos/cosmos-sdk/codec"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"
	clienttypes "github.com/cosmos/ibc-go/v6/modules/core/02-client/types"
	"github.com/cosmos/ibc-go/v6/modules/core/exported"
)

// ClientKeeper defines the expected interface of the light client used for IBC
type ClientKeeper interface {
	CreateClient(ctx sdk.Context, clientState exported.ClientState, consensusState exported.ConsensusState) (string, error)
	UpdateClient(ctx sdk.Context, clientID string, header exported.Height) error
	UpgradeClient(ctx sdk.Context, clientID string, upgradedClient exported.ClientState, upgradedConsState exported.ConsensusState, proofUpgradeClient, proofUpgradeConsState []byte) error
	GenerateClientIdentifier(ctx sdk.Context, clientType string) string
	SetClientState(ctx sdk.Context, clientID string, clientState exported.ClientState)
	GetClientState(ctx sdk.Context, clientID string) (exported.ClientState, bool)
	GetClientConsensusState(ctx sdk.Context, clientID string, height exported.Height) (exported.ConsensusState, bool)
	SetClientConsensusState(ctx sdk.Context, clientID string, height exported.Height, consensusState exported.ConsensusState)
	GetNextClientSequence(ctx sdk.Context) uint64
	SetNextClientSequence(ctx sdk.Context, sequence uint64)
	IterateConsensusStates(ctx sdk.Context, cb func(clientID string, cs clienttypes.ConsensusStateWithHeight) bool)
	GetAllGenesisClients(ctx sdk.Context) clienttypes.IdentifiedClientStates
	GetAllClientMetadata(ctx sdk.Context, genClients []clienttypes.IdentifiedClientState) ([]clienttypes.IdentifiedGenesisMetadata, error)
	SetAllClientMetadata(ctx sdk.Context, genMetadata []clienttypes.IdentifiedGenesisMetadata)
	GetAllConsensusStates(ctx sdk.Context) clienttypes.ClientsConsensusStates
	HasClientConsensusState(ctx sdk.Context, clientID string, height exported.Height) bool
	GetLatestClientConsensusState(ctx sdk.Context, clientID string) (exported.ConsensusState, bool)
	GetSelfConsensusState(ctx sdk.Context, height exported.Height) (exported.ConsensusState, error)
	ValidateSelfClient(ctx sdk.Context, clientState exported.ClientState) error
	GetUpgradePlan(ctx sdk.Context) (plan upgradetypes.Plan, havePlan bool)
	GetUpgradedClient(ctx sdk.Context, planHeight int64) ([]byte, bool)
	GetUpgradedConsensusState(ctx sdk.Context, planHeight int64) ([]byte, bool)
	SetUpgradedConsensusState(ctx sdk.Context, planHeight int64, bz []byte) error
	IterateClients(ctx sdk.Context, cb func(clientID string, cs exported.ClientState) bool)
	GetAllClients(ctx sdk.Context) (states []exported.ClientState)
	ClientStore(ctx sdk.Context, clientID string) sdk.KVStore

	// encoding-related functions
	UnmarshalClientState(bz []byte) (exported.ClientState, error)
	MustUnmarshalClientState(bz []byte) exported.ClientState
	UnmarshalConsensusState(bz []byte) (exported.ConsensusState, error)
	MustUnmarshalConsensusState(bz []byte) exported.ConsensusState
	MustMarshalClientState(clientState exported.ClientState) []byte
	MustMarshalConsensusState(consensusState exported.ConsensusState) []byte
	GetStoreKey() storetypes.StoreKey
	GetCdc() codec.BinaryCodec

	// params-related functions
	GetAllowedClients(ctx sdk.Context) []string
	GetParams(ctx sdk.Context) clienttypes.Params
	SetParams(ctx sdk.Context, params clienttypes.Params)

	// proposal-related functions
	ClientUpdateProposal(ctx sdk.Context, p *clienttypes.ClientUpdateProposal) error
	HandleUpgradeProposal(ctx sdk.Context, p *clienttypes.UpgradeProposal) error

	// GRPC query functions
	ClientState(context.Context, *clienttypes.QueryClientStateRequest) (*clienttypes.QueryClientStateResponse, error)
	ClientStates(context.Context, *clienttypes.QueryClientStatesRequest) (*clienttypes.QueryClientStatesResponse, error)
	ConsensusState(context.Context, *clienttypes.QueryConsensusStateRequest) (*clienttypes.QueryConsensusStateResponse, error)
	ConsensusStates(context.Context, *clienttypes.QueryConsensusStatesRequest) (*clienttypes.QueryConsensusStatesResponse, error)
	ConsensusStateHeights(context.Context, *clienttypes.QueryConsensusStateHeightsRequest) (*clienttypes.QueryConsensusStateHeightsResponse, error)
	ClientStatus(context.Context, *clienttypes.QueryClientStatusRequest) (*clienttypes.QueryClientStatusResponse, error)
	ClientParams(context.Context, *clienttypes.QueryClientParamsRequest) (*clienttypes.QueryClientParamsResponse, error)
	UpgradedClientState(context.Context, *clienttypes.QueryUpgradedClientStateRequest) (*clienttypes.QueryUpgradedClientStateResponse, error)
	UpgradedConsensusState(context.Context, *clienttypes.QueryUpgradedConsensusStateRequest) (*clienttypes.QueryUpgradedConsensusStateResponse, error)
}
