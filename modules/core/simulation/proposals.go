package simulation

import (
	"math/rand"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/address"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	"github.com/cosmos/cosmos-sdk/x/simulation"

	"github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	connectiontypes "github.com/cosmos/ibc-go/v8/modules/core/03-connection/types"
)

// Simulation operation weights constants
const (
	DefaultWeightMsgUpdateParams int = 100

	OpWeightMsgUpdateParams = "op_weight_msg_update_params" // #nosec
)

// ProposalMsgs defines the module weighted proposals' contents
func ProposalMsgs() []simtypes.WeightedProposalMsg {
	return []simtypes.WeightedProposalMsg{
		simulation.NewWeightedProposalMsg(
			OpWeightMsgUpdateParams,
			DefaultWeightMsgUpdateParams,
			SimulateClientMsgUpdateParams,
		),
		simulation.NewWeightedProposalMsg(
			OpWeightMsgUpdateParams,
			DefaultWeightMsgUpdateParams,
			SimulateConnectionMsgUpdateParams,
		),
	}
}

// SimulateClientMsgUpdateParams returns a random MsgUpdateParams for 02-client
func SimulateClientMsgUpdateParams(r *rand.Rand, _ sdk.Context, _ []simtypes.Account) sdk.Msg {
	var signer sdk.AccAddress = address.Module("gov")
	params := types.DefaultParams()
	params.AllowedClients = []string{"06-solomachine", "07-tendermint"}

	return &types.MsgUpdateParams{
		Signer: signer.String(),
		Params: params,
	}
}

// SimulateConnectionMsgUpdateParams returns a random MsgUpdateParams 03-connection
func SimulateConnectionMsgUpdateParams(r *rand.Rand, _ sdk.Context, _ []simtypes.Account) sdk.Msg {
	var signer sdk.AccAddress = address.Module("gov")
	params := connectiontypes.DefaultParams()
	params.MaxExpectedTimePerBlock = uint64(100)

	return &connectiontypes.MsgUpdateParams{
		Signer: signer.String(),
		Params: params,
	}
}
