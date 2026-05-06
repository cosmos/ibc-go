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
					Short:     "Query the IFT module parameters",
				},
				{
					RpcMethod: "IFTBridge",
					Use:       "bridge [denom] [client_id]",
					Short:     "Query an IFT bridge by denom and client-id",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "denom"},
						{ProtoField: "client_id"},
					},
				},
				{
					RpcMethod: "IFTBridges",
					Use:       "bridges",
					Short:     "Query all IFT bridges",
				},
				{
					RpcMethod: "IFTBridgesByDenom",
					Use:       "bridges-by-denom [denom]",
					Short:     "Query all IFT bridges for a specific denom",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "denom"},
					},
				},
				{
					RpcMethod: "PendingTransfer",
					Use:       "pending-transfer [denom] [client_id] [sequence]",
					Short:     "Query a pending IFT transfer",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "denom"},
						{ProtoField: "client_id"},
						{ProtoField: "sequence"},
					},
				},
			},
		},
		Tx: &autocliv1.ServiceCommandDescriptor{
			Service: Msg_serviceDesc.ServiceName,
			RpcCommandOptions: []*autocliv1.RpcCommandOptions{
				{
					RpcMethod: "RegisterIFTBridge",
					Use:       "register-bridge [denom] [client_id] [counterparty_ift_address] [ift_send_call_constructor]",
					Short:     "Register a new IFT bridge",
					Long:      "Register a new IFT bridge to a counterparty chain. Constructor type must be either \"evm\" or \"cosmostx\".",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "denom"},
						{ProtoField: "client_id"},
						{ProtoField: "counterparty_ift_address"},
						{ProtoField: "ift_send_call_constructor"},
					},
				},
				{
					RpcMethod: "RemoveIFTBridge",
					Use:       "remove-bridge [denom] [client_id]",
					Short:     "Remove an existing IFT bridge",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "denom"},
						{ProtoField: "client_id"},
					},
				},
				{
					RpcMethod: "IFTTransfer",
					Use:       "transfer [denom] [client_id] [receiver] [amount] [timeout_timestamp]",
					Short:     "Initiate a cross-chain IFT transfer",
					Long:      "Initiate a cross-chain token transfer via IFT. Timeout timestamp is in seconds since epoch.",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "denom"},
						{ProtoField: "client_id"},
						{ProtoField: "receiver"},
						{ProtoField: "amount"},
						{ProtoField: "timeout_timestamp"},
					},
				},
				{
					RpcMethod: "IFTMint",
					Skip:      true, // Internal: only callable by ICS27-GMP interchain accounts
				},
				{
					RpcMethod: "UpdateParams",
					Skip:      true, // Governance-only: restricted to module authority
				},
			},
		},
	}
}
