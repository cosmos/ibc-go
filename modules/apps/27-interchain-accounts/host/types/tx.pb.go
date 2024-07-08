// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: ibc/applications/interchain_accounts/host/v1/tx.proto

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

// MsgUpdateParams defines the payload for Msg/UpdateParams
type MsgUpdateParams struct {
	// signer address
	Signer string `protobuf:"bytes,1,opt,name=signer,proto3" json:"signer,omitempty"`
	// params defines the 27-interchain-accounts/host parameters to update.
	//
	// NOTE: All parameters must be supplied.
	Params Params `protobuf:"bytes,2,opt,name=params,proto3" json:"params"`
}

func (m *MsgUpdateParams) Reset()         { *m = MsgUpdateParams{} }
func (m *MsgUpdateParams) String() string { return proto.CompactTextString(m) }
func (*MsgUpdateParams) ProtoMessage()    {}
func (*MsgUpdateParams) Descriptor() ([]byte, []int) {
	return fileDescriptor_fa437afde7f1e7ae, []int{0}
}
func (m *MsgUpdateParams) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *MsgUpdateParams) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_MsgUpdateParams.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *MsgUpdateParams) XXX_Merge(src proto.Message) {
	xxx_messageInfo_MsgUpdateParams.Merge(m, src)
}
func (m *MsgUpdateParams) XXX_Size() int {
	return m.Size()
}
func (m *MsgUpdateParams) XXX_DiscardUnknown() {
	xxx_messageInfo_MsgUpdateParams.DiscardUnknown(m)
}

var xxx_messageInfo_MsgUpdateParams proto.InternalMessageInfo

// MsgUpdateParamsResponse defines the response for Msg/UpdateParams
type MsgUpdateParamsResponse struct {
}

func (m *MsgUpdateParamsResponse) Reset()         { *m = MsgUpdateParamsResponse{} }
func (m *MsgUpdateParamsResponse) String() string { return proto.CompactTextString(m) }
func (*MsgUpdateParamsResponse) ProtoMessage()    {}
func (*MsgUpdateParamsResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_fa437afde7f1e7ae, []int{1}
}
func (m *MsgUpdateParamsResponse) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *MsgUpdateParamsResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_MsgUpdateParamsResponse.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *MsgUpdateParamsResponse) XXX_Merge(src proto.Message) {
	xxx_messageInfo_MsgUpdateParamsResponse.Merge(m, src)
}
func (m *MsgUpdateParamsResponse) XXX_Size() int {
	return m.Size()
}
func (m *MsgUpdateParamsResponse) XXX_DiscardUnknown() {
	xxx_messageInfo_MsgUpdateParamsResponse.DiscardUnknown(m)
}

var xxx_messageInfo_MsgUpdateParamsResponse proto.InternalMessageInfo

// MsgModuleQuerySafe defines the payload for Msg/ModuleQuerySafe
type MsgModuleQuerySafe struct {
	// signer address
	Signer string `protobuf:"bytes,1,opt,name=signer,proto3" json:"signer,omitempty"`
	// requests defines the module safe queries to execute.
	Requests []QueryRequest `protobuf:"bytes,2,rep,name=requests,proto3" json:"requests"`
}

func (m *MsgModuleQuerySafe) Reset()         { *m = MsgModuleQuerySafe{} }
func (m *MsgModuleQuerySafe) String() string { return proto.CompactTextString(m) }
func (*MsgModuleQuerySafe) ProtoMessage()    {}
func (*MsgModuleQuerySafe) Descriptor() ([]byte, []int) {
	return fileDescriptor_fa437afde7f1e7ae, []int{2}
}
func (m *MsgModuleQuerySafe) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *MsgModuleQuerySafe) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_MsgModuleQuerySafe.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *MsgModuleQuerySafe) XXX_Merge(src proto.Message) {
	xxx_messageInfo_MsgModuleQuerySafe.Merge(m, src)
}
func (m *MsgModuleQuerySafe) XXX_Size() int {
	return m.Size()
}
func (m *MsgModuleQuerySafe) XXX_DiscardUnknown() {
	xxx_messageInfo_MsgModuleQuerySafe.DiscardUnknown(m)
}

var xxx_messageInfo_MsgModuleQuerySafe proto.InternalMessageInfo

