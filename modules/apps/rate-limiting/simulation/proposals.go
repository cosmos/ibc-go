package simulation

import (
	"math/rand"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/address"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	"github.com/cosmos/cosmos-sdk/x/simulation"

	"github.com/cosmos/ibc-go/v10/modules/apps/rate-limiting/types"
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
			SimulateMsgUpdateParams,
		),
	}
}

// SimulateMsgUpdateParams returns a MsgUpdateParams
func SimulateMsgUpdateParams(r *rand.Rand, _ sdk.Context, _ []simtypes.Account) sdk.Msg {
	// Use the module account as the signer
	var gov sdk.AccAddress = address.Module("gov")

	// Enable or disable module with probability
	enabled := RandomEnabled(r)

	// Set random max inflow and outflow values
	defaultMaxOutflow := RandomMaxValue(r, 100000, 10000000)
	defaultMaxInflow := RandomMaxValue(r, 100000, 10000000)

	// Set random period
	defaultPeriod := RandomPeriod(r)

	// Create the parameter update message
	return types.NewMsgUpdateParams(
		gov.String(),
		types.Params{
			Enabled:           enabled,
			DefaultMaxOutflow: defaultMaxOutflow,
			DefaultMaxInflow:  defaultMaxInflow,
			DefaultPeriod:     defaultPeriod,
		},
	)
}
