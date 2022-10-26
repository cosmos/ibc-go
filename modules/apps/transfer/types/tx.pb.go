// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: ibc/applications/transfer/v1/tx.proto

package types

import (
	context "context"
	fmt "fmt"
	types "github.com/cosmos/cosmos-sdk/types"
	types1 "github.com/cosmos/ibc-go/v5/modules/core/02-client/types"
	_ "github.com/gogo/protobuf/gogoproto"
	grpc1 "github.com/gogo/protobuf/grpc"
	proto "github.com/gogo/protobuf/proto"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
	io "io"
	math "math"
	math_bits "math/bits"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.GoGoProtoPackageIsVersion3 // please upgrade the proto package

// MsgTransfer defines a msg to transfer fungible tokens (i.e Coins) between
// ICS20 enabled chains. See ICS Spec here:
// https://github.com/cosmos/ibc/tree/master/spec/app/ics-020-fungible-token-transfer#data-structures
type MsgTransfer struct {
	// the port on which the packet will be sent
	SourcePort string `protobuf:"bytes,1,opt,name=source_port,json=sourcePort,proto3" json:"source_port,omitempty" yaml:"source_port"`
	// the channel by which the packet will be sent
	SourceChannel string `protobuf:"bytes,2,opt,name=source_channel,json=sourceChannel,proto3" json:"source_channel,omitempty" yaml:"source_channel"`
	// the tokens to be transferred
	Token types.Coin `protobuf:"bytes,3,opt,name=token,proto3" json:"token"`
	// the sender address
	Sender string `protobuf:"bytes,4,opt,name=sender,proto3" json:"sender,omitempty"`
	// the recipient address on the destination chain
	Receiver string `protobuf:"bytes,5,opt,name=receiver,proto3" json:"receiver,omitempty"`
	// Timeout height relative to the current block height.
	// The timeout is disabled when set to 0.
	TimeoutHeight types1.Height `protobuf:"bytes,6,opt,name=timeout_height,json=timeoutHeight,proto3" json:"timeout_height" yaml:"timeout_height"`
	// Timeout timestamp in absolute nanoseconds since unix epoch.
	// The timeout is disabled when set to 0.
	TimeoutTimestamp uint64 `protobuf:"varint,7,opt,name=timeout_timestamp,json=timeoutTimestamp,proto3" json:"timeout_timestamp,omitempty" yaml:"timeout_timestamp"`
	// optional memo
	Memo string `protobuf:"bytes,8,opt,name=memo,proto3" json:"memo,omitempty"`
}

func (m *MsgTransfer) Reset()         { *m = MsgTransfer{} }
func (m *MsgTransfer) String() string { return proto.CompactTextString(m) }
func (*MsgTransfer) ProtoMessage()    {}
func (*MsgTransfer) Descriptor() ([]byte, []int) {
	return fileDescriptor_7401ed9bed2f8e09, []int{0}
}
func (m *MsgTransfer) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *MsgTransfer) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_MsgTransfer.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *MsgTransfer) XXX_Merge(src proto.Message) {
	xxx_messageInfo_MsgTransfer.Merge(m, src)
}
func (m *MsgTransfer) XXX_Size() int {
	return m.Size()
}
func (m *MsgTransfer) XXX_DiscardUnknown() {
	xxx_messageInfo_MsgTransfer.DiscardUnknown(m)
}

var xxx_messageInfo_MsgTransfer proto.InternalMessageInfo

// MsgTransferResponse defines the Msg/Transfer response type.
type MsgTransferResponse struct {
	// sequence number of the transfer packet sent
	Sequence uint64 `protobuf:"varint,1,opt,name=sequence,proto3" json:"sequence,omitempty"`
}

