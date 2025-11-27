package types_test

import (
	"github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts/types"
	connectiontypes "github.com/cosmos/ibc-go/v10/modules/core/03-connection/types"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
)

// use TestVersion as metadata being compared against
func (s *TypesTestSuite) TestIsPreviousMetadataEqual() {
	var (
		metadata        types.Metadata
		previousVersion string
	)

	testCases := []struct {
		name     string
		malleate func()
		expEqual bool
	}{
		{
			"success",
			func() {
				versionBytes, err := types.ModuleCdc.MarshalJSON(&metadata)
				s.Require().NoError(err)
				previousVersion = string(versionBytes)
			},
			true,
		},
		{
			"success with empty account address",
			func() {
				metadata.Address = ""

				versionBytes, err := types.ModuleCdc.MarshalJSON(&metadata)
				s.Require().NoError(err)
				previousVersion = string(versionBytes)
			},
			true,
		},
		{
			"cannot decode previous version",
			func() {
				previousVersion = "invalid previous version"
			},
			false,
		},
		{
			"unequal and invalid encoding format",
			func() {
				metadata.Encoding = "invalid-encoding-format"

				versionBytes, err := types.ModuleCdc.MarshalJSON(&metadata)
				s.Require().NoError(err)
				previousVersion = string(versionBytes)
			},
			false,
		},
		{
			"unequal encoding format",
			func() {
				metadata.Encoding = types.EncodingProto3JSON

				versionBytes, err := types.ModuleCdc.MarshalJSON(&metadata)
				s.Require().NoError(err)
				previousVersion = string(versionBytes)
			},
			false,
		},
		{
			"unequal transaction type",
			func() {
				metadata.TxType = "invalid-tx-type"

				versionBytes, err := types.ModuleCdc.MarshalJSON(&metadata)
				s.Require().NoError(err)
				previousVersion = string(versionBytes)
			},
			false,
		},
		{
			"unequal controller connection",
			func() {
				metadata.ControllerConnectionId = "connection-10"

				versionBytes, err := types.ModuleCdc.MarshalJSON(&metadata)
				s.Require().NoError(err)
				previousVersion = string(versionBytes)
			},
			false,
		},
		{
			"unequal host connection",
			func() {
				metadata.HostConnectionId = "connection-10"

				versionBytes, err := types.ModuleCdc.MarshalJSON(&metadata)
				s.Require().NoError(err)
				previousVersion = string(versionBytes)
			},
			false,
		},
		{
			"unequal version",
			func() {
				metadata.Version = "invalid version"

				versionBytes, err := types.ModuleCdc.MarshalJSON(&metadata)
				s.Require().NoError(err)
				previousVersion = string(versionBytes)
			},
			false,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest() // reset

			path := ibctesting.NewPath(s.chainA, s.chainB)
			path.SetupConnections()

			expectedMetadata := types.NewMetadata(types.Version, ibctesting.FirstConnectionID, ibctesting.FirstConnectionID, TestOwnerAddress, types.EncodingProtobuf, types.TxTypeSDKMultiMsg)
			metadata = expectedMetadata // default success case

			tc.malleate() // malleate mutates test data

			equal := types.IsPreviousMetadataEqual(previousVersion, expectedMetadata)

			if tc.expEqual {
				s.Require().True(equal)
			} else {
				s.Require().False(equal)
			}
		})
	}
}

