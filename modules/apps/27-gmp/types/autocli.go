package types

import (
	autocliv1 "cosmossdk.io/api/cosmos/autocli/v1"
)

// AutoCLIOptions implements the autocli.HasAutoCLIConfig interface.
func AutoCLIOptions() *autocliv1.ModuleOptions {
	return &autocliv1.ModuleOptions{
		Query: &autocliv1.ServiceCommandDescriptor{
			Service: _Query_serviceDesc.ServiceName,
			RpcCommandOptions: []*autocliv1.RpcCommandOptions{
				{
					RpcMethod: "AccountAddress",
					Use:       "get-address [client_id] [sender] [salt]",
					Short:     "Get or pre-compute the address of an ICS27 GMP account",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "client_id"},
						{ProtoField: "sender"},
						{ProtoField: "salt"},
					},
				},
			},
		},
		Tx: &autocliv1.ServiceCommandDescriptor{
			Service: _Msg_serviceDesc.ServiceName,
			RpcCommandOptions: []*autocliv1.RpcCommandOptions{
				{
					RpcMethod: "SendCall",
					Use:       "send-call [source_client] [sender] [receiver] [salt] [payload] [timeout_timestamp] [memo] [encoding]",
					Short:     "Send a call to an ICS27 GMP account",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "source_client"},
						{ProtoField: "sender"},
						{ProtoField: "receiver"},
						{ProtoField: "salt"},
						{ProtoField: "payload"},
						{ProtoField: "timeout_timestamp"},
						{ProtoField: "memo"},
						{ProtoField: "encoding", Optional: true},
					},
				},
			},
		},
	}
}
