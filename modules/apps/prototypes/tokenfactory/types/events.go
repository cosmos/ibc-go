package types

const (
	// Event types
	TypeEvtCreateDenom   = "tokenfactory_create_denom"
	TypeEvtMint          = "tokenfactory_mint"
	TypeEvtBurn          = "tokenfactory_burn"
	TypeEvtChangeAdmin   = "tokenfactory_change_admin"
	TypeEvtRenounceAdmin = "tokenfactory_renounce_admin"

	// Attribute keys
	AttributeKeyDenom    = "denom"
	AttributeKeyAdmin    = "admin"
	AttributeKeyNewAdmin = "new_admin"
	AttributeKeyMintTo   = "mint_to"
	AttributeKeyBurnFrom = "burn_from"
	AttributeKeyAmount   = "amount"
)
