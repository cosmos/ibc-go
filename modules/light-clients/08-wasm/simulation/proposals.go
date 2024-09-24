package simulation

import (
	"math/rand"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/address"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	"github.com/cosmos/cosmos-sdk/x/simulation"

	"github.com/cosmos/ibc-go/modules/light-clients/08-wasm/types"
)

// Simulation operation weights constants
const (
	DefaultWeightMsgStoreCode int = 100

	OpWeightMsgStoreCode = "op_weight_msg_store_code" // #nosec
)

// ProposalMsgs defines the module weighted proposals' contents
func ProposalMsgs() []simtypes.WeightedProposalMsg {
	return []simtypes.WeightedProposalMsg{
		simulation.NewWeightedProposalMsg(
			OpWeightMsgStoreCode,
			DefaultWeightMsgStoreCode,
			SimulateMsgStoreCode,
		),
	}
}

// SimulateMsgStoreCode returns a random MsgStoreCode for the 08-wasm module
func SimulateMsgStoreCode(r *rand.Rand, _ sdk.Context, _ []simtypes.Account) sdk.Msg {
	var signer sdk.AccAddress = address.Module("gov")

	return &types.MsgStoreCode{
		Signer:       signer.String(),
		WasmByteCode: []byte{0x01},
	}
}
