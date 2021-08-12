package fee

import (
	"context"
	"encoding/json"
	"math/rand"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/types/module"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	capabilitytypes "github.com/cosmos/cosmos-sdk/x/capability/types"
	"github.com/gorilla/mux"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/spf13/cobra"
	abci "github.com/tendermint/tendermint/abci/types"

	capabilitykeeper "github.com/cosmos/cosmos-sdk/x/capability/keeper"
	"github.com/cosmos/ibc-go/modules/apps/29-fee/client/cli"
	"github.com/cosmos/ibc-go/modules/apps/29-fee/keeper"
	"github.com/cosmos/ibc-go/modules/apps/29-fee/types"

	// "github.com/cosmos/ibc-go/modules/apps/29-fee/client/cli"
	// "github.com/cosmos/ibc-go/modules/apps/29-fee/keeper"
	// "github.com/cosmos/ibc-go/modules/apps/29-fee/simulation"
	channeltypes "github.com/cosmos/ibc-go/modules/core/04-channel/types"
	porttypes "github.com/cosmos/ibc-go/modules/core/05-port/types"
	host "github.com/cosmos/ibc-go/modules/core/24-host"
	ibcexported "github.com/cosmos/ibc-go/modules/core/exported"
)

var (
	_ module.AppModule      = AppModule{}
	_ porttypes.IBCModule   = AppModule{}
	_ module.AppModuleBasic = AppModuleBasic{}
)

// AppModuleBasic is the 29-fee AppModuleBasic
type AppModuleBasic struct{}

// Name implements AppModuleBasic interface
func (AppModuleBasic) Name() string {
	return types.ModuleName
}

// RegisterLegacyAminoCodec implements AppModuleBasic interface
func (AppModuleBasic) RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {}

// RegisterInterfaces registers module concrete types into protobuf Any.
func (AppModuleBasic) RegisterInterfaces(registry codectypes.InterfaceRegistry) {
	//	types.RegisterInterfaces(registry)
}

// DefaultGenesis returns default genesis state as raw bytes for the ibc
// 29-fee module.
func (AppModuleBasic) DefaultGenesis(cdc codec.JSONCodec) json.RawMessage {
	//	return cdc.MustMarshalJSON(types.DefaultGenesisState())
	return nil
}

// ValidateGenesis performs genesis state validation for the 29-fee module.
func (AppModuleBasic) ValidateGenesis(cdc codec.JSONCodec, config client.TxEncodingConfig, bz json.RawMessage) error {
	// var gs types.GenesisState
	// if err := cdc.UnmarshalJSON(bz, &gs); err != nil {
	// 	return fmt.Errorf("failed to unmarshal %s genesis state: %w", types.ModuleName, err)
	// }

	// return gs.Validate()
	return nil
}

// RegisterRESTRoutes implements AppModuleBasic interface
func (AppModuleBasic) RegisterRESTRoutes(clientCtx client.Context, rtr *mux.Router) {
}

// RegisterGRPCGatewayRoutes registers the gRPC Gateway routes for the ibc-29-fee module.
func (AppModuleBasic) RegisterGRPCGatewayRoutes(clientCtx client.Context, mux *runtime.ServeMux) {
	types.RegisterQueryHandlerClient(context.Background(), mux, types.NewQueryClient(clientCtx))
}

// GetTxCmd implements AppModuleBasic interface
func (AppModuleBasic) GetTxCmd() *cobra.Command {
	return cli.NewTxCmd()
}

// GetQueryCmd implements AppModuleBasic interface
func (AppModuleBasic) GetQueryCmd() *cobra.Command {
	return cli.GetQueryCmd()
}

// AppModule represents the AppModule for this module
type AppModule struct {
	AppModuleBasic
	keeper       keeper.Keeper
	scopedKeeper capabilitykeeper.ScopedKeeper
	app          porttypes.IBCModule
}

// NewAppModule creates a new 29-fee module
func NewAppModule(k keeper.Keeper, app porttypes.IBCModule) AppModule {
	return AppModule{
		keeper: k,
		app:    app,
	}
}

// RegisterInvariants implements the AppModule interface
func (AppModule) RegisterInvariants(ir sdk.InvariantRegistry) {
	// TODO
}

// Route implements the AppModule interface
func (am AppModule) Route() sdk.Route {
	return sdk.Route{}
}

// QuerierRoute implements the AppModule interface
func (AppModule) QuerierRoute() string {
	return types.QuerierRoute
}

// LegacyQuerierHandler implements the AppModule interface
func (am AppModule) LegacyQuerierHandler(*codec.LegacyAmino) sdk.Querier {
	return nil
}

