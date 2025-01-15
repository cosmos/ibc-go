// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: ibc/core/client/v2/counterparty.proto

package types

import (
	context "context"
	fmt "fmt"
	_ "github.com/cosmos/cosmos-sdk/types/msgservice"
	_ "github.com/cosmos/gogoproto/gogoproto"
	grpc1 "github.com/cosmos/gogoproto/grpc"
	proto "github.com/cosmos/gogoproto/proto"
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

// CounterpartyInfo defines the key that the counterparty will use to message our client
type CounterpartyInfo struct {
	// merkle prefix key is the prefix that ics provable keys are stored under
	MerklePrefix [][]byte `protobuf:"bytes,1,rep,name=merkle_prefix,json=merklePrefix,proto3" json:"merkle_prefix,omitempty"`
	// client identifier is the identifier used to send packet messages to our client
	ClientId string `protobuf:"bytes,2,opt,name=client_id,json=clientId,proto3" json:"client_id,omitempty"`
}

func (m *CounterpartyInfo) Reset()         { *m = CounterpartyInfo{} }
func (m *CounterpartyInfo) String() string { return proto.CompactTextString(m) }
func (*CounterpartyInfo) ProtoMessage()    {}
func (*CounterpartyInfo) Descriptor() ([]byte, []int) {
	return fileDescriptor_bc4a81c3d2196cf1, []int{0}
}
func (m *CounterpartyInfo) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *CounterpartyInfo) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_CounterpartyInfo.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *CounterpartyInfo) XXX_Merge(src proto.Message) {
	xxx_messageInfo_CounterpartyInfo.Merge(m, src)
}
func (m *CounterpartyInfo) XXX_Size() int {
	return m.Size()
}
func (m *CounterpartyInfo) XXX_DiscardUnknown() {
	xxx_messageInfo_CounterpartyInfo.DiscardUnknown(m)
}

var xxx_messageInfo_CounterpartyInfo proto.InternalMessageInfo

func (m *CounterpartyInfo) GetMerklePrefix() [][]byte {
	if m != nil {
		return m.MerklePrefix
	}
	return nil
}

func (m *CounterpartyInfo) GetClientId() string {
	if m != nil {
		return m.ClientId
	}
	return ""
}

// MsgRegisterCounterparty defines a message to register a counterparty on a client
type MsgRegisterCounterparty struct {
	// client identifier
	ClientId string `protobuf:"bytes,1,opt,name=client_id,json=clientId,proto3" json:"client_id,omitempty"`
	// counterparty merkle prefix
	CounterpartyMerklePrefix [][]byte `protobuf:"bytes,2,rep,name=counterparty_merkle_prefix,json=counterpartyMerklePrefix,proto3" json:"counterparty_merkle_prefix,omitempty"`
	// counterparty client identifier
	CounterpartyClientId string `protobuf:"bytes,3,opt,name=counterparty_client_id,json=counterpartyClientId,proto3" json:"counterparty_client_id,omitempty"`
	// signer address
	Signer string `protobuf:"bytes,4,opt,name=signer,proto3" json:"signer,omitempty"`
}

func (m *MsgRegisterCounterparty) Reset()         { *m = MsgRegisterCounterparty{} }
func (m *MsgRegisterCounterparty) String() string { return proto.CompactTextString(m) }
func (*MsgRegisterCounterparty) ProtoMessage()    {}
func (*MsgRegisterCounterparty) Descriptor() ([]byte, []int) {
	return fileDescriptor_bc4a81c3d2196cf1, []int{1}
}
func (m *MsgRegisterCounterparty) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *MsgRegisterCounterparty) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_MsgRegisterCounterparty.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *MsgRegisterCounterparty) XXX_Merge(src proto.Message) {
	xxx_messageInfo_MsgRegisterCounterparty.Merge(m, src)
}
func (m *MsgRegisterCounterparty) XXX_Size() int {
	return m.Size()
}
func (m *MsgRegisterCounterparty) XXX_DiscardUnknown() {
	xxx_messageInfo_MsgRegisterCounterparty.DiscardUnknown(m)
}

var xxx_messageInfo_MsgRegisterCounterparty proto.InternalMessageInfo

// MsgRegisterCounterpartyResponse defines the Msg/RegisterCounterparty response type.
type MsgRegisterCounterpartyResponse struct {
}

