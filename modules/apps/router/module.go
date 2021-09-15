package router

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"

	"github.com/armon/go-metrics"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	capabilitytypes "github.com/cosmos/cosmos-sdk/x/capability/types"
	"github.com/gorilla/mux"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/spf13/cobra"
	abci "github.com/tendermint/tendermint/abci/types"

	"github.com/cosmos/ibc-go/modules/apps/router/client/cli"
	"github.com/cosmos/ibc-go/modules/apps/router/keeper"
	"github.com/cosmos/ibc-go/modules/apps/router/types"
	transfertypes "github.com/cosmos/ibc-go/modules/apps/transfer/types"
	channeltypes "github.com/cosmos/ibc-go/modules/core/04-channel/types"
	porttypes "github.com/cosmos/ibc-go/modules/core/05-port/types"
	ibcexported "github.com/cosmos/ibc-go/modules/core/exported"
)

var (
	_ module.AppModule      = AppModule{}
	_ porttypes.IBCModule   = AppModule{}
	_ module.AppModuleBasic = AppModuleBasic{}
)

// AppModuleBasic is the router AppModuleBasic
type AppModuleBasic struct{}

// Name implements AppModuleBasic interface
func (AppModuleBasic) Name() string {
	return types.ModuleName
}

// RegisterLegacyAminoCodec implements AppModuleBasic interface
func (AppModuleBasic) RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {}

// RegisterInterfaces registers module concrete types into protobuf Any.
func (AppModuleBasic) RegisterInterfaces(registry codectypes.InterfaceRegistry) {}

// DefaultGenesis returns default genesis state as raw bytes for the ibc
// router module.
func (AppModuleBasic) DefaultGenesis(cdc codec.JSONCodec) json.RawMessage {
	return cdc.MustMarshalJSON(types.DefaultGenesisState())
}

// ValidateGenesis performs genesis state validation for the router module.
func (AppModuleBasic) ValidateGenesis(cdc codec.JSONCodec, config client.TxEncodingConfig, bz json.RawMessage) error {
	var gs types.GenesisState
	if err := cdc.UnmarshalJSON(bz, &gs); err != nil {
		return fmt.Errorf("failed to unmarshal %s genesis state: %w", types.ModuleName, err)
	}

	return gs.Validate()
}

// RegisterRESTRoutes implements AppModuleBasic interface
func (AppModuleBasic) RegisterRESTRoutes(clientCtx client.Context, rtr *mux.Router) {}

// RegisterGRPCGatewayRoutes registers the gRPC Gateway routes for the ibc-router module.
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
	keeper keeper.Keeper
	app    porttypes.IBCModule
}

// NewAppModule creates a new router module
func NewAppModule(k keeper.Keeper, app porttypes.IBCModule) AppModule {
	return AppModule{
		keeper: k,
		app:    app,
	}
}

// RegisterInvariants implements the AppModule interface
func (AppModule) RegisterInvariants(ir sdk.InvariantRegistry) {}

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
	types.RegisterQueryServer(cfg.QueryServer(), am.keeper)
}

// InitGenesis performs genesis initialization for the ibc-router module. It returns
// no validator updates.
func (am AppModule) InitGenesis(ctx sdk.Context, cdc codec.JSONCodec, data json.RawMessage) []abci.ValidatorUpdate {
	var genesisState types.GenesisState
	cdc.MustUnmarshalJSON(data, &genesisState)
	am.keeper.InitGenesis(ctx, genesisState)
	return []abci.ValidatorUpdate{}
}

// ExportGenesis returns the exported genesis state as raw bytes for the ibc-router
// module.
func (am AppModule) ExportGenesis(ctx sdk.Context, cdc codec.JSONCodec) json.RawMessage {
	gs := am.keeper.ExportGenesis(ctx)
	return cdc.MustMarshalJSON(gs)
}

// ConsensusVersion implements AppModule/ConsensusVersion.
func (AppModule) ConsensusVersion() uint64 { return 1 }

// BeginBlock implements the AppModule interface
func (am AppModule) BeginBlock(ctx sdk.Context, req abci.RequestBeginBlock) {}

// EndBlock implements the AppModule interface
func (am AppModule) EndBlock(ctx sdk.Context, req abci.RequestEndBlock) []abci.ValidatorUpdate {
	return []abci.ValidatorUpdate{}
}