// MsgModuleQuerySafeResponse defines the response for Msg/ModuleQuerySafe
type MsgModuleQuerySafeResponse struct {
	// height at which the responses were queried
	Height uint64 `protobuf:"varint,1,opt,name=height,proto3" json:"height,omitempty"`
	// protobuf encoded responses for each query
	Responses [][]byte `protobuf:"bytes,2,rep,name=responses,proto3" json:"responses,omitempty"`
}

func (m *MsgModuleQuerySafeResponse) Reset()         { *m = MsgModuleQuerySafeResponse{} }
func (m *MsgModuleQuerySafeResponse) String() string { return proto.CompactTextString(m) }
func (*MsgModuleQuerySafeResponse) ProtoMessage()    {}
func (*MsgModuleQuerySafeResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_fa437afde7f1e7ae, []int{3}
}
func (m *MsgModuleQuerySafeResponse) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *MsgModuleQuerySafeResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_MsgModuleQuerySafeResponse.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *MsgModuleQuerySafeResponse) XXX_Merge(src proto.Message) {
	xxx_messageInfo_MsgModuleQuerySafeResponse.Merge(m, src)
}
func (m *MsgModuleQuerySafeResponse) XXX_Size() int {
	return m.Size()
}
func (m *MsgModuleQuerySafeResponse) XXX_DiscardUnknown() {
	xxx_messageInfo_MsgModuleQuerySafeResponse.DiscardUnknown(m)
}

var xxx_messageInfo_MsgModuleQuerySafeResponse proto.InternalMessageInfo

func (m *MsgModuleQuerySafeResponse) GetHeight() uint64 {
	if m != nil {
		return m.Height
	}
	return 0
}

func (m *MsgModuleQuerySafeResponse) GetResponses() [][]byte {
	if m != nil {
		return m.Responses
	}
	return nil
}

func init() {
	proto.RegisterType((*MsgUpdateParams)(nil), "ibc.applications.interchain_accounts.host.v1.MsgUpdateParams")
	proto.RegisterType((*MsgUpdateParamsResponse)(nil), "ibc.applications.interchain_accounts.host.v1.MsgUpdateParamsResponse")
	proto.RegisterType((*MsgModuleQuerySafe)(nil), "ibc.applications.interchain_accounts.host.v1.MsgModuleQuerySafe")
	proto.RegisterType((*MsgModuleQuerySafeResponse)(nil), "ibc.applications.interchain_accounts.host.v1.MsgModuleQuerySafeResponse")
}

func init() {
	proto.RegisterFile("ibc/applications/interchain_accounts/host/v1/tx.proto", fileDescriptor_fa437afde7f1e7ae)
}

