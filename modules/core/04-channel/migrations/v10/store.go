package v10

import (
	"errors"
	fmt "fmt"

	corestore "cosmossdk.io/core/store"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	// ParamsKey defines the key to store the params in the keeper.
	ParamsKey               = "channelParams"
	KeyPruningSequenceStart = "pruningSequenceStart"
	KeyPortPrefix           = "ports"
	KeyChannelPrefix        = "channels"

	KeyChannelUpgradePrefix = "channelUpgrades"
	KeyUpgradePrefix        = "upgrades"
	KeyUpgradeErrorPrefix   = "upgradeError"
	KeyCounterpartyUpgrade  = "counterpartyUpgrade"
)

// GetUpgradeErrorReceipt returns the upgrade error receipt for the provided port and channel identifiers.
func GetUpgradeErrorReceipt(ctx sdk.Context, storeService corestore.KVStoreService, cdc codec.BinaryCodec, portID, channelID string) (ErrorReceipt, bool) {
	store := storeService.OpenKVStore(ctx)
	bz, err := store.Get(ChannelUpgradeErrorKey(portID, channelID))
	if err != nil {
		panic(err)
	}

	if len(bz) == 0 {
		return ErrorReceipt{}, false
	}

	var errorReceipt ErrorReceipt
	cdc.MustUnmarshal(bz, &errorReceipt)

	return errorReceipt, true
}

// setUpgradeErrorReceipt sets the provided error receipt in store using the port and channel identifiers.
func setUpgradeErrorReceipt(ctx sdk.Context, storeService corestore.KVStoreService, cdc codec.BinaryCodec, portID, channelID string, errorReceipt ErrorReceipt) {
	store := storeService.OpenKVStore(ctx)
	bz := cdc.MustMarshal(&errorReceipt)
	if err := store.Set(ChannelUpgradeErrorKey(portID, channelID), bz); err != nil {
		panic(err)
	}
}

// ChannelUpgradeErrorKey returns the store key for a particular channelEnd used to stor the ErrorReceipt in the case that a chain does not accept the proposed upgrade
func ChannelUpgradeErrorKey(portID, channelID string) []byte {
	return []byte(fmt.Sprintf("%s/%s/%s", KeyChannelUpgradePrefix, KeyUpgradeErrorPrefix, channelPath(portID, channelID)))
}

// ChannelUpgradeKey returns the store key for a particular channel upgrade attempt
func ChannelUpgradeKey(portID, channelID string) []byte {
	return []byte(fmt.Sprintf("%s/%s/%s", KeyChannelUpgradePrefix, KeyUpgradePrefix, channelPath(portID, channelID)))
}

// ChannelCounterpartyUpgradeKey returns the store key for the upgrade used on the counterparty channel.
func ChannelCounterpartyUpgradeKey(portID, channelID string) []byte {
	return []byte(fmt.Sprintf("%s/%s/%s", KeyChannelUpgradePrefix, KeyCounterpartyUpgrade, channelPath(portID, channelID)))
}

// PruningSequenceStartKey returns the store key for the pruning sequence start of a particular channel
func PruningSequenceStartKey(portID, channelID string) []byte {
	return []byte(fmt.Sprintf("%s/%s", KeyPruningSequenceStart, channelPath(portID, channelID)))
}

func channelPath(portID, channelID string) string {
	return fmt.Sprintf("%s/%s/%s/%s", KeyPortPrefix, portID, KeyChannelPrefix, channelID)
}

// SetParams sets the channel parameters.
func SetParams(ctx sdk.Context, storeService corestore.KVStoreService, cdc codec.BinaryCodec, params Params) {
	store := storeService.OpenKVStore(ctx)
	bz := cdc.MustMarshal(&params)
	if err := store.Set([]byte(ParamsKey), bz); err != nil {
		panic(err)
	}
}

// GetParams returns the total set of the channel parameters.
func GetParams(ctx sdk.Context, storeService corestore.KVStoreService, cdc codec.BinaryCodec) Params {
	store := storeService.OpenKVStore(ctx)
	bz, err := store.Get([]byte(ParamsKey))
	if err != nil {
		panic(err)
	}

	if bz == nil { // only panic on unset params and not on empty params
		panic(errors.New("channel params are not set in store"))
	}

	var params Params
	cdc.MustUnmarshal(bz, &params)
	return params
}