func (m *MsgTransferResponse) Reset()         { *m = MsgTransferResponse{} }
func (m *MsgTransferResponse) String() string { return proto.CompactTextString(m) }
func (*MsgTransferResponse) ProtoMessage()    {}
func (*MsgTransferResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_7401ed9bed2f8e09, []int{1}
}
func (m *MsgTransferResponse) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *MsgTransferResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_MsgTransferResponse.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *MsgTransferResponse) XXX_Merge(src proto.Message) {
	xxx_messageInfo_MsgTransferResponse.Merge(m, src)
}
func (m *MsgTransferResponse) XXX_Size() int {
	return m.Size()
}
func (m *MsgTransferResponse) XXX_DiscardUnknown() {
	xxx_messageInfo_MsgTransferResponse.DiscardUnknown(m)
}

var xxx_messageInfo_MsgTransferResponse proto.InternalMessageInfo

func (m *MsgTransferResponse) GetSequence() uint64 {
	if m != nil {
		return m.Sequence
	}
	return 0
}

func init() {
	proto.RegisterType((*MsgTransfer)(nil), "ibc.applications.transfer.v1.MsgTransfer")
	proto.RegisterType((*MsgTransferResponse)(nil), "ibc.applications.transfer.v1.MsgTransferResponse")
}

func init() {
	proto.RegisterFile("ibc/applications/transfer/v1/tx.proto", fileDescriptor_7401ed9bed2f8e09)
}

