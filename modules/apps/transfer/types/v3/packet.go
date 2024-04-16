package v3

// NewFungibleTokenPacketData constructs a new FungibleTokenPacketData instance
func NewFungibleTokenPacketData(
	tokens []*Token,
	sender, receiver string,
	memo string,
) FungibleTokenPacketData {
	return FungibleTokenPacketData{
		Tokens:   tokens,
		Sender:   sender,
		Receiver: receiver,
		Memo:     memo,
	}
}