func (m *MsgRegisterCounterpartyResponse) Reset()         { *m = MsgRegisterCounterpartyResponse{} }
func (m *MsgRegisterCounterpartyResponse) String() string { return proto.CompactTextString(m) }
func (*MsgRegisterCounterpartyResponse) ProtoMessage()    {}
func (*MsgRegisterCounterpartyResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_bc4a81c3d2196cf1, []int{2}
}
func (m *MsgRegisterCounterpartyResponse) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *MsgRegisterCounterpartyResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_MsgRegisterCounterpartyResponse.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *MsgRegisterCounterpartyResponse) XXX_Merge(src proto.Message) {
	xxx_messageInfo_MsgRegisterCounterpartyResponse.Merge(m, src)
}
func (m *MsgRegisterCounterpartyResponse) XXX_Size() int {
	return m.Size()
}
func (m *MsgRegisterCounterpartyResponse) XXX_DiscardUnknown() {
	xxx_messageInfo_MsgRegisterCounterpartyResponse.DiscardUnknown(m)
}

var xxx_messageInfo_MsgRegisterCounterpartyResponse proto.InternalMessageInfo

func init() {
	proto.RegisterType((*CounterpartyInfo)(nil), "ibc.core.client.v2.CounterpartyInfo")
	proto.RegisterType((*MsgRegisterCounterparty)(nil), "ibc.core.client.v2.MsgRegisterCounterparty")
	proto.RegisterType((*MsgRegisterCounterpartyResponse)(nil), "ibc.core.client.v2.MsgRegisterCounterpartyResponse")
}

func init() {
	proto.RegisterFile("ibc/core/client/v2/counterparty.proto", fileDescriptor_bc4a81c3d2196cf1)
}