var fileDescriptor_7401ed9bed2f8e09 = []byte{
<<<<<<< HEAD
	// 521 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x8c, 0x93, 0x31, 0x6f, 0xd3, 0x40,
	0x14, 0xc7, 0x6d, 0x92, 0x86, 0x70, 0xa1, 0x15, 0x18, 0xa8, 0xdc, 0xa8, 0xd8, 0x91, 0x25, 0xa4,
	0x30, 0x70, 0x27, 0x17, 0x55, 0x95, 0x3a, 0xa1, 0x74, 0x81, 0xa1, 0x12, 0x58, 0x9d, 0x58, 0xca,
	0xf9, 0xf2, 0x70, 0x4e, 0xc4, 0x77, 0xc6, 0x77, 0xb1, 0xe8, 0x37, 0x60, 0xe4, 0x23, 0xf4, 0xd3,
	0xa0, 0x8e, 0x1d, 0x99, 0x22, 0x94, 0x2c, 0xcc, 0xf9, 0x04, 0xe8, 0x6c, 0x27, 0x38, 0x0b, 0xea,
	0x94, 0xfb, 0xbf, 0xf7, 0x7b, 0xf9, 0xfb, 0xdd, 0x7b, 0x87, 0x5e, 0xf0, 0x98, 0x11, 0x9a, 0x65,
	0x53, 0xce, 0xa8, 0xe6, 0x52, 0x28, 0xa2, 0x73, 0x2a, 0xd4, 0x67, 0xc8, 0x49, 0x11, 0x12, 0xfd,
	0x0d, 0x67, 0xb9, 0xd4, 0xd2, 0x39, 0xe4, 0x31, 0xc3, 0x4d, 0x0c, 0xaf, 0x31, 0x5c, 0x84, 0xfd,
	0xa7, 0x89, 0x4c, 0x64, 0x09, 0x12, 0x73, 0xaa, 0x6a, 0xfa, 0x1e, 0x93, 0x2a, 0x95, 0x8a, 0xc4,
	0x54, 0x01, 0x29, 0xc2, 0x18, 0x34, 0x0d, 0x09, 0x93, 0x5c, 0xd4, 0x79, 0xdf, 0x58, 0x33, 0x99,
	0x03, 0x61, 0x53, 0x0e, 0x42, 0x1b, 0xc3, 0xea, 0x54, 0x01, 0xc1, 0xcf, 0x16, 0xea, 0x9d, 0xab,
	0xe4, 0xa2, 0x76, 0x72, 0x4e, 0x50, 0x4f, 0xc9, 0x59, 0xce, 0xe0, 0x32, 0x93, 0xb9, 0x76, 0xed,
	0x81, 0x3d, 0x7c, 0x30, 0xda, 0x5f, 0xcd, 0x7d, 0xe7, 0x8a, 0xa6, 0xd3, 0xd3, 0xa0, 0x91, 0x0c,
	0x22, 0x54, 0xa9, 0xf7, 0x32, 0xd7, 0xce, 0x1b, 0xb4, 0x57, 0xe7, 0xd8, 0x84, 0x0a, 0x01, 0x53,
	0xf7, 0x5e, 0x59, 0x7b, 0xb0, 0x9a, 0xfb, 0xcf, 0xb6, 0x6a, 0xeb, 0x7c, 0x10, 0xed, 0x56, 0x81,
	0xb3, 0x4a, 0x3b, 0xc7, 0x68, 0x47, 0xcb, 0x2f, 0x20, 0xdc, 0xd6, 0xc0, 0x1e, 0xf6, 0x8e, 0x0e,
	0x70, 0xd5, 0x1b, 0x36, 0xbd, 0xe1, 0xba, 0x37, 0x7c, 0x26, 0xb9, 0x18, 0xb5, 0x6f, 0xe6, 0xbe,
	0x15, 0x55, 0xb4, 0xb3, 0x8f, 0x3a, 0x0a, 0xc4, 0x18, 0x72, 0xb7, 0x6d, 0x0c, 0xa3, 0x5a, 0x39,
	0x7d, 0xd4, 0xcd, 0x81, 0x01, 0x2f, 0x20, 0x77, 0x77, 0xca, 0xcc, 0x46, 0x3b, 0x9f, 0xd0, 0x9e,
	0xe6, 0x29, 0xc8, 0x99, 0xbe, 0x9c, 0x00, 0x4f, 0x26, 0xda, 0xed, 0x94, 0x9e, 0x7d, 0x6c, 0x66,
	0x60, 0xee, 0x0b, 0xd7, 0xb7, 0x54, 0x84, 0xf8, 0x6d, 0x49, 0x8c, 0x9e, 0x1b, 0xd3, 0x7f, 0xcd,
	0x6c, 0xd7, 0x07, 0xd1, 0x6e, 0x1d, 0xa8, 0x68, 0xe7, 0x1d, 0x7a, 0xbc, 0x26, 0xcc, 0xaf, 0xd2,
	0x34, 0xcd, 0xdc, 0xfb, 0x03, 0x7b, 0xd8, 0x1e, 0x1d, 0xae, 0xe6, 0xbe, 0xbb, 0xfd, 0x27, 0x1b,
	0x24, 0x88, 0x1e, 0xd5, 0xb1, 0x8b, 0x75, 0xc8, 0x34, 0x92, 0x82, 0xa6, 0x63, 0xaa, 0xa9, 0xdb,
	0x1d, 0xd8, 0xc3, 0x87, 0xd1, 0x46, 0x9f, 0x76, 0xbf, 0x5f, 0xfb, 0xd6, 0x9f, 0x6b, 0xdf, 0x0a,
	0x42, 0xf4, 0xa4, 0x31, 0xc7, 0x08, 0x54, 0x26, 0x85, 0x02, 0x53, 0xac, 0xe0, 0xeb, 0x0c, 0x04,
	0x83, 0x72, 0x98, 0xed, 0x68, 0xa3, 0x8f, 0x24, 0x6a, 0x9d, 0xab, 0xc4, 0x99, 0xa0, 0xee, 0x66,
	0xfc, 0x2f, 0xf1, 0xff, 0x96, 0x10, 0x37, 0x1c, 0xfa, 0xe1, 0x9d, 0xd1, 0xf5, 0xc7, 0x8c, 0x3e,
	0xdc, 0x2c, 0x3c, 0xfb, 0x76, 0xe1, 0xd9, 0xbf, 0x17, 0x9e, 0xfd, 0x63, 0xe9, 0x59, 0xb7, 0x4b,
	0xcf, 0xfa, 0xb5, 0xf4, 0xac, 0x8f, 0x27, 0x09, 0xd7, 0x93, 0x59, 0x8c, 0x99, 0x4c, 0x49, 0xbd,
	0xd2, 0x3c, 0x66, 0xaf, 0x12, 0x49, 0x8a, 0x63, 0x92, 0xca, 0xf1, 0x6c, 0x0a, 0xca, 0x3c, 0xa1,
	0xc6, 0xd3, 0xd1, 0x57, 0x19, 0xa8, 0xb8, 0x53, 0xae, 0xf1, 0xeb, 0xbf, 0x01, 0x00, 0x00, 0xff,
	0xff, 0xa0, 0x73, 0x9e, 0xaf, 0x64, 0x03, 0x00, 0x00,
=======
	// 516 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x8c, 0x53, 0xc1, 0x6e, 0xd3, 0x40,
	0x10, 0xb5, 0x49, 0x1a, 0xc2, 0x46, 0xad, 0x60, 0x81, 0xca, 0x8d, 0x8a, 0x1d, 0x59, 0x42, 0x0a,
	0x07, 0x76, 0xe5, 0x22, 0xa8, 0xd4, 0x13, 0x4a, 0x2f, 0x70, 0xa8, 0x04, 0x56, 0x4f, 0x5c, 0x8a,
	0xbd, 0x1d, 0x9c, 0x15, 0xb1, 0xc7, 0x78, 0x37, 0x16, 0xfd, 0x03, 0x8e, 0x7c, 0x42, 0xbf, 0x84,
	0x73, 0x8f, 0x3d, 0x72, 0x8a, 0x50, 0x72, 0xe1, 0x9c, 0x2f, 0x40, 0x6b, 0x3b, 0x21, 0xb9, 0x20,
	0x4e, 0x9e, 0x99, 0xf7, 0xc6, 0x6f, 0xdf, 0xce, 0x2c, 0x79, 0x2a, 0x63, 0xc1, 0xa3, 0x3c, 0x9f,
	0x48, 0x11, 0x69, 0x89, 0x99, 0xe2, 0xba, 0x88, 0x32, 0xf5, 0x09, 0x0a, 0x5e, 0x06, 0x5c, 0x7f,
	0x65, 0x79, 0x81, 0x1a, 0xe9, 0xa1, 0x8c, 0x05, 0xdb, 0xa4, 0xb1, 0x15, 0x8d, 0x95, 0x41, 0xff,
	0x51, 0x82, 0x09, 0x56, 0x44, 0x6e, 0xa2, 0xba, 0xa7, 0xef, 0x0a, 0x54, 0x29, 0x2a, 0x1e, 0x47,
	0x0a, 0x78, 0x19, 0xc4, 0xa0, 0xa3, 0x80, 0x0b, 0x94, 0x59, 0x83, 0x7b, 0x46, 0x5a, 0x60, 0x01,
	0x5c, 0x4c, 0x24, 0x64, 0xda, 0x08, 0xd6, 0x51, 0x4d, 0xf0, 0x7f, 0xb4, 0x48, 0xef, 0x4c, 0x25,
	0xe7, 0x8d, 0x12, 0x3d, 0x26, 0x3d, 0x85, 0xd3, 0x42, 0xc0, 0x45, 0x8e, 0x85, 0x76, 0xec, 0x81,
	0x3d, 0xbc, 0x37, 0xda, 0x5f, 0xce, 0x3c, 0x7a, 0x15, 0xa5, 0x93, 0x13, 0x7f, 0x03, 0xf4, 0x43,
	0x52, 0x67, 0xef, 0xb0, 0xd0, 0xf4, 0x35, 0xd9, 0x6b, 0x30, 0x31, 0x8e, 0xb2, 0x0c, 0x26, 0xce,
	0x9d, 0xaa, 0xf7, 0x60, 0x39, 0xf3, 0x1e, 0x6f, 0xf5, 0x36, 0xb8, 0x1f, 0xee, 0xd6, 0x85, 0xd3,
	0x3a, 0xa7, 0x2f, 0xc9, 0x8e, 0xc6, 0xcf, 0x90, 0x39, 0xad, 0x81, 0x3d, 0xec, 0x1d, 0x1d, 0xb0,
	0xda, 0x1b, 0x33, 0xde, 0x58, 0xe3, 0x8d, 0x9d, 0xa2, 0xcc, 0x46, 0xed, 0x9b, 0x99, 0x67, 0x85,
	0x35, 0x9b, 0xee, 0x93, 0x8e, 0x82, 0xec, 0x12, 0x0a, 0xa7, 0x6d, 0x04, 0xc3, 0x26, 0xa3, 0x7d,
	0xd2, 0x2d, 0x40, 0x80, 0x2c, 0xa1, 0x70, 0x76, 0x2a, 0x64, 0x9d, 0xd3, 0x8f, 0x64, 0x4f, 0xcb,
	0x14, 0x70, 0xaa, 0x2f, 0xc6, 0x20, 0x93, 0xb1, 0x76, 0x3a, 0x95, 0x66, 0x9f, 0x99, 0x19, 0x98,
	0xfb, 0x62, 0xcd, 0x2d, 0x95, 0x01, 0x7b, 0x53, 0x31, 0x46, 0x4f, 0x8c, 0xe8, 0x5f, 0x33, 0xdb,
	0xfd, 0x7e, 0xb8, 0xdb, 0x14, 0x6a, 0x36, 0x7d, 0x4b, 0x1e, 0xac, 0x18, 0xe6, 0xab, 0x74, 0x94,
	0xe6, 0xce, 0xdd, 0x81, 0x3d, 0x6c, 0x8f, 0x0e, 0x97, 0x33, 0xcf, 0xd9, 0xfe, 0xc9, 0x9a, 0xe2,
	0x87, 0xf7, 0x9b, 0xda, 0xf9, 0xaa, 0x44, 0x29, 0x69, 0xa7, 0x90, 0xa2, 0xd3, 0xad, 0x4c, 0x54,
	0xf1, 0x49, 0xf7, 0xdb, 0xb5, 0x67, 0xfd, 0xbe, 0xf6, 0x2c, 0x3f, 0x20, 0x0f, 0x37, 0xe6, 0x17,
	0x82, 0xca, 0x31, 0x53, 0x60, 0xdc, 0x2b, 0xf8, 0x32, 0x85, 0x4c, 0x40, 0x35, 0xc4, 0x76, 0xb8,
	0xce, 0x8f, 0x90, 0xb4, 0xce, 0x54, 0x42, 0xc7, 0xa4, 0xbb, 0x1e, 0xfb, 0x33, 0xf6, 0xaf, 0xe5,
	0x63, 0x1b, 0x0a, 0xfd, 0xe0, 0xbf, 0xa9, 0xab, 0xc3, 0x8c, 0xde, 0xdf, 0xcc, 0x5d, 0xfb, 0x76,
	0xee, 0xda, 0xbf, 0xe6, 0xae, 0xfd, 0x7d, 0xe1, 0x5a, 0xb7, 0x0b, 0xd7, 0xfa, 0xb9, 0x70, 0xad,
	0x0f, 0xc7, 0x89, 0xd4, 0xe3, 0x69, 0xcc, 0x04, 0xa6, 0xbc, 0x59, 0x65, 0x19, 0x8b, 0xe7, 0x09,
	0xf2, 0xf2, 0x15, 0x4f, 0xf1, 0x72, 0x3a, 0x01, 0x65, 0x9e, 0xce, 0xc6, 0x93, 0xd1, 0x57, 0x39,
	0xa8, 0xb8, 0x53, 0xad, 0xef, 0x8b, 0x3f, 0x01, 0x00, 0x00, 0xff, 0xff, 0xc1, 0x2a, 0x32, 0x96,
	0x5c, 0x03, 0x00, 0x00,
>>>>>>> 05685b3 (refactor: adapting transfer metadata bytes field to memo string (#2595))
}

// Reference imports to suppress errors if they are not otherwise used.
var _ context.Context
var _ grpc.ClientConn

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc.SupportPackageIsVersion4

// MsgClient is the client API for Msg service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://godoc.org/google.golang.org/grpc#ClientConn.NewStream.
type MsgClient interface {
	// Transfer defines a rpc handler method for MsgTransfer.
	Transfer(ctx context.Context, in *MsgTransfer, opts ...grpc.CallOption) (*MsgTransferResponse, error)
}

type msgClient struct {
	cc grpc1.ClientConn
}

func NewMsgClient(cc grpc1.ClientConn) MsgClient {
	return &msgClient{cc}
}

func (c *msgClient) Transfer(ctx context.Context, in *MsgTransfer, opts ...grpc.CallOption) (*MsgTransferResponse, error) {
	out := new(MsgTransferResponse)
	err := c.cc.Invoke(ctx, "/ibc.applications.transfer.v1.Msg/Transfer", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// MsgServer is the server API for Msg service.
type MsgServer interface {
	// Transfer defines a rpc handler method for MsgTransfer.
	Transfer(context.Context, *MsgTransfer) (*MsgTransferResponse, error)
}

// UnimplementedMsgServer can be embedded to have forward compatible implementations.
type UnimplementedMsgServer struct {
}

func (*UnimplementedMsgServer) Transfer(ctx context.Context, req *MsgTransfer) (*MsgTransferResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Transfer not implemented")
}

func RegisterMsgServer(s grpc1.Server, srv MsgServer) {
	s.RegisterService(&_Msg_serviceDesc, srv)
}

func _Msg_Transfer_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgTransfer)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).Transfer(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/ibc.applications.transfer.v1.Msg/Transfer",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).Transfer(ctx, req.(*MsgTransfer))
	}
	return interceptor(ctx, in, info, handler)
}

var _Msg_serviceDesc = grpc.ServiceDesc{
	ServiceName: "ibc.applications.transfer.v1.Msg",
	HandlerType: (*MsgServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Transfer",
			Handler:    _Msg_Transfer_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "ibc/applications/transfer/v1/tx.proto",
}

func (m *MsgTransfer) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *MsgTransfer) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *MsgTransfer) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if len(m.Memo) > 0 {
		i -= len(m.Memo)
		copy(dAtA[i:], m.Memo)
		i = encodeVarintTx(dAtA, i, uint64(len(m.Memo)))
		i--
		dAtA[i] = 0x42
	}
	if m.TimeoutTimestamp != 0 {
		i = encodeVarintTx(dAtA, i, uint64(m.TimeoutTimestamp))
		i--
		dAtA[i] = 0x38
	}
	{
		size, err := m.TimeoutHeight.MarshalToSizedBuffer(dAtA[:i])
		if err != nil {
			return 0, err
		}
		i -= size
		i = encodeVarintTx(dAtA, i, uint64(size))
	}
	i--
	dAtA[i] = 0x32
	if len(m.Receiver) > 0 {
		i -= len(m.Receiver)
		copy(dAtA[i:], m.Receiver)
		i = encodeVarintTx(dAtA, i, uint64(len(m.Receiver)))
		i--
		dAtA[i] = 0x2a
	}
	if len(m.Sender) > 0 {
		i -= len(m.Sender)
		copy(dAtA[i:], m.Sender)
		i = encodeVarintTx(dAtA, i, uint64(len(m.Sender)))
		i--
		dAtA[i] = 0x22
	}
	{
		size, err := m.Token.MarshalToSizedBuffer(dAtA[:i])
		if err != nil {
			return 0, err
		}
		i -= size
		i = encodeVarintTx(dAtA, i, uint64(size))
	}
	i--
	dAtA[i] = 0x1a
	if len(m.SourceChannel) > 0 {
		i -= len(m.SourceChannel)
		copy(dAtA[i:], m.SourceChannel)
		i = encodeVarintTx(dAtA, i, uint64(len(m.SourceChannel)))
		i--
		dAtA[i] = 0x12
	}
	if len(m.SourcePort) > 0 {
		i -= len(m.SourcePort)
		copy(dAtA[i:], m.SourcePort)
		i = encodeVarintTx(dAtA, i, uint64(len(m.SourcePort)))
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

func (m *MsgTransferResponse) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *MsgTransferResponse) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *MsgTransferResponse) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if m.Sequence != 0 {
		i = encodeVarintTx(dAtA, i, uint64(m.Sequence))
		i--
		dAtA[i] = 0x8
	}
	return len(dAtA) - i, nil
}

