package avalanche_test

import (
	"github.com/ava-labs/avalanchego/utils/crypto/bls"
	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	ibcava "github.com/cosmos/ibc-go/v8/modules/light-clients/14-avalanche"
)

func (suite *AvalancheTestSuite) TestHeaderValidateBasic() {
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

	var header *ibcava.Header
	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{"valid header", func() {}, true},
		{"PchainHeader is nil", func() {
			header.PchainHeader = nil
		}, false},
		{"SubnetHeader header is nil", func() {
			header.SubnetHeader = nil
		}, false},
		{"PrevSubnetHeader header is nil", func() {
			header.PrevSubnetHeader = nil
		}, false},
		{"trusted height is equal to header height", func() {
			header.SubnetHeader = header.PrevSubnetHeader
		}, false},
		{"ValidatorSet set nil", func() {
			header.ValidatorSet = nil
		}, false},
		{"validator set nil", func() {
			header.Vdrs = nil
		}, false},
		{"StorageRoot set nil", func() {
			header.StorageRoot = nil
		}, false},
		{"SignersInput set nil", func() {
			header.SignersInput = nil
		}, false},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupTest()

			header = &ibcava.Header{
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
				Vdrs:              []*ibcava.Validator{vdrs[0], vdrs[1], vdrs[2]},
				ValidatorSet:      []byte("ValidatorSet"),
				StorageRoot:       []byte("StorageRoot"),
				SignedStorageRoot: []byte("SignedStorageRoot"),
				SignedValidatorSet: []byte("SignedValidatorSet"),
				SignersInput: []byte("SignersInput"),
			}

			tc.malleate()

			err := header.ValidateBasic()

			if tc.expPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}