var fileDescriptor_bc4a81c3d2196cf1 = []byte{
	// 388 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xe2, 0x52, 0xcd, 0x4c, 0x4a, 0xd6,
	0x4f, 0xce, 0x2f, 0x4a, 0xd5, 0x4f, 0xce, 0xc9, 0x4c, 0xcd, 0x2b, 0xd1, 0x2f, 0x33, 0xd2, 0x4f,
	0xce, 0x2f, 0xcd, 0x2b, 0x49, 0x2d, 0x2a, 0x48, 0x2c, 0x2a, 0xa9, 0xd4, 0x2b, 0x28, 0xca, 0x2f,
	0xc9, 0x17, 0x12, 0xca, 0x4c, 0x4a, 0xd6, 0x03, 0x29, 0xd3, 0x83, 0x28, 0xd3, 0x2b, 0x33, 0x92,
	0x12, 0x4f, 0xce, 0x2f, 0xce, 0xcd, 0x2f, 0xd6, 0xcf, 0x2d, 0x4e, 0xd7, 0x2f, 0x33, 0x04, 0x51,
	0x10, 0xc5, 0x52, 0x22, 0xe9, 0xf9, 0xe9, 0xf9, 0x60, 0xa6, 0x3e, 0x88, 0x05, 0x11, 0x55, 0x0a,
	0xe1, 0x12, 0x70, 0x46, 0x32, 0xd8, 0x33, 0x2f, 0x2d, 0x5f, 0x48, 0x99, 0x8b, 0x37, 0x37, 0xb5,
	0x28, 0x3b, 0x27, 0x35, 0xbe, 0xa0, 0x28, 0x35, 0x2d, 0xb3, 0x42, 0x82, 0x51, 0x81, 0x59, 0x83,
	0x27, 0x88, 0x07, 0x22, 0x18, 0x00, 0x16, 0x13, 0x92, 0xe6, 0xe2, 0x84, 0x58, 0x1a, 0x9f, 0x99,
	0x22, 0xc1, 0xa4, 0xc0, 0xa8, 0xc1, 0x19, 0xc4, 0x01, 0x11, 0xf0, 0x4c, 0x51, 0xba, 0xcc, 0xc8,
	0x25, 0xee, 0x5b, 0x9c, 0x1e, 0x94, 0x9a, 0x9e, 0x59, 0x5c, 0x92, 0x5a, 0x84, 0x6c, 0x03, 0xaa,
	0x46, 0x46, 0x54, 0x8d, 0x42, 0x36, 0x5c, 0x52, 0xc8, 0xfe, 0x8c, 0x47, 0x75, 0x07, 0x13, 0xd8,
	0x1d, 0x12, 0xc8, 0x2a, 0x7c, 0x91, 0xdd, 0x64, 0xc2, 0x25, 0x86, 0xa2, 0x1b, 0x61, 0x0f, 0x33,
	0xd8, 0x1e, 0x11, 0x64, 0x59, 0x67, 0x98, 0x9d, 0x62, 0x5c, 0x6c, 0xc5, 0x99, 0xe9, 0x79, 0xa9,
	0x45, 0x12, 0x2c, 0x60, 0x55, 0x50, 0x9e, 0x15, 0x7f, 0xc7, 0x02, 0x79, 0x86, 0xa6, 0xe7, 0x1b,
	0xb4, 0xa0, 0x02, 0x4a, 0x8a, 0x5c, 0xf2, 0x38, 0x3c, 0x15, 0x94, 0x5a, 0x5c, 0x90, 0x9f, 0x57,
	0x9c, 0x6a, 0x34, 0x89, 0x91, 0x8b, 0x1f, 0x59, 0xc2, 0xb7, 0x38, 0x5d, 0xa8, 0x82, 0x4b, 0x04,
	0x6b, 0x40, 0x68, 0xeb, 0x61, 0x46, 0x9f, 0x1e, 0x0e, 0x0b, 0xa4, 0x8c, 0x49, 0x50, 0x0c, 0x73,
	0x8d, 0x14, 0x6b, 0xc3, 0xf3, 0x0d, 0x5a, 0x8c, 0x4e, 0x41, 0x27, 0x1e, 0xc9, 0x31, 0x5e, 0x78,
	0x24, 0xc7, 0xf8, 0xe0, 0x91, 0x1c, 0xe3, 0x84, 0xc7, 0x72, 0x0c, 0x17, 0x1e, 0xcb, 0x31, 0xdc,
	0x78, 0x2c, 0xc7, 0x10, 0x65, 0x91, 0x9e, 0x59, 0x92, 0x51, 0x9a, 0xa4, 0x97, 0x9c, 0x9f, 0xab,
	0x0f, 0x4d, 0x37, 0x99, 0x49, 0xc9, 0xba, 0xe9, 0xf9, 0xfa, 0x65, 0x96, 0xfa, 0xb9, 0xf9, 0x29,
	0xa5, 0x39, 0xa9, 0xc5, 0x90, 0x74, 0x68, 0x60, 0xa4, 0x0b, 0x4d, 0x8a, 0x25, 0x95, 0x05, 0xa9,
	0xc5, 0x49, 0x6c, 0xe0, 0xe4, 0x63, 0x0c, 0x08, 0x00, 0x00, 0xff, 0xff, 0xda, 0x00, 0x13, 0xe2,
	0xaa, 0x02, 0x00, 0x00,
}

// Reference imports to suppress errors if they are not otherwise used.
var _ context.Context
var _ grpc.ClientConn

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc.SupportPackageIsVersion4

// CounterpartyMsgClient is the client API for CounterpartyMsg service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://godoc.org/google.golang.org/grpc#ClientConn.NewStream.
type CounterpartyMsgClient interface {
	// RegisterCounterparty defines a rpc handler method for MsgRegisterCounterparty.
	RegisterCounterparty(ctx context.Context, in *MsgRegisterCounterparty, opts ...grpc.CallOption) (*MsgRegisterCounterpartyResponse, error)
}

type counterpartyMsgClient struct {
	cc grpc1.ClientConn
}

func NewCounterpartyMsgClient(cc grpc1.ClientConn) CounterpartyMsgClient {
	return &counterpartyMsgClient{cc}
}