func encodeVarintTx(dAtA []byte, offset int, v uint64) int {
	offset -= sovTx(v)
	base := offset
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return base
}
func (m *MsgTransfer) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.SourcePort)
	if l > 0 {
		n += 1 + l + sovTx(uint64(l))
	}
	l = len(m.SourceChannel)
	if l > 0 {
		n += 1 + l + sovTx(uint64(l))
	}
	l = m.Token.Size()
	n += 1 + l + sovTx(uint64(l))
	l = len(m.Sender)
	if l > 0 {
		n += 1 + l + sovTx(uint64(l))
	}
	l = len(m.Receiver)
	if l > 0 {
		n += 1 + l + sovTx(uint64(l))
	}
	l = m.TimeoutHeight.Size()
	n += 1 + l + sovTx(uint64(l))
	if m.TimeoutTimestamp != 0 {
		n += 1 + sovTx(uint64(m.TimeoutTimestamp))
	}
	l = len(m.Memo)
	if l > 0 {
		n += 1 + l + sovTx(uint64(l))
	}
	return n
}

func (m *MsgTransferResponse) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	if m.Sequence != 0 {
		n += 1 + sovTx(uint64(m.Sequence))
	}
	return n
}

func sovTx(x uint64) (n int) {
	return (math_bits.Len64(x|1) + 6) / 7
}
func sozTx(x uint64) (n int) {
	return sovTx(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (m *MsgTransfer) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowTx
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: MsgTransfer: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: MsgTransfer: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field SourcePort", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTx
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthTx
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthTx
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.SourcePort = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field SourceChannel", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTx
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthTx
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthTx
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.SourceChannel = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 3:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Token", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTx
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				msglen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if msglen < 0 {
				return ErrInvalidLengthTx
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthTx
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := m.Token.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 4:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Sender", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTx
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthTx
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthTx
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Sender = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 5:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Receiver", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTx
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthTx
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthTx
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Receiver = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 6:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field TimeoutHeight", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTx
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				msglen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if msglen < 0 {
				return ErrInvalidLengthTx
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthTx
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := m.TimeoutHeight.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 7:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field TimeoutTimestamp", wireType)
			}
			m.TimeoutTimestamp = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTx
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.TimeoutTimestamp |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 8:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Memo", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTx
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthTx
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthTx
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Memo = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipTx(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthTx
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func (m *MsgTransferResponse) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowTx
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: MsgTransferResponse: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: MsgTransferResponse: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Sequence", wireType)
			}
			m.Sequence = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTx
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.Sequence |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		default:
			iNdEx = preIndex
			skippy, err := skipTx(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthTx
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func skipTx(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	depth := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowTx
			}
			if iNdEx >= l {
				return 0, io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= (uint64(b) & 0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		wireType := int(wire & 0x7)
		switch wireType {
		case 0:
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return 0, ErrIntOverflowTx
				}
				if iNdEx >= l {
					return 0, io.ErrUnexpectedEOF
				}
				iNdEx++
				if dAtA[iNdEx-1] < 0x80 {
					break
				}
			}
		case 1:
			iNdEx += 8
		case 2:
			var length int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return 0, ErrIntOverflowTx
				}
				if iNdEx >= l {
					return 0, io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				length |= (int(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if length < 0 {
				return 0, ErrInvalidLengthTx
			}
			iNdEx += length
		case 3:
			depth++
		case 4:
			if depth == 0 {
				return 0, ErrUnexpectedEndOfGroupTx
			}
			depth--
		case 5:
			iNdEx += 4
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
		if iNdEx < 0 {
			return 0, ErrInvalidLengthTx
		}
		if depth == 0 {
			return iNdEx, nil
		}
	}
	return 0, io.ErrUnexpectedEOF
}

var (
	ErrInvalidLengthTx        = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowTx          = fmt.Errorf("proto: integer overflow")
	ErrUnexpectedEndOfGroupTx = fmt.Errorf("proto: unexpected end of group")
)