// RegisterServices registers module services.
func (am AppModule) RegisterServices(cfg module.Configurator) {
	// types.RegisterMsgServer(cfg.MsgServer(), am.keeper)
	// types.RegisterQueryServer(cfg.QueryServer(), am.keeper)
}

// InitGenesis performs genesis initialization for the ibc-29-fee module. It returns
// no validator updates.
func (am AppModule) InitGenesis(ctx sdk.Context, cdc codec.JSONCodec, data json.RawMessage) []abci.ValidatorUpdate {
	// var genesisState types.GenesisState
	// cdc.MustUnmarshalJSON(data, &genesisState)
	// am.keeper.InitGenesis(ctx, genesisState)
	return []abci.ValidatorUpdate{}
}

// ExportGenesis returns the exported genesis state as raw bytes for the ibc-29-fee
// module.
func (am AppModule) ExportGenesis(ctx sdk.Context, cdc codec.JSONCodec) json.RawMessage {
	// gs := am.keeper.ExportGenesis(ctx)
	// return cdc.MustMarshalJSON(gs)
	return nil
}

// ConsensusVersion implements AppModule/ConsensusVersion.
func (AppModule) ConsensusVersion() uint64 { return 1 }

// BeginBlock implements the AppModule interface
func (am AppModule) BeginBlock(ctx sdk.Context, req abci.RequestBeginBlock) {
}

// EndBlock implements the AppModule interface
func (am AppModule) EndBlock(ctx sdk.Context, req abci.RequestEndBlock) []abci.ValidatorUpdate {
	return []abci.ValidatorUpdate{}
}

// AppModuleSimulation functions

// GenerateGenesisState creates a randomized GenState of the 29-fee module.
func (AppModule) GenerateGenesisState(simState *module.SimulationState) {
	// simulation.RandomizedGenState(simState)
}

// ProposalContents doesn't return any content functions for governance proposals.
func (AppModule) ProposalContents(_ module.SimulationState) []simtypes.WeightedProposalContent {
	return nil
}

// RandomizedParams creates randomized ibc-29-fee param changes for the simulator.
func (AppModule) RandomizedParams(r *rand.Rand) []simtypes.ParamChange {
	// return simulation.ParamChanges(r)
	return nil
}

// RegisterStoreDecoder registers a decoder for 29-fee module's types
func (am AppModule) RegisterStoreDecoder(sdr sdk.StoreDecoderRegistry) {
	// sdr[types.StoreKey] = simulation.NewDecodeStore(am.keeper)
}

// WeightedOperations returns the all the 29-fee module operations with their respective weights.
func (am AppModule) WeightedOperations(_ module.SimulationState) []simtypes.WeightedOperation {
	return nil
}

// OnChanOpenInit implements the IBCModule interface
func (am AppModule) OnChanOpenInit(
	ctx sdk.Context,
	order channeltypes.Order,
	connectionHops []string,
	portID string,
	channelID string,
	chanCap *capabilitytypes.Capability,
	counterparty channeltypes.Counterparty,
	version string,
) error {
	feeVersion, appVersion := channeltypes.SplitChannelVersion(version)
	if feeVersion != types.Version {
		return sdkerrors.Wrapf(types.ErrInvalidVersion, "expected: %s, got: %s", types.Version, feeVersion)
	}
	// Claim channel capability passed back by IBC module
	if err := am.scopedKeeper.ClaimCapability(ctx, chanCap, host.ChannelCapabilityPath(portID, channelID)); err != nil {
		return err
	}

	appCap, err := am.scopedKeeper.NewCapability(ctx, types.AppCapabilityName(channelID, portID))
	if err != nil {
		return sdkerrors.Wrap(err, "could not create capability for underlying application")
	}

	// call underlying app's OnChanOpenInit callback with the appVersion
	return am.app.OnChanOpenInit(ctx, order, connectionHops, portID, channelID,
		appCap, counterparty, appVersion)
}