func (c *counterpartyMsgClient) RegisterCounterparty(ctx context.Context, in *MsgRegisterCounterparty, opts ...grpc.CallOption) (*MsgRegisterCounterpartyResponse, error) {
	out := new(MsgRegisterCounterpartyResponse)
	err := c.cc.Invoke(ctx, "/ibc.core.client.v2.CounterpartyMsg/RegisterCounterparty", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// CounterpartyMsgServer is the server API for CounterpartyMsg service.
type CounterpartyMsgServer interface {
	// RegisterCounterparty defines a rpc handler method for MsgRegisterCounterparty.
	RegisterCounterparty(context.Context, *MsgRegisterCounterparty) (*MsgRegisterCounterpartyResponse, error)
}

// UnimplementedCounterpartyMsgServer can be embedded to have forward compatible implementations.
type UnimplementedCounterpartyMsgServer struct {
}

func (*UnimplementedCounterpartyMsgServer) RegisterCounterparty(ctx context.Context, req *MsgRegisterCounterparty) (*MsgRegisterCounterpartyResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method RegisterCounterparty not implemented")
}

func RegisterCounterpartyMsgServer(s grpc1.Server, srv CounterpartyMsgServer) {
	s.RegisterService(&_CounterpartyMsg_serviceDesc, srv)
}

func _CounterpartyMsg_RegisterCounterparty_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgRegisterCounterparty)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(CounterpartyMsgServer).RegisterCounterparty(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/ibc.core.client.v2.CounterpartyMsg/RegisterCounterparty",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(CounterpartyMsgServer).RegisterCounterparty(ctx, req.(*MsgRegisterCounterparty))
	}
	return interceptor(ctx, in, info, handler)
}

var _CounterpartyMsg_serviceDesc = grpc.ServiceDesc{
	ServiceName: "ibc.core.client.v2.CounterpartyMsg",
	HandlerType: (*CounterpartyMsgServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "RegisterCounterparty",
			Handler:    _CounterpartyMsg_RegisterCounterparty_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "ibc/core/client/v2/counterparty.proto",
}

func (m *CounterpartyInfo) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *CounterpartyInfo) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *CounterpartyInfo) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if len(m.ClientId) > 0 {
		i -= len(m.ClientId)
		copy(dAtA[i:], m.ClientId)
		i = encodeVarintCounterparty(dAtA, i, uint64(len(m.ClientId)))
		i--
		dAtA[i] = 0x12
	}
	if len(m.MerklePrefix) > 0 {
		for iNdEx := len(m.MerklePrefix) - 1; iNdEx >= 0; iNdEx-- {
			i -= len(m.MerklePrefix[iNdEx])
			copy(dAtA[i:], m.MerklePrefix[iNdEx])
			i = encodeVarintCounterparty(dAtA, i, uint64(len(m.MerklePrefix[iNdEx])))
			i--
			dAtA[i] = 0xa
		}
	}
	return len(dAtA) - i, nil
}

func (m *MsgRegisterCounterparty) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *MsgRegisterCounterparty) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *MsgRegisterCounterparty) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if len(m.Signer) > 0 {
		i -= len(m.Signer)
		copy(dAtA[i:], m.Signer)
		i = encodeVarintCounterparty(dAtA, i, uint64(len(m.Signer)))
		i--
		dAtA[i] = 0x22
	}
	if len(m.CounterpartyClientId) > 0 {
		i -= len(m.CounterpartyClientId)
		copy(dAtA[i:], m.CounterpartyClientId)
		i = encodeVarintCounterparty(dAtA, i, uint64(len(m.CounterpartyClientId)))
		i--
		dAtA[i] = 0x1a
	}
	if len(m.CounterpartyMerklePrefix) > 0 {
		for iNdEx := len(m.CounterpartyMerklePrefix) - 1; iNdEx >= 0; iNdEx-- {
			i -= len(m.CounterpartyMerklePrefix[iNdEx])
			copy(dAtA[i:], m.CounterpartyMerklePrefix[iNdEx])
			i = encodeVarintCounterparty(dAtA, i, uint64(len(m.CounterpartyMerklePrefix[iNdEx])))
			i--
			dAtA[i] = 0x12
		}
	}
	if len(m.ClientId) > 0 {
		i -= len(m.ClientId)
		copy(dAtA[i:], m.ClientId)
		i = encodeVarintCounterparty(dAtA, i, uint64(len(m.ClientId)))
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

func (m *MsgRegisterCounterpartyResponse) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *MsgRegisterCounterpartyResponse) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *MsgRegisterCounterpartyResponse) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	return len(dAtA) - i, nil
}

func encodeVarintCounterparty(dAtA []byte, offset int, v uint64) int {
	offset -= sovCounterparty(v)
	base := offset
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return base
}
func (m *CounterpartyInfo) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	if len(m.MerklePrefix) > 0 {
		for _, b := range m.MerklePrefix {
			l = len(b)
			n += 1 + l + sovCounterparty(uint64(l))
		}
	}
	l = len(m.ClientId)
	if l > 0 {
		n += 1 + l + sovCounterparty(uint64(l))
	}
	return n
}

