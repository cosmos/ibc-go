package ibctesting

import (
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/simapp"
	capabilitykeeper "github.com/cosmos/cosmos-sdk/x/capability/keeper"
	abci "github.com/tendermint/tendermint/abci/types"

	"github.com/cosmos/ibc-go/modules/core/keeper"
)

type TestingApp interface {
	simapp.App

	GetIBCKeeper() *keeper.Keeper
	GetScopedIBCKeeper() capabilitykeeper.ScopedKeeper      // TODO remove
	GetScopedTransferKeeper() capabilitykeeper.ScopedKeeper // TODO remove
	GetScopedIBCMockKeeper() capabilitykeeper.ScopedKeeper  // TODO remove

	AppCodec() codec.Marshaler
	Query(req abci.RequestQuery) (res abci.ResponseQuery)
}
