package callbacks

import (
	"encoding/json"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/ibc-go/v7/modules/apps/transfer/types"
	channeltypes "github.com/cosmos/ibc-go/v7/modules/core/04-channel/types"
)

// MyICS20Payload my random payload example
type MyICS20Payload struct {
	SrcCallbackAddress string `json:"src_callback_address"`
	DstCallbackAddress string `json:"dst_callback_address"`
}

var _ Decoder[MyICS20Payload] = &MyICS20Decoder{}

type MyICS20Decoder struct { //nolint:gofumpt
}

func (m MyICS20Decoder) Decode(packet channeltypes.Packet) (*MyICS20Payload, error) {
	var data types.FungibleTokenPacketData
	if err := types.ModuleCdc.UnmarshalJSON(packet.GetData(), &data); err != nil {
		return nil, err
	}
	// do we need a sanity check that type was unpacked proper? like denom not empty

	if data.Memo == "" {
		return nil, nil
	}
	// unmarshal json to type, I prefer using concrete type here:
	var r MyICS20Payload
	if err := json.Unmarshal([]byte(data.Memo), &r); err != nil {
		return nil, err
	}
	// todo: check that contains valid data
	return &r, nil
}

var _ Executor[MyICS20Payload] = &MyICS20Callbacks{}

type MyICS20Callbacks struct { //nolint:gofumpt
}

func (m MyICS20Callbacks) OnRecvPacket(ctx sdk.Context, obj MyICS20Payload, relayer sdk.AccAddress) error {
	ctx.Logger().Info("OnRecvPacket executed")
	return nil
}

func (m MyICS20Callbacks) OnAcknowledgementPacket(ctx sdk.Context, obj MyICS20Payload, acknowledgement []byte, relayer sdk.AccAddress) error {
	ctx.Logger().Info("OnAcknowledgementPacket executed")
	return nil
}

func (m MyICS20Callbacks) OnTimeoutPacket(ctx sdk.Context, obj MyICS20Payload, relayer sdk.AccAddress) error {
	ctx.Logger().Info("OnTimeoutPacket executed")
	return nil
}