var fileDescriptor_fa437afde7f1e7ae = []byte{
	// 457 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xa4, 0x93, 0x41, 0x6b, 0xd4, 0x40,
	0x14, 0xc7, 0x33, 0x6d, 0x5d, 0xec, 0xb4, 0x50, 0x08, 0x62, 0x6b, 0x90, 0xb4, 0xec, 0x69, 0x29,
	0xee, 0x0c, 0x5d, 0x95, 0x4a, 0x41, 0x90, 0x82, 0x20, 0xc2, 0x82, 0x8e, 0x78, 0x11, 0x41, 0x26,
	0xb3, 0xe3, 0x64, 0xa0, 0xc9, 0xc4, 0xbc, 0xc9, 0x62, 0x6f, 0xe2, 0xc9, 0x93, 0x78, 0xf0, 0x26,
	0x82, 0x1f, 0xa1, 0xdf, 0xc1, 0x4b, 0x8f, 0x3d, 0x7a, 0x12, 0xd9, 0x3d, 0xf4, 0x6b, 0x48, 0x66,
	0xa7, 0xad, 0x4d, 0xed, 0x21, 0xf4, 0x96, 0x49, 0xe6, 0xff, 0x7b, 0xbf, 0x97, 0x99, 0x87, 0xef,
	0xeb, 0x44, 0x50, 0x5e, 0x14, 0x7b, 0x5a, 0x70, 0xab, 0x4d, 0x0e, 0x54, 0xe7, 0x56, 0x96, 0x22,
	0xe5, 0x3a, 0x7f, 0xc3, 0x85, 0x30, 0x55, 0x6e, 0x81, 0xa6, 0x06, 0x2c, 0x1d, 0x6f, 0x51, 0xfb,
	0x9e, 0x14, 0xa5, 0xb1, 0x26, 0xbc, 0xa3, 0x13, 0x41, 0xfe, 0x8d, 0x91, 0xff, 0xc4, 0x48, 0x1d,
	0x23, 0xe3, 0xad, 0xe8, 0x86, 0x32, 0xca, 0xb8, 0x20, 0xad, 0x9f, 0x66, 0x8c, 0x68, 0x55, 0x18,
	0xc8, 0x0c, 0xd0, 0x0c, 0x54, 0xcd, 0xce, 0x40, 0xf9, 0x0f, 0xdb, 0xad, 0x9c, 0x5c, 0x11, 0x17,
	0xec, 0x7e, 0x46, 0x78, 0x65, 0x08, 0xea, 0x65, 0x31, 0xe2, 0x56, 0x3e, 0xe3, 0x25, 0xcf, 0x20,
	0xbc, 0x89, 0x3b, 0xa0, 0x55, 0x2e, 0xcb, 0x35, 0xb4, 0x81, 0x7a, 0x8b, 0xcc, 0xaf, 0x42, 0x86,
	0x3b, 0x85, 0xdb, 0xb1, 0x36, 0xb7, 0x81, 0x7a, 0x4b, 0x83, 0x7b, 0xa4, 0x4d, 0x4b, 0x64, 0x46,
	0xdf, 0x5d, 0x38, 0xfc, 0xbd, 0x1e, 0x30, 0x4f, 0xda, 0x59, 0xf9, 0xf4, 0x63, 0x3d, 0xf8, 0x78,
	0x7c, 0xb0, 0xe9, 0x8b, 0x74, 0x6f, 0xe1, 0xd5, 0x86, 0x0f, 0x93, 0x50, 0x98, 0x1c, 0x64, 0xf7,
	0x1b, 0xc2, 0xe1, 0x10, 0xd4, 0xd0, 0x8c, 0xaa, 0x3d, 0xf9, 0xbc, 0x92, 0xe5, 0xfe, 0x0b, 0xfe,
	0x56, 0x5e, 0xaa, 0xfb, 0x1a, 0x5f, 0x2f, 0xe5, 0xbb, 0x4a, 0x82, 0xad, 0x85, 0xe7, 0x7b, 0x4b,
	0x83, 0x9d, 0x76, 0xc2, 0xae, 0x04, 0x9b, 0x21, 0xbc, 0xf6, 0x29, 0xf1, 0xa2, 0x38, 0xc3, 0xd1,
	0x45, 0xb9, 0x13, 0xf7, 0x5a, 0x32, 0x95, 0x5a, 0xa5, 0xd6, 0x49, 0x2e, 0x30, 0xbf, 0x0a, 0x6f,
	0xe3, 0xc5, 0xd2, 0xef, 0x99, 0x59, 0x2e, 0xb3, 0xb3, 0x17, 0x83, 0x9f, 0x73, 0x78, 0x7e, 0x08,
	0x2a, 0xfc, 0x8a, 0xf0, 0xf2, 0xb9, 0x23, 0x7a, 0xd8, 0xae, 0x93, 0xc6, 0x1f, 0x8d, 0x1e, 0x5f,
	0x29, 0x7e, 0xda, 0xd4, 0xf7, 0xfa, 0xf2, 0x34, 0x4e, 0xe3, 0x51, 0x6b, 0x74, 0x83, 0x10, 0x3d,
	0xb9, 0x2a, 0xe1, 0xc4, 0x2f, 0xba, 0xf6, 0xe1, 0xf8, 0x60, 0x13, 0xed, 0x8e, 0x0e, 0x27, 0x31,
	0x3a, 0x9a, 0xc4, 0xe8, 0xcf, 0x24, 0x46, 0x5f, 0xa6, 0x71, 0x70, 0x34, 0x8d, 0x83, 0x5f, 0xd3,
	0x38, 0x78, 0xf5, 0x54, 0x69, 0x9b, 0x56, 0x09, 0x11, 0x26, 0xa3, 0x7e, 0xb4, 0x74, 0x22, 0xfa,
	0xca, 0xd0, 0xf1, 0x03, 0x9a, 0x39, 0x2a, 0xd4, 0x63, 0x05, 0x74, 0xb0, 0xdd, 0x3f, 0x93, 0xe8,
	0x9f, 0x9f, 0x28, 0xbb, 0x5f, 0x48, 0x48, 0x3a, 0x6e, 0xa0, 0xee, 0xfe, 0x0d, 0x00, 0x00, 0xff,
	0xff, 0xd3, 0x2f, 0xef, 0x8d, 0x1f, 0x04, 0x00, 0x00,
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
	// UpdateParams defines a rpc handler for MsgUpdateParams.
	UpdateParams(ctx context.Context, in *MsgUpdateParams, opts ...grpc.CallOption) (*MsgUpdateParamsResponse, error)
	// ModuleQuerySafe defines a rpc handler for MsgModuleQuerySafe.
	ModuleQuerySafe(ctx context.Context, in *MsgModuleQuerySafe, opts ...grpc.CallOption) (*MsgModuleQuerySafeResponse, error)
}

