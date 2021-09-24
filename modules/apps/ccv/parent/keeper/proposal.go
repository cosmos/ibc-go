package keeper

import (
	"encoding/binary"
	"strings"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/ibc-go/modules/apps/ccv/parent/types"
	ccv "github.com/cosmos/ibc-go/modules/apps/ccv/types"
	clienttypes "github.com/cosmos/ibc-go/modules/core/02-client/types"
	commitmenttypes "github.com/cosmos/ibc-go/modules/core/23-commitment/types"
	"github.com/cosmos/ibc-go/modules/core/exported"
	ibcexported "github.com/cosmos/ibc-go/modules/core/exported"
	ibctmtypes "github.com/cosmos/ibc-go/modules/light-clients/07-tendermint/types"
)

// CreateChildChainProposal will receive the child chain's client state from the proposal.
// If the spawn time has already passed, then set the child chain. Otherwise store the client
// as a pending client, and set once spawn time has passed.
func (k Keeper) CreateChildChainProposal(ctx sdk.Context, p *ccv.CreateChildChainProposal) error {
	clientState, err := clienttypes.UnpackClientState(p.ClientState)
	if err != nil {
		return err
	}
	if ctx.BlockTime().After(p.SpawnTime) {
		err = k.CreateChildClient(ctx, p.ChainId, clientState)
		if err != nil {
			return err
		}
		return nil
	}

	k.SetPendingClient(ctx, p.SpawnTime, p.ChainId, clientState)
	return nil
}

// CreateChildClient will create the CCV client for the given child chain. The CCV channel must be built
// on top of the CCV client to ensure connection with the right child chain.
func (k Keeper) CreateChildClient(ctx sdk.Context, chainID string, clientState ibcexported.ClientState) error {
	// TODO: Allow for current validators to set different keys
	consensusState := ibctmtypes.NewConsensusState(ctx.BlockTime(), commitmenttypes.NewMerkleRoot([]byte(ibctmtypes.SentinelRoot)), ctx.BlockHeader().NextValidatorsHash)
	clientID, err := k.clientKeeper.CreateClient(ctx, clientState, consensusState)
	if err != nil {
		return err
	}
	k.SetChildClient(ctx, chainID, clientID)
	return nil
}

// SetChildClient sets the clientID for the given chainID
func (k Keeper) SetChildClient(ctx sdk.Context, clientID, chainID string) {
	store := ctx.KVStore(k.storeKey)
	store.Set(types.ChainToClientKey(chainID), []byte(clientID))
}

// GetChildClient returns the clientID for the given chainID.
func (k Keeper) GetChildClient(ctx sdk.Context, chainID string) string {
	store := ctx.KVStore(k.storeKey)
	return string(store.Get(types.ChainToClientKey(chainID)))
}

// SetPendingClient sets an IdentifiedClient for the given timestamp
func (k Keeper) SetPendingClient(ctx sdk.Context, timestamp time.Time, chainID string, clientState ibcexported.ClientState) error {
	store := ctx.KVStore(k.storeKey)
	bz, err := k.cdc.MarshalInterface(clientState)
	if err != nil {
		return err
	}
	store.Set(types.PendingClientKey(timestamp, chainID), bz)
	return nil
}

// GetPendingClient gets an IdentifiedClient for the given timestamp
func (k Keeper) GetPendingClient(ctx sdk.Context, timestamp time.Time, chainID string) clienttypes.IdentifiedClientState {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.PendingClientKey(timestamp, chainID))
	var ic clienttypes.IdentifiedClientState
	k.cdc.MustUnmarshal(bz, &ic)
	return ic
}

// IteratePendingClients iterates over the pending clients in order and sets the child client if the spawn time has passed,
// otherwise it will break out of loop and return.
func (k Keeper) IteratePendingClients(ctx sdk.Context) {
	store := ctx.KVStore(k.storeKey)
	iterator := sdk.KVStorePrefixIterator(store, []byte(types.PendingClientKeyPrefix+"/"))
	defer iterator.Close()

	if !iterator.Valid() {
		return
	}

	for ; iterator.Valid(); iterator.Next() {
		suffixKey := iterator.Key()
		// splitKey contains the bigendian time in the first element and the chainID in the second element/
		splitKey := strings.Split(string(suffixKey), "/")

		timeNano := binary.BigEndian.Uint64([]byte(splitKey[0]))
		spawnTime := time.Unix(0, int64(timeNano))
		var cs exported.ClientState
		k.cdc.UnmarshalInterface(iterator.Value(), cs)

		if ctx.BlockTime().After(spawnTime) {
			k.CreateChildClient(ctx, splitKey[1], cs)
		} else {
			break
		}
	}
}