// AppModuleSimulation functions

// GenerateGenesisState creates a randomized GenState of the router module.
func (AppModule) GenerateGenesisState(simState *module.SimulationState) {}

// ProposalContents doesn't return any content functions for governance proposals.
func (AppModule) ProposalContents(_ module.SimulationState) []simtypes.WeightedProposalContent {
	return nil
}

// RandomizedParams creates randomized ibc-router param changes for the simulator.
func (AppModule) RandomizedParams(r *rand.Rand) []simtypes.ParamChange {
	return nil
}

// RegisterStoreDecoder registers a decoder for router module's types
func (am AppModule) RegisterStoreDecoder(sdr sdk.StoreDecoderRegistry) {}

// WeightedOperations returns the all the router module operations with their respective weights.
func (am AppModule) WeightedOperations(_ module.SimulationState) []simtypes.WeightedOperation {
	return nil
}

// ICS 30 callbacks

// OnChanOpenInit implements the IBCModule interface
func (am AppModule) OnChanOpenInit(ctx sdk.Context, order channeltypes.Order, connectionHops []string, portID string, channelID string, chanCap *capabilitytypes.Capability, counterparty channeltypes.Counterparty, version string) error {
	// call underlying app's (transfer) callback
	return am.app.OnChanOpenInit(ctx, order, connectionHops, portID, channelID,
		chanCap, counterparty, version)
}

// OnChanOpenTry implements the IBCModule interface
func (am AppModule) OnChanOpenTry(ctx sdk.Context, order channeltypes.Order, connectionHops []string, portID, channelID string, chanCap *capabilitytypes.Capability, counterparty channeltypes.Counterparty, version, counterpartyVersion string,
) error {
	// call underlying app's (transfer) callback
	return am.app.OnChanOpenTry(ctx, order, connectionHops, portID, channelID,
		chanCap, counterparty, version, counterpartyVersion)
}

// OnChanOpenAck implements the IBCModule interface
func (am AppModule) OnChanOpenAck(ctx sdk.Context, portID, channelID string, counterpartyVersion string) error {
	return am.app.OnChanOpenAck(ctx, portID, channelID, counterpartyVersion)
}

// OnChanOpenConfirm implements the IBCModule interface
func (am AppModule) OnChanOpenConfirm(ctx sdk.Context, portID, channelID string) error {
	// call underlying app's OnChanOpenConfirm callback.
	return am.app.OnChanOpenConfirm(ctx, portID, channelID)
}

// OnChanCloseInit implements the IBCModule interface
func (am AppModule) OnChanCloseInit(ctx sdk.Context, portID, channelID string) error {
	// TODO: Unescrow all remaining funds for unprocessed packets
	return am.app.OnChanCloseInit(ctx, portID, channelID)
}

// OnChanCloseConfirm implements the IBCModule interface
func (am AppModule) OnChanCloseConfirm(ctx sdk.Context, portID, channelID string) error {
	// TODO: Unescrow all remaining funds for unprocessed packets
	return am.app.OnChanCloseConfirm(ctx, portID, channelID)
}