type msgClient struct {
	cc grpc1.ClientConn
}

func NewMsgClient(cc grpc1.ClientConn) MsgClient {
	return &msgClient{cc}
}

func (c *msgClient) UpdateParams(ctx context.Context, in *MsgUpdateParams, opts ...grpc.CallOption) (*MsgUpdateParamsResponse, error) {
	out := new(MsgUpdateParamsResponse)
	err := c.cc.Invoke(ctx, "/ibc.applications.interchain_accounts.host.v1.Msg/UpdateParams", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *msgClient) ModuleQuerySafe(ctx context.Context, in *MsgModuleQuerySafe, opts ...grpc.CallOption) (*MsgModuleQuerySafeResponse, error) {
	out := new(MsgModuleQuerySafeResponse)
	err := c.cc.Invoke(ctx, "/ibc.applications.interchain_accounts.host.v1.Msg/ModuleQuerySafe", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// MsgServer is the server API for Msg service.
type MsgServer interface {
	// UpdateParams defines a rpc handler for MsgUpdateParams.
	UpdateParams(context.Context, *MsgUpdateParams) (*MsgUpdateParamsResponse, error)
	// ModuleQuerySafe defines a rpc handler for MsgModuleQuerySafe.
	ModuleQuerySafe(context.Context, *MsgModuleQuerySafe) (*MsgModuleQuerySafeResponse, error)
}

// UnimplementedMsgServer can be embedded to have forward compatible implementations.
type UnimplementedMsgServer struct {
}

func (*UnimplementedMsgServer) UpdateParams(ctx context.Context, req *MsgUpdateParams) (*MsgUpdateParamsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method UpdateParams not implemented")
}
func (*UnimplementedMsgServer) ModuleQuerySafe(ctx context.Context, req *MsgModuleQuerySafe) (*MsgModuleQuerySafeResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ModuleQuerySafe not implemented")
}

func RegisterMsgServer(s grpc1.Server, srv MsgServer) {
	s.RegisterService(&_Msg_serviceDesc, srv)
}

func _Msg_UpdateParams_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgUpdateParams)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).UpdateParams(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/ibc.applications.interchain_accounts.host.v1.Msg/UpdateParams",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).UpdateParams(ctx, req.(*MsgUpdateParams))
	}
	return interceptor(ctx, in, info, handler)
}

func _Msg_ModuleQuerySafe_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgModuleQuerySafe)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).ModuleQuerySafe(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/ibc.applications.interchain_accounts.host.v1.Msg/ModuleQuerySafe",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).ModuleQuerySafe(ctx, req.(*MsgModuleQuerySafe))
	}
	return interceptor(ctx, in, info, handler)
}

var _Msg_serviceDesc = grpc.ServiceDesc{
	ServiceName: "ibc.applications.interchain_accounts.host.v1.Msg",
	HandlerType: (*MsgServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "UpdateParams",
			Handler:    _Msg_UpdateParams_Handler,
		},
		{
			MethodName: "ModuleQuerySafe",
			Handler:    _Msg_ModuleQuerySafe_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "ibc/applications/interchain_accounts/host/v1/tx.proto",
}

func (m *MsgUpdateParams) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *MsgUpdateParams) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *MsgUpdateParams) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	{
		size, err := m.Params.MarshalToSizedBuffer(dAtA[:i])
		if err != nil {
			return 0, err
		}
		i -= size
		i = encodeVarintTx(dAtA, i, uint64(size))
	}
	i--
	dAtA[i] = 0x12
	if len(m.Signer) > 0 {
		i -= len(m.Signer)
		copy(dAtA[i:], m.Signer)
		i = encodeVarintTx(dAtA, i, uint64(len(m.Signer)))
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

func (m *MsgUpdateParamsResponse) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *MsgUpdateParamsResponse) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *MsgUpdateParamsResponse) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	return len(dAtA) - i, nil
}

