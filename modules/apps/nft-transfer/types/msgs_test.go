package types

import (
	"testing"

	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	clienttypes "github.com/cosmos/ibc-go/v3/modules/core/02-client/types"
)

var (
	sender   = sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address()).String()
	receiver = sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address()).String()
)

func TestMsgTransfer_ValidateBasic(t *testing.T) {
	tests := []struct {
		name    string
		msg     *MsgTransfer
		wantErr bool
	}{
		{"valid msg", NewMsgTransfer("nft-transfer", "channel-1", "cryptoCat", []string{"kitty"}, sender, receiver, clienttypes.NewHeight(1, 1), 1), false},
		{"invalid msg with port", NewMsgTransfer("@nft-transfer", "channel-1", "cryptoCat", []string{"kitty"}, sender, receiver, clienttypes.NewHeight(1, 1), 1), true},
		{"invalid msg with channel", NewMsgTransfer("nft-transfer", "@channel-1", "cryptoCat", []string{"kitty"}, sender, receiver, clienttypes.NewHeight(1, 1), 1), true},
		{"invalid msg with class", NewMsgTransfer("nft-transfer", "channel-1", "", []string{"kitty"}, sender, receiver, clienttypes.NewHeight(1, 1), 1), true},
		{"invalid msg with token_id", NewMsgTransfer("nft-transfer", "channel-1", "cryptoCat", []string{""}, sender, receiver, clienttypes.NewHeight(1, 1), 1), true},
		{"invalid msg with sender", NewMsgTransfer("nft-transfer", "channel-1", "cryptoCat", []string{"kitty"}, "", receiver, clienttypes.NewHeight(1, 1), 1), true},
		{"invalid msg with receiver", NewMsgTransfer("nft-transfer", "channel-1", "cryptoCat", []string{"kitty"}, sender, "", clienttypes.NewHeight(1, 1), 1), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.msg.ValidateBasic(); (err != nil) != tt.wantErr {
				t.Errorf("MsgTransfer.ValidateBasic() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