// OnChanOpenTry implements the IBCModule interface
func (am AppModule) OnChanOpenTry(
	ctx sdk.Context,
	order channeltypes.Order,
	connectionHops []string,
	portID,
	channelID string,
	chanCap *capabilitytypes.Capability,
	counterparty channeltypes.Counterparty,
	version,
	counterpartyVersion string,
) error {
	feeVersion, appVersion := channeltypes.SplitChannelVersion(version)
	cpFeeVersion, cpAppVersion := channeltypes.SplitChannelVersion(counterpartyVersion)

	if feeVersion != types.Version {
		return sdkerrors.Wrapf(types.ErrInvalidVersion, "expected: %s, got: %s", types.Version, feeVersion)
	}
	if cpFeeVersion != feeVersion {
		return sdkerrors.Wrapf(types.ErrInvalidVersion, "expected counterparty version: %s, got: %s", types.Version, cpFeeVersion)
	}
	var (
		appCap *capabilitytypes.Capability
		err    error
		ok     bool
	)
	// Module may have already claimed capability in OnChanOpenInit in the case of crossing hellos
	// (ie chainA and chainB both call ChanOpenInit before one of them calls ChanOpenTry)
	// If module can already authenticate the capability then module already owns it so we don't need to claim
	// Otherwise, module does not have channel capability and we must claim it from IBC
	if !am.scopedKeeper.AuthenticateCapability(ctx, chanCap, host.ChannelCapabilityPath(portID, channelID)) {
		// Only claim channel capability passed back by IBC module if we do not already own it
		if err := am.scopedKeeper.ClaimCapability(ctx, chanCap, host.ChannelCapabilityPath(portID, channelID)); err != nil {
			return err
		}
		appCap, err = am.scopedKeeper.NewCapability(ctx, types.AppCapabilityName(channelID, portID))
		if err != nil {
			return sdkerrors.Wrap(err, "could not create capability for underlying app")
		}
	}
	appCap, ok = am.scopedKeeper.GetCapability(ctx, types.AppCapabilityName(channelID, portID))
	if !ok {
		return sdkerrors.Wrap(capabilitytypes.ErrCapabilityNotFound,
			"could not find app capability on OnChanOpenTry even after OnChanOpenInit called on this chain first (crossing hellos)")
	}
	// call underlying app's OnChanOpenTry callback with the app versions
	return am.app.OnChanOpenTry(ctx, order, connectionHops, portID, channelID,
		appCap, counterparty, appVersion, cpAppVersion)
}

// OnChanOpenAck implements the IBCModule interface
func (am AppModule) OnChanOpenAck(
	ctx sdk.Context,
	portID,
	channelID string,
	counterpartyVersion string,
) error {
	cpFeeVersion, cpAppVersion := channeltypes.SplitChannelVersion(counterpartyVersion)

	if cpFeeVersion != types.Version {
		return sdkerrors.Wrapf(types.ErrInvalidVersion, "expected counterparty version: %s, got: %s", types.Version, cpFeeVersion)
	}
	// call underlying app's OnChanOpenAck callback with the counterparty app version.
	return am.app.OnChanOpenAck(ctx, portID, channelID, cpAppVersion)
}

// OnChanOpenConfirm implements the IBCModule interface
func (am AppModule) OnChanOpenConfirm(
	ctx sdk.Context,
	portID,
	channelID string,
) error {
	// call underlying app's OnChanOpenConfirm callback.
	return am.app.OnChanOpenConfirm(ctx, portID, channelID)
}

// OnChanCloseInit implements the IBCModule interface
func (am AppModule) OnChanCloseInit(
	ctx sdk.Context,
	portID,
	channelID string,
) error {
	// TODO: Unescrow all remaining funds for unprocessed packets
	return am.app.OnChanCloseInit(ctx, portID, channelID)
}

// OnChanCloseConfirm implements the IBCModule interface
func (am AppModule) OnChanCloseConfirm(
	ctx sdk.Context,
	portID,
	channelID string,
) error {
	// TODO: Unescrow all remaining funds for unprocessed packets
	return am.app.OnChanCloseConfirm(ctx, portID, channelID)
}

// OnRecvPacket implements the IBCModule interface.
func (am AppModule) OnRecvPacket(
	ctx sdk.Context,
	packet channeltypes.Packet,
	relayer sdk.AccAddress,
) ibcexported.Acknowledgement {
	// construct incentivised ack
	return am.app.OnRecvPacket(ctx, packet, relayer) 
}

// OnAcknowledgementPacket implements the IBCModule interface
func (am AppModule) OnAcknowledgementPacket(
	ctx sdk.Context,
	packet channeltypes.Packet,
	acknowledgement []byte,
	relayer sdk.AccAddress,
) error {
	// pay fee to forward and reverse relayers
	return am.app.OnAcknowledgementPacket(ctx, packet, acknowledgement, relayer) 
}

// OnTimeoutPacket implements the IBCModule interface
func (am AppModule) OnTimeoutPacket(
	ctx sdk.Context,
	packet channeltypes.Packet,
	relayer sdk.AccAddress,
) error {
	// pay timeout fee
	return am.app.OnTimeoutPacket(ctx, packet, relayer)
}
