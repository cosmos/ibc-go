package types_test

import (
	"testing"

	"github.com/cosmos/ibc-go/v9/modules/core/02-client/v2/types"
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
				CounterpartyInfos: []types.CounterpartyInfo{
					types.NewCounterpartyInfo([][]byte{{01}}, "test-0"),
					types.NewCounterpartyInfo([][]byte{{01}}, "test-1"),
					types.NewCounterpartyInfo([][]byte{{01}}, "test-2"),
					types.NewCounterpartyInfo([][]byte{{01}}, "test-3"),
				},
			},
			wantErr: false,
		},
		{
			name: "invalid - duplicate client IDs",
			genState: types.GenesisState{
				CounterpartyInfos: []types.CounterpartyInfo{
					types.NewCounterpartyInfo([][]byte{{01}}, "test-0"), // test-0 ID duplicated
					types.NewCounterpartyInfo([][]byte{{01}}, "test-0"),
				},
			},
			wantErr: true,
		},
		{
			name: "invalid - invalid client ID",
			genState: types.GenesisState{
				CounterpartyInfos: []types.CounterpartyInfo{
					types.NewCounterpartyInfo([][]byte{{01}}, ""), // empty client ID
					types.NewCounterpartyInfo([][]byte{{01}}, "test-1"),
					types.NewCounterpartyInfo([][]byte{{01}}, "test-2"),
					types.NewCounterpartyInfo([][]byte{{01}}, "test-3"),
				},
			},
			wantErr: true,
		},
		{
			name: "invalid - invalid merkle prefix",
			genState: types.GenesisState{
				CounterpartyInfos: []types.CounterpartyInfo{
					types.NewCounterpartyInfo(nil, "test-0"), // nil prefix
					types.NewCounterpartyInfo([][]byte{{01}}, "test-1"),
					types.NewCounterpartyInfo([][]byte{{01}}, "test-2"),
					types.NewCounterpartyInfo([][]byte{{01}}, "test-3"),
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