func DeleteParams(ctx sdk.Context, storeService corestore.KVStoreService) {
	store := storeService.OpenKVStore(ctx)
	store.Delete([]byte(ParamsKey))
}

// hasUpgrade returns true if a proposed upgrade exists in store
func hasUpgrade(ctx sdk.Context, storeService corestore.KVStoreService, cdc codec.BinaryCodec, portID, channelID string) bool {
	store := storeService.OpenKVStore(ctx)
	has, err := store.Has(ChannelUpgradeKey(portID, channelID))
	if err != nil {
		panic(err)
	}
	return has
}

// GetUpgrade returns the proposed upgrade for the provided port and channel identifiers.
func GetUpgrade(ctx sdk.Context, storeService corestore.KVStoreService, cdc codec.BinaryCodec, portID, channelID string) (Upgrade, bool) {
	store := storeService.OpenKVStore(ctx)
	bz, err := store.Get(ChannelUpgradeKey(portID, channelID))
	if err != nil {
		panic(err)
	}

	if len(bz) == 0 {
		return Upgrade{}, false
	}

	var upgrade Upgrade
	cdc.MustUnmarshal(bz, &upgrade)

	return upgrade, true
}

// SetUpgrade sets the proposed upgrade using the provided port and channel identifiers.
func SetUpgrade(ctx sdk.Context, storeService corestore.KVStoreService, cdc codec.BinaryCodec, portID, channelID string, upgrade Upgrade) {
	store := storeService.OpenKVStore(ctx)
	bz := cdc.MustMarshal(&upgrade)
	if err := store.Set(ChannelUpgradeKey(portID, channelID), bz); err != nil {
		panic(err)
	}
}

// deleteUpgrade deletes the upgrade for the provided port and channel identifiers.
func deleteUpgrade(ctx sdk.Context, storeService corestore.KVStoreService, cdc codec.BinaryCodec, portID, channelID string) {
	store := storeService.OpenKVStore(ctx)
	if err := store.Delete(ChannelUpgradeKey(portID, channelID)); err != nil {
		panic(err)
	}
}

// hasCounterpartyUpgrade returns true if a counterparty upgrade exists in store
func hasCounterpartyUpgrade(ctx sdk.Context, storeService corestore.KVStoreService, cdc codec.BinaryCodec, portID, channelID string) bool {
	store := storeService.OpenKVStore(ctx)
	has, err := store.Has(ChannelCounterpartyUpgradeKey(portID, channelID))
	if err != nil {
		panic(err)
	}
	return has
}

// GetCounterpartyUpgrade gets the counterparty upgrade from the store.
func GetCounterpartyUpgrade(ctx sdk.Context, storeService corestore.KVStoreService, cdc codec.BinaryCodec, portID, channelID string) (Upgrade, bool) {
	store := storeService.OpenKVStore(ctx)
	bz, err := store.Get(ChannelCounterpartyUpgradeKey(portID, channelID))
	if err != nil {
		panic(err)
	}

	if len(bz) == 0 {
		return Upgrade{}, false
	}

	var upgrade Upgrade
	cdc.MustUnmarshal(bz, &upgrade)

	return upgrade, true
}

// SetCounterpartyUpgrade sets the counterparty upgrade in the store.
func SetCounterpartyUpgrade(ctx sdk.Context, storeService corestore.KVStoreService, cdc codec.BinaryCodec, portID, channelID string, upgrade Upgrade) {
	store := storeService.OpenKVStore(ctx)
	bz := cdc.MustMarshal(&upgrade)
	if err := store.Set(ChannelCounterpartyUpgradeKey(portID, channelID), bz); err != nil {
		panic(err)
	}
}

// deleteCounterpartyUpgrade deletes the counterparty upgrade in the store.
func deleteCounterpartyUpgrade(ctx sdk.Context, storeService corestore.KVStoreService, cdc codec.BinaryCodec, portID, channelID string) {
	store := storeService.OpenKVStore(ctx)
	if err := store.Delete(ChannelCounterpartyUpgradeKey(portID, channelID)); err != nil {
		panic(err)
	}
}

// deleteUpgradeInfo deletes all auxiliary upgrade information.
func deleteUpgradeInfo(ctx sdk.Context, storeService corestore.KVStoreService, cdc codec.BinaryCodec, portID, channelID string) {
	// k.deleteUpgrade(ctx, portID, channelID)
	// k.deleteCounterpartyUpgrade(ctx, portID, channelID)
}