func (s *TypesTestSuite) TestValidateControllerMetadata() {
	var metadata types.Metadata

	testCases := []struct {
		name     string
		malleate func()
		expErr   error
	}{
		{
			"success",
			func() {},
			nil,
		},
		{
			"success with empty account address",
			func() {
				metadata = types.Metadata{
					Version:                types.Version,
					ControllerConnectionId: ibctesting.FirstConnectionID,
					HostConnectionId:       ibctesting.FirstConnectionID,
					Address:                "",
					Encoding:               types.EncodingProtobuf,
					TxType:                 types.TxTypeSDKMultiMsg,
				}
			},
			nil,
		},
		{
			"success with EncodingProto3JSON",
			func() {
				metadata = types.Metadata{
					Version:                types.Version,
					ControllerConnectionId: ibctesting.FirstConnectionID,
					HostConnectionId:       ibctesting.FirstConnectionID,
					Address:                TestOwnerAddress,
					Encoding:               types.EncodingProto3JSON,
					TxType:                 types.TxTypeSDKMultiMsg,
				}
			},
			nil,
		},
		{
			"unsupported encoding format",
			func() {
				metadata = types.Metadata{
					Version:                types.Version,
					ControllerConnectionId: ibctesting.FirstConnectionID,
					HostConnectionId:       ibctesting.FirstConnectionID,
					Address:                TestOwnerAddress,
					Encoding:               "invalid-encoding-format",
					TxType:                 types.TxTypeSDKMultiMsg,
				}
			},
			types.ErrInvalidCodec,
		},
		{
			"unsupported transaction type",
			func() {
				metadata = types.Metadata{
					Version:                types.Version,
					ControllerConnectionId: ibctesting.FirstConnectionID,
					HostConnectionId:       ibctesting.FirstConnectionID,
					Address:                TestOwnerAddress,
					Encoding:               types.EncodingProtobuf,
					TxType:                 "invalid-tx-type",
				}
			},
			types.ErrUnknownDataType,
		},
		{
			"invalid controller connection",
			func() {
				metadata = types.Metadata{
					Version:                types.Version,
					ControllerConnectionId: "connection-10",
					HostConnectionId:       ibctesting.FirstConnectionID,
					Address:                TestOwnerAddress,
					Encoding:               types.EncodingProtobuf,
					TxType:                 types.TxTypeSDKMultiMsg,
				}
			},
			connectiontypes.ErrInvalidConnection,
		},
		{
			"invalid host connection",
			func() {
				metadata = types.Metadata{
					Version:                types.Version,
					ControllerConnectionId: ibctesting.FirstConnectionID,
					HostConnectionId:       "connection-10",
					Address:                TestOwnerAddress,
					Encoding:               types.EncodingProtobuf,
					TxType:                 types.TxTypeSDKMultiMsg,
				}
			},
			connectiontypes.ErrInvalidConnection,
		},
		{
			"invalid address",
			func() {
				metadata = types.Metadata{
					Version:                types.Version,
					ControllerConnectionId: ibctesting.FirstConnectionID,
					HostConnectionId:       ibctesting.FirstConnectionID,
					Address:                " ",
					Encoding:               types.EncodingProtobuf,
					TxType:                 types.TxTypeSDKMultiMsg,
				}
			},
			types.ErrInvalidAccountAddress,
		},
		{
			"invalid version",
			func() {
				metadata = types.Metadata{
					Version:                "invalid version",
					ControllerConnectionId: ibctesting.FirstConnectionID,
					HostConnectionId:       ibctesting.FirstConnectionID,
					Address:                TestOwnerAddress,
					Encoding:               types.EncodingProtobuf,
					TxType:                 types.TxTypeSDKMultiMsg,
				}
			},
			types.ErrInvalidVersion,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest() // reset

			path := ibctesting.NewPath(s.chainA, s.chainB)
			path.SetupConnections()

			metadata = types.NewMetadata(types.Version, ibctesting.FirstConnectionID, ibctesting.FirstConnectionID, TestOwnerAddress, types.EncodingProtobuf, types.TxTypeSDKMultiMsg)

			tc.malleate() // malleate mutates test data

			err := types.ValidateControllerMetadata(
				s.chainA.GetContext(),
				s.chainA.App.GetIBCKeeper().ChannelKeeper,
				[]string{ibctesting.FirstConnectionID},
				metadata,
			)

			if tc.expErr == nil {
				s.Require().NoError(err, tc.name)
			} else {
				s.Require().Error(err, tc.name)
				s.Require().ErrorIs(err, tc.expErr)
			}
		})
	}
}

