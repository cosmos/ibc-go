package simulation

import (
	"math/rand"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/address"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	"github.com/cosmos/cosmos-sdk/x/simulation"

	"github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts/host/types"
	controllertypes "github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts/controller/types"
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
			SimulateHostMsgUpdateParams,
		),
		simulation.NewWeightedProposalMsg(
			OpWeightMsgUpdateParams,
			DefaultWeightMsgUpdateParams,
			SimulateControllerMsgUpdateParams,
		),
	}
}

// SimulateHostMsgUpdateParams returns a random MsgUpdateParams for the host module
func SimulateHostMsgUpdateParams(r *rand.Rand, _ sdk.Context, _ []simtypes.Account) sdk.Msg {
	var authority sdk.AccAddress = address.Module("gov")
	params := types.DefaultParams()
	params.HostEnabled = false

	return &types.MsgUpdateParams{
		Authority: authority.String(),
		Params:    params,
	}
}

// SimulateControllerMsgUpdateParams returns a random MsgUpdateParams for the controller module
func SimulateControllerMsgUpdateParams(r *rand.Rand, _ sdk.Context, _ []simtypes.Account) sdk.Msg {
	var authority sdk.AccAddress = address.Module("gov")
	params := controllertypes.DefaultParams()
	params.ControllerEnabled = false

	return &controllertypes.MsgUpdateParams{
		Authority: authority.String(),
		Params:    params,
	}
}
