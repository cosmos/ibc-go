package types_test

import (
	"testing"

	"github.com/cosmos/ibc-go/v10/modules/core/02-client/v2/types"
)

func TestGenesisState_Validate(t *testing.T) {
	tests := []struct {
		name     string
		genState types.GenesisState
		wantErr  bool
	}{
		{
			name:     "default genesis",
			genState: types.DefaultGenesisState(),
			wantErr:  false,
		},
		{
			name: "valid genesis",
			genState: types.GenesisState{
				CounterpartyInfos: []types.GenesisCounterpartyInfo{
					{
						ClientId:         "test-1",
						CounterpartyInfo: types.NewCounterpartyInfo([][]byte{{0o1}}, "test-0"),
					},
					{
						ClientId:         "test-0",
						CounterpartyInfo: types.NewCounterpartyInfo([][]byte{{0o1}}, "test-1"),
					},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid - duplicate client IDs",
			genState: types.GenesisState{
				CounterpartyInfos: []types.GenesisCounterpartyInfo{
					{
						ClientId:         "test-1",
						CounterpartyInfo: types.NewCounterpartyInfo([][]byte{{0o1}}, "test-0"),
					},
					{
						ClientId:         "test-1",
						CounterpartyInfo: types.NewCounterpartyInfo([][]byte{{0o1}}, "test-0"),
					},
				},
			},
			wantErr: true,
		},
		{
			name: "client has itself as counterparty info",
			genState: types.GenesisState{
				CounterpartyInfos: []types.GenesisCounterpartyInfo{
					{
						ClientId:         "test-1",
						CounterpartyInfo: types.NewCounterpartyInfo([][]byte{{0o1}}, "test-1"),
					},
					{
						ClientId:         "test-0",
						CounterpartyInfo: types.NewCounterpartyInfo([][]byte{{0o1}}, "test-1"),
					},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid - invalid client ID",
			genState: types.GenesisState{
				CounterpartyInfos: []types.GenesisCounterpartyInfo{
					{
						ClientId:         "",
						CounterpartyInfo: types.NewCounterpartyInfo([][]byte{{0o1}}, "test-0"),
					},
					{
						ClientId:         "test-0",
						CounterpartyInfo: types.NewCounterpartyInfo([][]byte{{0o1}}, "test-1"),
					},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid - invalid merkle prefix",
			genState: types.GenesisState{
				CounterpartyInfos: []types.GenesisCounterpartyInfo{
					{
						ClientId:         "test-1",
						CounterpartyInfo: types.NewCounterpartyInfo(nil, "test-0"),
					},
					{
						ClientId:         "test-0",
						CounterpartyInfo: types.NewCounterpartyInfo([][]byte{{0o1}}, "test-1"),
					},
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.genState.Validate(); (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
