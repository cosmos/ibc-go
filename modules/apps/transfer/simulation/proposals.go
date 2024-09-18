package simulation

import (
	"math/rand"

	"cosmossdk.io/core/address"
	sdk "github.com/cosmos/cosmos-sdk/types"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/cosmos-sdk/x/simulation"

	"github.com/cosmos/ibc-go/v9/modules/apps/transfer/types"
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
func SimulateMsgUpdateParams(_ *rand.Rand, _ []simtypes.Account, _ address.Codec) (sdk.Msg, error) {
	var gov sdk.AccAddress = authtypes.NewModuleAddress("gov")
	params := types.DefaultParams()
	params.SendEnabled = false

	return &types.MsgUpdateParams{
		Signer: gov.String(),
		Params: params,
	}, nil
}
