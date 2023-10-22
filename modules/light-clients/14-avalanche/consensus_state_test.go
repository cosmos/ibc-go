package avalanche_test

import (
	"time"

	ibcava "github.com/cosmos/ibc-go/v8/modules/light-clients/14-avalanche"
)

func (suite *AvalancheTestSuite) TestConsensusStateValidateBasic() {
	storageRoot := []byte("StorageRoot")
	validatorSet := []byte("ValidatorSet")
	signersInput := []byte("signersInput")
	signedStorageRoot := ibcava.SetSignature(storageRoot)
	signedValidatorSet := ibcava.SetSignature(validatorSet)
	testCases := []struct {
		msg            string
		consensusState *ibcava.ConsensusState
		expectPass     bool
	}{
		{
			"success",
			&ibcava.ConsensusState{
				Timestamp:          suite.now,
				StorageRoot:        storageRoot,
				SignedStorageRoot:  signedStorageRoot[:],
				ValidatorSet:       validatorSet,
				SignedValidatorSet: signedValidatorSet[:],
				Vdrs:               []*ibcava.Validator{&ibcava.Validator{PublicKeyByte: []byte("PublicKeyByte"), Weight: 100, NodeIDs: [][]byte{}}},
				SignersInput:       signersInput,
			},
			true,
		},
		{
			"signedStorageRoot len is wrong",
			&ibcava.ConsensusState{
				Timestamp:          suite.now,
				StorageRoot:        storageRoot,
				SignedStorageRoot:  []byte("123"),
				ValidatorSet:       validatorSet,
				SignedValidatorSet: signedValidatorSet[:],
				Vdrs:               []*ibcava.Validator{&ibcava.Validator{PublicKeyByte: []byte("PublicKeyByte"), Weight: 100, NodeIDs: [][]byte{}}},
				SignersInput:       signersInput,
			},
			false,
		},
		{
			"SignedValidatorSet len is wrong",
			&ibcava.ConsensusState{
				Timestamp:          suite.now,
				StorageRoot:        storageRoot,
				SignedStorageRoot:  signedStorageRoot[:],
				ValidatorSet:       validatorSet,
				SignedValidatorSet: []byte("123"),
				Vdrs:               []*ibcava.Validator{&ibcava.Validator{PublicKeyByte: []byte("PublicKeyByte"), Weight: 100, NodeIDs: [][]byte{}}},
				SignersInput:       signersInput,
			},
			false,
		},
		{
			"root is nil",
			&ibcava.ConsensusState{
				Timestamp:          suite.now,
				StorageRoot:        nil,
				SignedStorageRoot:  signedStorageRoot[:],
				ValidatorSet:       validatorSet,
				SignedValidatorSet: signedValidatorSet[:],
				Vdrs:               []*ibcava.Validator{&ibcava.Validator{PublicKeyByte: []byte("PublicKeyByte"), Weight: 100, NodeIDs: [][]byte{}}},
				SignersInput:       signersInput,
			},
			false,
		},
		{
			"root is empty",
			&ibcava.ConsensusState{
				Timestamp:          suite.now,
				StorageRoot:        []byte{},
				SignedStorageRoot:  signedStorageRoot[:],
				ValidatorSet:       validatorSet,
				SignedValidatorSet: signedValidatorSet[:],
				Vdrs:               []*ibcava.Validator{&ibcava.Validator{PublicKeyByte: []byte("PublicKeyByte"), Weight: 100, NodeIDs: [][]byte{}}},
				SignersInput:       signersInput,
			},
			false,
		},
		{
			"timestamp is zero",
			&ibcava.ConsensusState{
				Timestamp:          time.Time{},
				StorageRoot:        []byte{},
				SignedStorageRoot:  signedStorageRoot[:],
				ValidatorSet:       validatorSet,
				SignedValidatorSet: signedValidatorSet[:],
				Vdrs:               []*ibcava.Validator{&ibcava.Validator{PublicKeyByte: []byte("PublicKeyByte"), Weight: 100, NodeIDs: [][]byte{}}},
				SignersInput:       signersInput,
			},
			false,
		},
		{
			"signersInput is empty",
			&ibcava.ConsensusState{
				Timestamp:          time.Time{},
				StorageRoot:        storageRoot,
				SignedStorageRoot:  signedStorageRoot[:],
				ValidatorSet:       validatorSet,
				SignedValidatorSet: signedValidatorSet[:],
				Vdrs:               []*ibcava.Validator{&ibcava.Validator{PublicKeyByte: []byte("PublicKeyByte"), Weight: 100, NodeIDs: [][]byte{}}},
				SignersInput:       []byte{},
			},
			false,
		},
		{
			"signersInput is nil",
			&ibcava.ConsensusState{
				Timestamp:          time.Time{},
				StorageRoot:        storageRoot,
				SignedStorageRoot:  signedStorageRoot[:],
				ValidatorSet:       validatorSet,
				SignedValidatorSet: signedValidatorSet[:],
				Vdrs:               []*ibcava.Validator{&ibcava.Validator{PublicKeyByte: []byte("PublicKeyByte"), Weight: 100, NodeIDs: [][]byte{}}},
				SignersInput:       nil,
			},
			false,
		},
	}

	for i, tc := range testCases {
		tc := tc

		err := tc.consensusState.ValidateBasic()
		if tc.expectPass {
			suite.Require().NoError(err, "valid test case %d failed: %s", i, tc.msg)
		} else {
			suite.Require().Error(err, "invalid test case %d passed: %s", i, tc.msg)
		}
	}
}