func (m *MsgRegisterCounterparty) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.ClientId)
	if l > 0 {
		n += 1 + l + sovCounterparty(uint64(l))
	}
	if len(m.CounterpartyMerklePrefix) > 0 {
		for _, b := range m.CounterpartyMerklePrefix {
			l = len(b)
			n += 1 + l + sovCounterparty(uint64(l))
		}
	}
	l = len(m.CounterpartyClientId)
	if l > 0 {
		n += 1 + l + sovCounterparty(uint64(l))
	}
	l = len(m.Signer)
	if l > 0 {
		n += 1 + l + sovCounterparty(uint64(l))
	}
	return n
}

func (m *MsgRegisterCounterpartyResponse) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	return n
}

func sovCounterparty(x uint64) (n int) {
	return (math_bits.Len64(x|1) + 6) / 7
}
func sozCounterparty(x uint64) (n int) {
	return sovCounterparty(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (m *CounterpartyInfo) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowCounterparty
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
			return fmt.Errorf("proto: CounterpartyInfo: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: CounterpartyInfo: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field MerklePrefix", wireType)
			}
			var byteLen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowCounterparty
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				byteLen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if byteLen < 0 {
				return ErrInvalidLengthCounterparty
			}
			postIndex := iNdEx + byteLen
			if postIndex < 0 {
				return ErrInvalidLengthCounterparty
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.MerklePrefix = append(m.MerklePrefix, make([]byte, postIndex-iNdEx))
			copy(m.MerklePrefix[len(m.MerklePrefix)-1], dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field ClientId", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowCounterparty
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
				return ErrInvalidLengthCounterparty
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthCounterparty
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.ClientId = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipCounterparty(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthCounterparty
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
func (m *MsgRegisterCounterparty) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowCounterparty
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
			return fmt.Errorf("proto: MsgRegisterCounterparty: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: MsgRegisterCounterparty: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field ClientId", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowCounterparty
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
				return ErrInvalidLengthCounterparty
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthCounterparty
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.ClientId = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field CounterpartyMerklePrefix", wireType)
			}
			var byteLen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowCounterparty
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				byteLen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if byteLen < 0 {
				return ErrInvalidLengthCounterparty
			}
			postIndex := iNdEx + byteLen
			if postIndex < 0 {
				return ErrInvalidLengthCounterparty
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.CounterpartyMerklePrefix = append(m.CounterpartyMerklePrefix, make([]byte, postIndex-iNdEx))
			copy(m.CounterpartyMerklePrefix[len(m.CounterpartyMerklePrefix)-1], dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 3:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field CounterpartyClientId", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowCounterparty
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
				return ErrInvalidLengthCounterparty
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthCounterparty
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.CounterpartyClientId = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 4:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Signer", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowCounterparty
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
				return ErrInvalidLengthCounterparty
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthCounterparty
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Signer = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipCounterparty(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthCounterparty
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
func (m *MsgRegisterCounterpartyResponse) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowCounterparty
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
			return fmt.Errorf("proto: MsgRegisterCounterpartyResponse: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: MsgRegisterCounterpartyResponse: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		default:
			iNdEx = preIndex
			skippy, err := skipCounterparty(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthCounterparty
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
func skipCounterparty(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	depth := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowCounterparty
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
					return 0, ErrIntOverflowCounterparty
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
					return 0, ErrIntOverflowCounterparty
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
				return 0, ErrInvalidLengthCounterparty
			}
			iNdEx += length
		case 3:
			depth++
		case 4:
			if depth == 0 {
				return 0, ErrUnexpectedEndOfGroupCounterparty
			}
			depth--
		case 5:
			iNdEx += 4
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
		if iNdEx < 0 {
			return 0, ErrInvalidLengthCounterparty
		}
		if depth == 0 {
			return iNdEx, nil
		}
	}
	return 0, io.ErrUnexpectedEOF
}

var (
	ErrInvalidLengthCounterparty        = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowCounterparty          = fmt.Errorf("proto: integer overflow")
	ErrUnexpectedEndOfGroupCounterparty = fmt.Errorf("proto: unexpected end of group")
)