func (m *MsgModuleQuerySafe) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *MsgModuleQuerySafe) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *MsgModuleQuerySafe) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if len(m.Requests) > 0 {
		for iNdEx := len(m.Requests) - 1; iNdEx >= 0; iNdEx-- {
			{
				size, err := m.Requests[iNdEx].MarshalToSizedBuffer(dAtA[:i])
				if err != nil {
					return 0, err
				}
				i -= size
				i = encodeVarintTx(dAtA, i, uint64(size))
			}
			i--
			dAtA[i] = 0x12
		}
	}
	if len(m.Signer) > 0 {
		i -= len(m.Signer)
		copy(dAtA[i:], m.Signer)
		i = encodeVarintTx(dAtA, i, uint64(len(m.Signer)))
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

func (m *MsgModuleQuerySafeResponse) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *MsgModuleQuerySafeResponse) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *MsgModuleQuerySafeResponse) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if len(m.Responses) > 0 {
		for iNdEx := len(m.Responses) - 1; iNdEx >= 0; iNdEx-- {
			i -= len(m.Responses[iNdEx])
			copy(dAtA[i:], m.Responses[iNdEx])
			i = encodeVarintTx(dAtA, i, uint64(len(m.Responses[iNdEx])))
			i--
			dAtA[i] = 0x12
		}
	}
	if m.Height != 0 {
		i = encodeVarintTx(dAtA, i, uint64(m.Height))
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
func (m *MsgUpdateParams) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.Signer)
	if l > 0 {
		n += 1 + l + sovTx(uint64(l))
	}
	l = m.Params.Size()
	n += 1 + l + sovTx(uint64(l))
	return n
}

func (m *MsgUpdateParamsResponse) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	return n
}

func (m *MsgModuleQuerySafe) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.Signer)
	if l > 0 {
		n += 1 + l + sovTx(uint64(l))
	}
	if len(m.Requests) > 0 {
		for _, e := range m.Requests {
			l = e.Size()
			n += 1 + l + sovTx(uint64(l))
		}
	}
	return n
}

func (m *MsgModuleQuerySafeResponse) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	if m.Height != 0 {
		n += 1 + sovTx(uint64(m.Height))
	}
	if len(m.Responses) > 0 {
		for _, b := range m.Responses {
			l = len(b)
			n += 1 + l + sovTx(uint64(l))
		}
	}
	return n
}

func sovTx(x uint64) (n int) {
	return (math_bits.Len64(x|1) + 6) / 7
}
func sozTx(x uint64) (n int) {
	return sovTx(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (m *MsgUpdateParams) Unmarshal(dAtA []byte) error {
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
			return fmt.Errorf("proto: MsgUpdateParams: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: MsgUpdateParams: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Signer", wireType)
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
			m.Signer = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Params", wireType)
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
			if err := m.Params.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
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
func (m *MsgUpdateParamsResponse) Unmarshal(dAtA []byte) error {
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
			return fmt.Errorf("proto: MsgUpdateParamsResponse: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: MsgUpdateParamsResponse: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
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
func (m *MsgModuleQuerySafe) Unmarshal(dAtA []byte) error {
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
			return fmt.Errorf("proto: MsgModuleQuerySafe: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: MsgModuleQuerySafe: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Signer", wireType)
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
			m.Signer = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Requests", wireType)
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
			m.Requests = append(m.Requests, QueryRequest{})
			if err := m.Requests[len(m.Requests)-1].Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
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
func (m *MsgModuleQuerySafeResponse) Unmarshal(dAtA []byte) error {
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
			return fmt.Errorf("proto: MsgModuleQuerySafeResponse: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: MsgModuleQuerySafeResponse: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Height", wireType)
			}
			m.Height = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTx
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.Height |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Responses", wireType)
			}
			var byteLen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTx
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
				return ErrInvalidLengthTx
			}
			postIndex := iNdEx + byteLen
			if postIndex < 0 {
				return ErrInvalidLengthTx
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Responses = append(m.Responses, make([]byte, postIndex-iNdEx))
			copy(m.Responses[len(m.Responses)-1], dAtA[iNdEx:postIndex])
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