// OnRecvPacket implements the IBCModule interface.
func (am AppModule) OnRecvPacket(ctx sdk.Context, packet channeltypes.Packet, relayer sdk.AccAddress) ibcexported.Acknowledgement {
	var data transfertypes.FungibleTokenPacketData
	if err := transfertypes.ModuleCdc.UnmarshalJSON(packet.GetData(), &data); err != nil {
		return channeltypes.NewErrorAcknowledgement("cannot unmarshal ICS-20 transfer packet data")
	}

	// parse out any forwarding info
	receiver, finalDest, port, channel, err := ParseIncomingTransferField(data.Receiver)
	switch {

	// if this isn't a packet to forward, just use the transfer module normally
	case finalDest == "" && port == "" && channel == "" && err == nil:
		return am.app.OnRecvPacket(ctx, packet, relayer)

	// If the parsing fails return a failure ack
	case err != nil:
		return channeltypes.NewErrorAcknowledgement("cannot parse packet fowrading information")

	// Otherwise we have a packet to forward
	default:
		// Modify packet data to process packet transfer for this chain, omitting forwarding info
		newData := data
		newData.Receiver = receiver.String()
		bz, err := transfertypes.ModuleCdc.MarshalJSON(&newData)
		if err != nil {
			return channeltypes.NewErrorAcknowledgement(err.Error())
		}
		newPacket := packet
		newPacket.Data = bz

		ack := am.app.OnRecvPacket(ctx, newPacket, relayer)
		if ack.Success() {
			// recalculate denom, skip checks that were already done in app.OnRecvPacket
			var denom string
			if transfertypes.ReceiverChainIsSource(packet.GetSourcePort(), packet.GetSourceChannel(), newData.Denom) {
				denom = transfertypes.ParseDenomTrace(newData.Denom).IBCDenom()
			} else {
				prefixedDenom := transfertypes.GetDenomPrefix(packet.GetDestPort(), packet.GetDestChannel()) + newData.Denom
				denom = transfertypes.ParseDenomTrace(prefixedDenom).IBCDenom()
			}
			var token = sdk.NewCoin(denom, sdk.NewIntFromUint64(newData.Amount))
			if err := am.keeper.ForwardTransferPacket(ctx, receiver, token, port, channel, finalDest, []metrics.Label{}); err != nil {
				ack = channeltypes.NewErrorAcknowledgement("failed to foward transfer packet")
			}
		}
		return ack
	}
}

// OnAcknowledgementPacket implements the IBCModule interface
func (am AppModule) OnAcknowledgementPacket(ctx sdk.Context, packet channeltypes.Packet, acknowledgement []byte, relayer sdk.AccAddress) error {
	return am.app.OnAcknowledgementPacket(ctx, packet, acknowledgement, relayer)
}

// OnTimeoutPacket implements the IBCModule interface
func (am AppModule) OnTimeoutPacket(ctx sdk.Context, packet channeltypes.Packet, relayer sdk.AccAddress) error {
	return am.app.OnTimeoutPacket(ctx, packet, relayer)
}

// For now this assumes one hop, should be better parsing
func ParseIncomingTransferField(receiverData string) (thischainaddr sdk.AccAddress, finaldestination, port, channel string, err error) {
	sep1 := strings.Split(receiverData, ":")
	switch {
	case len(sep1) == 1:
		thischainaddr, err = sdk.AccAddressFromBech32(receiverData)
		return
	case len(sep1) >= 2:
		finaldestination = strings.Join(sep1[:1], ":")
	default:
		return nil, "", "", "", fmt.Errorf("unparsable reciever field, need: '{address_on_this_chain}|{portid}/{channelid}:{final_dest_address}', got: '%s'", receiverData)
	}
	sep2 := strings.Split(sep1[0], "|")
	if len(sep2) != 2 {
		err = fmt.Errorf("formatting incorect, need: '{address_on_this_chain}|{portid}/{channelid}:{final_dest_address}', got: '%s'", receiverData)
		return
	}
	thischainaddr, err = sdk.AccAddressFromBech32(sep2[0])
	if err != nil {
		return
	}
	sep3 := strings.Split(sep2[1], "/")
	port = sep3[0]
	channel = sep3[1]
	return
}

// sending chain reviecer field
// cosmos16plylpsgxechajltx9yeseqexzdzut9g8vla4k|transfer/channel-0:cosmos16plylpsgxechajltx9yeseqexzdzut9g8vla4k|transfer/channel-0: cosmos16plylpsgxechajltx9yeseqexzdzut9g8vla4k|transfer/channel-0:cosmos16plylpsgxechajltx9yeseqexzdzut9g8vla4k

// first proxy chain reviever field
// cosmos16plylpsgxechajltx9yeseqexzdzut9g8vla4k|transfer/channel-0:cosmos16plylpsgxechajltx9yeseqexzdzut9g8vla4k|transfer/channel-0:cosmos16plylpsgxechajltx9yeseqexzdzut9g8vla4k

// second proxy chain reviever field
// cosmos16plylpsgxechajltx9yeseqexzdzut9g8vla4k|transfer/channel-0:cosmos16plylpsgxechajltx9yeseqexzdzut9g8vla4k

// final proxy chain reciever field
// cosmos16plylpsgxechajltx9yeseqexzdzut9g8vla4k
