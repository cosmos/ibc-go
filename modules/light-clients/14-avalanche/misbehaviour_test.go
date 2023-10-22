package avalanche_test

import (
	"github.com/ava-labs/avalanchego/utils/crypto/bls"

	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	ibcava "github.com/cosmos/ibc-go/v8/modules/light-clients/14-avalanche"
)

func (suite *AvalancheTestSuite) TestMisbehaviourValidateBasic() {
	testVdrs = []*testValidator{
		newTestValidator(),
		newTestValidator(),
		newTestValidator(),
	}

	vdrs := []*ibcava.Validator{
		{
			NodeIDs:       [][]byte{testVdrs[0].nodeID.Bytes()},
			PublicKeyByte: bls.PublicKeyToBytes(testVdrs[0].vdr.PublicKey),
			Weight:        testVdrs[0].vdr.Weight,
			EndTime:       suite.chainA.GetContext().BlockTime().Add(900000000000000),
		},
		{
			NodeIDs:       [][]byte{testVdrs[1].nodeID.Bytes()},
			PublicKeyByte: bls.PublicKeyToBytes(testVdrs[1].vdr.PublicKey),
			Weight:        testVdrs[1].vdr.Weight,
			EndTime:       suite.chainA.GetContext().BlockTime().Add(900000000000000),
		},
		{
			NodeIDs:       [][]byte{testVdrs[2].nodeID.Bytes()},
			PublicKeyByte: bls.PublicKeyToBytes(testVdrs[2].vdr.PublicKey),
			Weight:        testVdrs[2].vdr.Weight,
			EndTime:       suite.chainA.GetContext().BlockTime().Add(900000000000000),
		},
	}

	header2 := &ibcava.Header{
		SubnetHeader: &ibcava.SubnetHeader{
			Height:     &clienttypes.Height{RevisionNumber: 2, RevisionHeight: 2},
			Timestamp:  suite.chainA.GetContext().BlockTime(),
			BlockHash:  []byte("SubnetHeaderBlockHash"),
			PchainVdrs: []*ibcava.Validator{vdrs[0], vdrs[1], vdrs[2]},
		},
		PrevSubnetHeader: &ibcava.SubnetHeader{
			Height:     &clienttypes.Height{RevisionNumber: 2, RevisionHeight: 1},
			Timestamp:  suite.chainA.GetContext().BlockTime(),
			BlockHash:  []byte("SubnetHeaderBlockHash"),
			PchainVdrs: []*ibcava.Validator{vdrs[0], vdrs[1], vdrs[2]},
		},
		PchainHeader: &ibcava.PchainHeader{
			Height:    &clienttypes.Height{RevisionNumber: 3, RevisionHeight: 3},
			Timestamp: suite.chainA.GetContext().BlockTime(),
			BlockHash: []byte("PchainHeaderBlockHash"),
		},
		Vdrs: []*ibcava.Validator{vdrs[0], vdrs[1], vdrs[2]},
		ValidatorSet:      []byte("ValidatorSet"),
		StorageRoot:       []byte("StorageRoot"),
		SignedStorageRoot: []byte("SignedStorageRoot"),
		SignedValidatorSet: []byte("SignedValidatorSet"),
		SignersInput: []byte("SignersInput"),
	}

	header1 := &ibcava.Header{
		SubnetHeader: &ibcava.SubnetHeader{
			Height:     &clienttypes.Height{RevisionNumber: 2, RevisionHeight: 3},
			Timestamp:  suite.chainA.GetContext().BlockTime().Add(100),
			BlockHash:  []byte("SubnetHeaderBlockHash"),
			PchainVdrs: []*ibcava.Validator{vdrs[0], vdrs[1], vdrs[2]},
		},
		PrevSubnetHeader: &ibcava.SubnetHeader{
			Height:     &clienttypes.Height{RevisionNumber: 2, RevisionHeight: 2},
			Timestamp:  suite.chainA.GetContext().BlockTime(),
			BlockHash:  []byte("SubnetHeaderBlockHash"),
			PchainVdrs: []*ibcava.Validator{vdrs[0], vdrs[1], vdrs[2]},
		},
		PchainHeader: &ibcava.PchainHeader{
			Height:    &clienttypes.Height{RevisionNumber: 3, RevisionHeight: 4},
			Timestamp: suite.chainA.GetContext().BlockTime(),
			BlockHash: []byte("PchainHeaderBlockHash"),
		},
		Vdrs: []*ibcava.Validator{vdrs[0], vdrs[1], vdrs[2]},
		ValidatorSet:      []byte("ValidatorSet"),
		StorageRoot:       []byte("StorageRoot"),
		SignedStorageRoot: []byte("SignedStorageRoot"),
		SignedValidatorSet: []byte("SignedValidatorSet"),
		SignersInput: []byte("SignersInput"),
	}

	testCases := []struct {
		name                 string
		misbehaviour         *ibcava.Misbehaviour
		malleateMisbehaviour func(misbehaviour *ibcava.Misbehaviour) error
		expPass              bool
	}{
		{
			"valid fork misbehaviour, two headers at same height have different time",
			&ibcava.Misbehaviour{
				Header1: header1,
				Header2: header2,
			},
			func(misbehaviour *ibcava.Misbehaviour) error { return nil },
			true,
		},
		{
			"misbehaviour Header1 is nil",
			ibcava.NewMisbehaviour(nil, header2),
			func(m *ibcava.Misbehaviour) error { return nil },
			false,
		},
		{
			"misbehaviour Header2 is nil",
			ibcava.NewMisbehaviour(header1, nil),
			func(m *ibcava.Misbehaviour) error { return nil },
			false,
		},
		{
			"trusted height is 0 in Header1",
			&ibcava.Misbehaviour{
				Header1: &ibcava.Header{
					SubnetHeader: &ibcava.SubnetHeader{
						Height:     &clienttypes.Height{RevisionNumber: 2, RevisionHeight: 0},
						Timestamp:  suite.chainA.GetContext().BlockTime(),
						BlockHash:  []byte("SubnetHeaderBlockHash"),
						PchainVdrs: []*ibcava.Validator{vdrs[0], vdrs[1], vdrs[2]},
					},
					PrevSubnetHeader: &ibcava.SubnetHeader{
						Height:     &clienttypes.Height{RevisionNumber: 2, RevisionHeight: 1},
						Timestamp:  suite.chainA.GetContext().BlockTime(),
						BlockHash:  []byte("SubnetHeaderBlockHash"),
						PchainVdrs: []*ibcava.Validator{vdrs[0], vdrs[1], vdrs[2]},
					},
					PchainHeader: &ibcava.PchainHeader{
						Height:    &clienttypes.Height{RevisionNumber: 3, RevisionHeight: 3},
						Timestamp: suite.chainA.GetContext().BlockTime(),
						BlockHash: []byte("PchainHeaderBlockHash"),
					},
					Vdrs: []*ibcava.Validator{vdrs[0], vdrs[1], vdrs[2]},
					ValidatorSet:      []byte("ValidatorSet"),
					StorageRoot:       []byte("StorageRoot"),
					SignedStorageRoot: []byte("SignedStorageRoot"),
					SignedValidatorSet: []byte("SignedValidatorSet"),
					SignersInput: []byte("SignersInput"),
				},
				Header2: header2,
			},
			func(misbehaviour *ibcava.Misbehaviour) error { return nil },
			false,
		},
		{
			"trusted height is 0 in Header2",
			&ibcava.Misbehaviour{
				Header1: header1,
				Header2: &ibcava.Header{
					SubnetHeader: &ibcava.SubnetHeader{
						Height:     &clienttypes.Height{RevisionNumber: 2, RevisionHeight: 0},
						Timestamp:  suite.chainA.GetContext().BlockTime(),
						BlockHash:  []byte("SubnetHeaderBlockHash"),
						PchainVdrs: []*ibcava.Validator{vdrs[0], vdrs[1], vdrs[2]},
					},
					PrevSubnetHeader: &ibcava.SubnetHeader{
						Height:     &clienttypes.Height{RevisionNumber: 2, RevisionHeight: 2},
						Timestamp:  suite.chainA.GetContext().BlockTime(),
						BlockHash:  []byte("SubnetHeaderBlockHash"),
						PchainVdrs: []*ibcava.Validator{vdrs[0], vdrs[1], vdrs[2]},
					},
					PchainHeader: &ibcava.PchainHeader{
						Height:    &clienttypes.Height{RevisionNumber: 3, RevisionHeight: 4},
						Timestamp: suite.chainA.GetContext().BlockTime(),
						BlockHash: []byte("PchainHeaderBlockHash"),
					},
					Vdrs: []*ibcava.Validator{vdrs[0], vdrs[1], vdrs[2]},
					ValidatorSet:      []byte("ValidatorSet"),
					StorageRoot:       []byte("StorageRoot"),
					SignedStorageRoot: []byte("SignedStorageRoot"),
					SignedValidatorSet: []byte("SignedValidatorSet"),
					SignersInput: []byte("SignersInput"),
				},
			},
			func(misbehaviour *ibcava.Misbehaviour) error { return nil },
			false,
		},
		{
			"trusted valset is nil in Header1",
			&ibcava.Misbehaviour{
				Header1: &ibcava.Header{
					SubnetHeader: &ibcava.SubnetHeader{
						Height:     &clienttypes.Height{RevisionNumber: 2, RevisionHeight: 2},
						Timestamp:  suite.chainA.GetContext().BlockTime(),
						BlockHash:  []byte("SubnetHeaderBlockHash"),
						PchainVdrs: []*ibcava.Validator{vdrs[0], vdrs[1], vdrs[2]},
					},
					PrevSubnetHeader: &ibcava.SubnetHeader{
						Height:     &clienttypes.Height{RevisionNumber: 2, RevisionHeight: 1},
						Timestamp:  suite.chainA.GetContext().BlockTime(),
						BlockHash:  []byte("SubnetHeaderBlockHash"),
						PchainVdrs: []*ibcava.Validator{vdrs[0], vdrs[1], vdrs[2]},
					},
					PchainHeader: &ibcava.PchainHeader{
						Height:    &clienttypes.Height{RevisionNumber: 3, RevisionHeight: 3},
						Timestamp: suite.chainA.GetContext().BlockTime(),
						BlockHash: []byte("PchainHeaderBlockHash"),
					},
					Vdrs: []*ibcava.Validator{},
					ValidatorSet:      []byte("ValidatorSet"),
					StorageRoot:       []byte("StorageRoot"),
					SignedStorageRoot: []byte("SignedStorageRoot"),
					SignedValidatorSet: []byte("SignedValidatorSet"),
					SignersInput: []byte("SignersInput"),
				},
				Header2: header2,
			},
			func(misbehaviour *ibcava.Misbehaviour) error { return nil },
			false,
		},
		{
			"trusted valset is nil in Header2",
			&ibcava.Misbehaviour{
				Header1: header1,
				Header2: &ibcava.Header{
					SubnetHeader: &ibcava.SubnetHeader{
						Height:     &clienttypes.Height{RevisionNumber: 2, RevisionHeight: 3},
						Timestamp:  suite.chainA.GetContext().BlockTime().Add(100),
						BlockHash:  []byte("SubnetHeaderBlockHash"),
						PchainVdrs: []*ibcava.Validator{vdrs[0], vdrs[1], vdrs[2]},
					},
					PrevSubnetHeader: &ibcava.SubnetHeader{
						Height:     &clienttypes.Height{RevisionNumber: 2, RevisionHeight: 2},
						Timestamp:  suite.chainA.GetContext().BlockTime(),
						BlockHash:  []byte("SubnetHeaderBlockHash"),
						PchainVdrs: []*ibcava.Validator{vdrs[0], vdrs[1], vdrs[2]},
					},
					PchainHeader: &ibcava.PchainHeader{
						Height:    &clienttypes.Height{RevisionNumber: 3, RevisionHeight: 4},
						Timestamp: suite.chainA.GetContext().BlockTime(),
						BlockHash: []byte("PchainHeaderBlockHash"),
					},
					Vdrs: []*ibcava.Validator{},
					ValidatorSet:      []byte("ValidatorSet"),
					StorageRoot:       []byte("StorageRoot"),
					SignedStorageRoot: []byte("SignedStorageRoot"),
					SignedValidatorSet: []byte("SignedValidatorSet"),
					SignersInput: []byte("SignersInput"),
				},
			},
			func(misbehaviour *ibcava.Misbehaviour) error { return nil },
			false,
		},
		{
			"header2 height is greater",
			&ibcava.Misbehaviour{
				Header1: header1,
				Header2: &ibcava.Header{
					SubnetHeader: &ibcava.SubnetHeader{
						Height:     &clienttypes.Height{RevisionNumber: 2, RevisionHeight: 5},
						Timestamp:  suite.chainA.GetContext().BlockTime().Add(100),
						BlockHash:  []byte("SubnetHeaderBlockHash"),
						PchainVdrs: []*ibcava.Validator{vdrs[0], vdrs[1], vdrs[2]},
					},
					PrevSubnetHeader: &ibcava.SubnetHeader{
						Height:     &clienttypes.Height{RevisionNumber: 2, RevisionHeight: 2},
						Timestamp:  suite.chainA.GetContext().BlockTime(),
						BlockHash:  []byte("SubnetHeaderBlockHash"),
						PchainVdrs: []*ibcava.Validator{vdrs[0], vdrs[1], vdrs[2]},
					},
					PchainHeader: &ibcava.PchainHeader{
						Height:    &clienttypes.Height{RevisionNumber: 3, RevisionHeight: 4},
						Timestamp: suite.chainA.GetContext().BlockTime(),
						BlockHash: []byte("PchainHeaderBlockHash"),
					},
					Vdrs: []*ibcava.Validator{vdrs[0], vdrs[1], vdrs[2]},
					ValidatorSet:      []byte("ValidatorSet"),
					StorageRoot:       []byte("StorageRoot"),
					SignedStorageRoot: []byte("SignedStorageRoot"),
					SignedValidatorSet: []byte("SignedValidatorSet"),
					SignersInput: []byte("SignersInput"),
				},
			},
			func(misbehaviour *ibcava.Misbehaviour) error { return nil },
			false,
		},
	}

	for i, tc := range testCases {
		tc := tc

		err := tc.malleateMisbehaviour(tc.misbehaviour)
		suite.Require().NoError(err)

		if tc.expPass {
			suite.Require().NoError(tc.misbehaviour.ValidateBasic(), "valid test case %d failed: %s", i, tc.name)
		} else {
			suite.Require().Error(tc.misbehaviour.ValidateBasic(), "invalid test case %d passed: %s", i, tc.name)
		}
	}
}
