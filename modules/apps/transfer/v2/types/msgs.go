package types

// GetCoins returns the tokens which will be transferred.
// If MsgTransfer is populated in the Token field, only that field
// will be returned in the coin array.
func (msg MsgTransfer) GetCoins() sdk.Coins {
	coins := msg.Tokens
	if isValidIBCCoin(msg.Token) {
		coins = []sdk.Coin{msg.Token}
	}
	return coins
}
