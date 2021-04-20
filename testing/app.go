package ibctesting

import (
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	capabilitykeeper "github.com/cosmos/cosmos-sdk/x/capability/keeper"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	abci "github.com/tendermint/tendermint/abci/types"

	"github.com/cosmos/ibc-go/modules/core/keeper"
)

type TestingApp interface {
	abci.Application

	// ibc-go additions
	GetBaseApp() *baseapp.BaseApp
	GetStakingKeeper() stakingkeeper.Keeper
	GetIBCKeeper() *keeper.Keeper
	GetScopedIBCKeeper() capabilitykeeper.ScopedKeeper      // TODO remove
	GetScopedTransferKeeper() capabilitykeeper.ScopedKeeper // TODO remove
	GetScopedIBCMockKeeper() capabilitykeeper.ScopedKeeper  // TODO remove

	// Implemented by SimApp
	AppCodec() codec.Marshaler

	// Implemented by BaseApp
	LastCommitID() sdk.CommitID
	LastBlockHeight() int64
	Query(req abci.RequestQuery) (res abci.ResponseQuery)
}
