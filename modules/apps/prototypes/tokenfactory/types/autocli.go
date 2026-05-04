package types

import (
	autocliv1 "cosmossdk.io/api/cosmos/autocli/v1"
)

// AutoCLIOptions implements the autocli.HasAutoCLIConfig interface.
func AutoCLIOptions() *autocliv1.ModuleOptions {
	return &autocliv1.ModuleOptions{
		Query: &autocliv1.ServiceCommandDescriptor{
			Service: Query_serviceDesc.ServiceName,
			RpcCommandOptions: []*autocliv1.RpcCommandOptions{
				{
					RpcMethod: "Params",
					Use:       "params",
					Short:     "Query the tokenfactory parameters",
				},
				{
					RpcMethod: "DenomAuthorityMetadata",
					Use:       "denom-authority-metadata [denom]",
					Short:     "Query the authority metadata for a specific denom",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "denom"},
					},
				},
				{
					RpcMethod: "DenomsByCreator",
					Use:       "denoms-by-creator [creator]",
					Short:     "Query all denoms created by a specific creator",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "creator"},
					},
				},
			},
		},
		Tx: &autocliv1.ServiceCommandDescriptor{
			Service:              Msg_serviceDesc.ServiceName,
			EnhanceCustomCommand: true,
			RpcCommandOptions: []*autocliv1.RpcCommandOptions{
				{
					RpcMethod: "CreateDenom",
					Use:       "create-denom [denom]",
					Short:     "Create a new tokenfactory denomination",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "denom"},
					},
				},
				{
					RpcMethod: "Mint",
					Skip:      true, // Custom CLI: Coin positional arg panics with AutoCLI's protov2 reflection
				},
				{
					RpcMethod: "Burn",
					Skip:      true, // Custom CLI: Coin positional arg panics with AutoCLI's protov2 reflection
				},
				{
					RpcMethod: "ChangeAdmin",
					Use:       "change-admin [denom] [new_admin]",
					Short:     "Transfer admin authority to a new address",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "denom"},
						{ProtoField: "new_admin"},
					},
				},
				{
					RpcMethod: "RenounceAdmin",
					Use:       "renounce-admin [denom]",
					Short:     "Permanently remove admin authority",
					Long:      "Permanently remove admin authority. After renouncing, no one can mint or burn via MsgMint/MsgBurn.",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "denom"},
					},
				},
			},
		},
	}
}