func (s *TypesTestSuite) TestValidateHostMetadata() {
	var metadata types.Metadata

	testCases := []struct {
		name     string
		malleate func()
		expError error
	}{
		{
			"success",
			func() {},
			nil,
		},
		{
			"success with empty account address",
			func() {
				metadata = types.Metadata{
					Version:                types.Version,
					ControllerConnectionId: ibctesting.FirstConnectionID,
					HostConnectionId:       ibctesting.FirstConnectionID,
					Address:                "",
					Encoding:               types.EncodingProtobuf,
					TxType:                 types.TxTypeSDKMultiMsg,
				}
			},
			nil,
		},
		{
			"success with EncodingProto3JSON",
			func() {
				metadata = types.Metadata{
					Version:                types.Version,
					ControllerConnectionId: ibctesting.FirstConnectionID,
					HostConnectionId:       ibctesting.FirstConnectionID,
					Address:                TestOwnerAddress,
					Encoding:               types.EncodingProto3JSON,
					TxType:                 types.TxTypeSDKMultiMsg,
				}
			},
			nil,
		},
		{
			"unsupported encoding format",
			func() {
				metadata = types.Metadata{
					Version:                types.Version,
					ControllerConnectionId: ibctesting.FirstConnectionID,
					HostConnectionId:       ibctesting.FirstConnectionID,
					Address:                TestOwnerAddress,
					Encoding:               "invalid-encoding-format",
					TxType:                 types.TxTypeSDKMultiMsg,
				}
			},
			types.ErrInvalidCodec,
		},
		{
			"unsupported transaction type",
			func() {
				metadata = types.Metadata{
					Version:                types.Version,
					ControllerConnectionId: ibctesting.FirstConnectionID,
					HostConnectionId:       ibctesting.FirstConnectionID,
					Address:                TestOwnerAddress,
					Encoding:               types.EncodingProtobuf,
					TxType:                 "invalid-tx-type",
				}
			},
			types.ErrUnknownDataType,
		},
		{
			"invalid controller connection",
			func() {
				metadata = types.Metadata{
					Version:                types.Version,
					ControllerConnectionId: "connection-10",
					HostConnectionId:       ibctesting.FirstConnectionID,
					Address:                TestOwnerAddress,
					Encoding:               types.EncodingProtobuf,
					TxType:                 types.TxTypeSDKMultiMsg,
				}
			},
			connectiontypes.ErrInvalidConnection,
		},
		{
			"invalid host connection",
			func() {
				metadata = types.Metadata{
					Version:                types.Version,
					ControllerConnectionId: ibctesting.FirstConnectionID,
					HostConnectionId:       "connection-10",
					Address:                TestOwnerAddress,
					Encoding:               types.EncodingProtobuf,
					TxType:                 types.TxTypeSDKMultiMsg,
				}
			},
			connectiontypes.ErrInvalidConnection,
		},
		{
			"invalid address",
			func() {
				metadata = types.Metadata{
					Version:                types.Version,
					ControllerConnectionId: ibctesting.FirstConnectionID,
					HostConnectionId:       ibctesting.FirstConnectionID,
					Address:                " ",
					Encoding:               types.EncodingProtobuf,
					TxType:                 types.TxTypeSDKMultiMsg,
				}
			},
			types.ErrInvalidAccountAddress,
		},
		{
			"invalid version",
			func() {
				metadata = types.Metadata{
					Version:                "invalid version",
					ControllerConnectionId: ibctesting.FirstConnectionID,
					HostConnectionId:       ibctesting.FirstConnectionID,
					Address:                TestOwnerAddress,
					Encoding:               types.EncodingProtobuf,
					TxType:                 types.TxTypeSDKMultiMsg,
				}
			},
			types.ErrInvalidVersion,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest() // reset

			path := ibctesting.NewPath(s.chainA, s.chainB)
			path.SetupConnections()

			metadata = types.NewMetadata(types.Version, ibctesting.FirstConnectionID, ibctesting.FirstConnectionID, TestOwnerAddress, types.EncodingProtobuf, types.TxTypeSDKMultiMsg)

			tc.malleate() // malleate mutates test data

			err := types.ValidateHostMetadata(
				s.chainA.GetContext(),
				s.chainA.App.GetIBCKeeper().ChannelKeeper,
				[]string{ibctesting.FirstConnectionID},
				metadata,
			)

			if tc.expError == nil {
				s.Require().NoError(err, tc.name)
			} else {
				s.Require().ErrorIs(err, tc.expError)
			}
		})
	}
}
