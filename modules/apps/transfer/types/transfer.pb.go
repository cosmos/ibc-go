// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: ibc/applications/transfer/v1/transfer.proto

package types

import (
	fmt "fmt"
	_ "github.com/cosmos/gogoproto/gogoproto"
	proto "github.com/cosmos/gogoproto/proto"
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

// Params defines the set of IBC transfer parameters.
// NOTE: To prevent a single token from being transferred, set the
// TransfersEnabled parameter to true and then set the bank module's SendEnabled
// parameter for the denomination to false.
type Params struct {
	// send_enabled enables or disables all cross-chain token transfers from this
	// chain.
	SendEnabled bool `protobuf:"varint,1,opt,name=send_enabled,json=sendEnabled,proto3" json:"send_enabled,omitempty"`
	// receive_enabled enables or disables all cross-chain token transfers to this
	// chain.
	ReceiveEnabled bool `protobuf:"varint,2,opt,name=receive_enabled,json=receiveEnabled,proto3" json:"receive_enabled,omitempty"`
}

func (m *Params) Reset()         { *m = Params{} }
func (m *Params) String() string { return proto.CompactTextString(m) }
func (*Params) ProtoMessage()    {}
func (*Params) Descriptor() ([]byte, []int) {
	return fileDescriptor_5041673e96e97901, []int{0}
}
func (m *Params) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *Params) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_Params.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *Params) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Params.Merge(m, src)
}
func (m *Params) XXX_Size() int {
	return m.Size()
}
func (m *Params) XXX_DiscardUnknown() {
	xxx_messageInfo_Params.DiscardUnknown(m)
}

var xxx_messageInfo_Params proto.InternalMessageInfo

func (m *Params) GetSendEnabled() bool {
	if m != nil {
		return m.SendEnabled
	}
	return false
}

func (m *Params) GetReceiveEnabled() bool {
	if m != nil {
		return m.ReceiveEnabled
	}
	return false
}

// Forwarding defines a list of port ID, channel ID pairs determining the path
// through which a packet must be forwarded, and an unwind boolean indicating if
// the coin should be unwinded to its native chain before forwarding.
type Forwarding struct {
	// optional unwinding for the token transfered
	Unwind bool `protobuf:"varint,1,opt,name=unwind,proto3" json:"unwind,omitempty"`
	// intermediate path through which packet will be forwarded
	Hops []Hop `protobuf:"bytes,2,rep,name=hops,proto3" json:"hops"`
}

func (m *Forwarding) Reset()         { *m = Forwarding{} }
func (m *Forwarding) String() string { return proto.CompactTextString(m) }
func (*Forwarding) ProtoMessage()    {}
func (*Forwarding) Descriptor() ([]byte, []int) {
	return fileDescriptor_5041673e96e97901, []int{1}
}
func (m *Forwarding) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *Forwarding) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_Forwarding.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *Forwarding) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Forwarding.Merge(m, src)
}
func (m *Forwarding) XXX_Size() int {
	return m.Size()
}
func (m *Forwarding) XXX_DiscardUnknown() {
	xxx_messageInfo_Forwarding.DiscardUnknown(m)
}

var xxx_messageInfo_Forwarding proto.InternalMessageInfo

func (m *Forwarding) GetUnwind() bool {
	if m != nil {
		return m.Unwind
	}
	return false
}

func (m *Forwarding) GetHops() []Hop {
	if m != nil {
		return m.Hops
	}
	return nil
}

// Hop defines a port ID, channel ID pair specifying where tokens must be forwarded
// next in a multihop transfer.
type Hop struct {
	PortId    string `protobuf:"bytes,1,opt,name=port_id,json=portId,proto3" json:"port_id,omitempty"`
	ChannelId string `protobuf:"bytes,2,opt,name=channel_id,json=channelId,proto3" json:"channel_id,omitempty"`
}

func (m *Hop) Reset()         { *m = Hop{} }
func (m *Hop) String() string { return proto.CompactTextString(m) }
func (*Hop) ProtoMessage()    {}
func (*Hop) Descriptor() ([]byte, []int) {
	return fileDescriptor_5041673e96e97901, []int{2}
}
func (m *Hop) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *Hop) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_Hop.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *Hop) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Hop.Merge(m, src)
}
func (m *Hop) XXX_Size() int {
	return m.Size()
}
func (m *Hop) XXX_DiscardUnknown() {
	xxx_messageInfo_Hop.DiscardUnknown(m)
}

var xxx_messageInfo_Hop proto.InternalMessageInfo

func (m *Hop) GetPortId() string {
	if m != nil {
		return m.PortId
	}
	return ""
}

func (m *Hop) GetChannelId() string {
	if m != nil {
		return m.ChannelId
	}
	return ""
}

func init() {
	proto.RegisterType((*Params)(nil), "ibc.applications.transfer.v1.Params")
	proto.RegisterType((*Forwarding)(nil), "ibc.applications.transfer.v1.Forwarding")
	proto.RegisterType((*Hop)(nil), "ibc.applications.transfer.v1.Hop")
}

func init() {
	proto.RegisterFile("ibc/applications/transfer/v1/transfer.proto", fileDescriptor_5041673e96e97901)
}

var fileDescriptor_5041673e96e97901 = []byte{
	// 323 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x7c, 0x91, 0x41, 0x4b, 0xc3, 0x30,
	0x14, 0xc7, 0xdb, 0x6d, 0x54, 0x97, 0x89, 0x42, 0x10, 0x1d, 0xa2, 0x75, 0xdb, 0xc5, 0x81, 0xd8,
	0x30, 0x3d, 0x28, 0x88, 0x97, 0x81, 0xb2, 0xdd, 0x74, 0x78, 0xf2, 0x32, 0xd2, 0x34, 0x76, 0x81,
	0x36, 0x2f, 0x24, 0x59, 0x87, 0xdf, 0xc2, 0x8f, 0xb5, 0xe3, 0x8e, 0x9e, 0x44, 0xb6, 0x2f, 0x22,
	0xed, 0xea, 0xd8, 0xc9, 0xdb, 0x7b, 0xbf, 0xf7, 0x7b, 0x8f, 0x90, 0x3f, 0xba, 0x14, 0x21, 0x23,
	0x54, 0xa9, 0x44, 0x30, 0x6a, 0x05, 0x48, 0x43, 0xac, 0xa6, 0xd2, 0xbc, 0x73, 0x4d, 0xb2, 0xde,
	0xa6, 0x0e, 0x94, 0x06, 0x0b, 0xf8, 0x54, 0x84, 0x2c, 0xd8, 0x96, 0x83, 0x8d, 0x90, 0xf5, 0x4e,
	0x0e, 0x63, 0x88, 0xa1, 0x10, 0x49, 0x5e, 0xad, 0x77, 0x3a, 0xaf, 0xc8, 0x7b, 0xa6, 0x9a, 0xa6,
	0x06, 0xb7, 0xd1, 0x9e, 0xe1, 0x32, 0x1a, 0x73, 0x49, 0xc3, 0x84, 0x47, 0x4d, 0xb7, 0xe5, 0x76,
	0x77, 0x47, 0x8d, 0x9c, 0x3d, 0xae, 0x11, 0xbe, 0x40, 0x07, 0x9a, 0x33, 0x2e, 0x32, 0xbe, 0xb1,
	0x2a, 0x85, 0xb5, 0x5f, 0xe2, 0x52, 0xec, 0x50, 0x84, 0x9e, 0x40, 0xcf, 0xa8, 0x8e, 0x84, 0x8c,
	0xf1, 0x11, 0xf2, 0xa6, 0x72, 0x26, 0xe4, 0xdf, 0xcd, 0xb2, 0xc3, 0xf7, 0xa8, 0x36, 0x01, 0x65,
	0x9a, 0x95, 0x56, 0xb5, 0xdb, 0xb8, 0x6e, 0x07, 0xff, 0x3d, 0x3f, 0x18, 0x80, 0xea, 0xd7, 0xe6,
	0xdf, 0xe7, 0xce, 0xa8, 0x58, 0xea, 0x3c, 0xa0, 0xea, 0x00, 0x14, 0x3e, 0x46, 0x3b, 0x0a, 0xb4,
	0x1d, 0x8b, 0xf5, 0xf1, 0xfa, 0xc8, 0xcb, 0xdb, 0x61, 0x84, 0xcf, 0x10, 0x62, 0x13, 0x2a, 0x25,
	0x4f, 0xf2, 0x59, 0xa5, 0x98, 0xd5, 0x4b, 0x32, 0x8c, 0xfa, 0x2f, 0xf3, 0xa5, 0xef, 0x2e, 0x96,
	0xbe, 0xfb, 0xb3, 0xf4, 0xdd, 0xcf, 0x95, 0xef, 0x2c, 0x56, 0xbe, 0xf3, 0xb5, 0xf2, 0x9d, 0xb7,
	0xdb, 0x58, 0xd8, 0xc9, 0x34, 0x0c, 0x18, 0xa4, 0x84, 0x81, 0x49, 0xc1, 0x10, 0x11, 0xb2, 0xab,
	0x18, 0x48, 0x76, 0x47, 0x52, 0x88, 0xa6, 0x09, 0x37, 0x79, 0x24, 0x5b, 0x51, 0xd8, 0x0f, 0xc5,
	0x4d, 0xe8, 0x15, 0x3f, 0x7a, 0xf3, 0x1b, 0x00, 0x00, 0xff, 0xff, 0x94, 0x05, 0x3d, 0x15, 0xb4,
	0x01, 0x00, 0x00,
}

func (m *Params) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *Params) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *Params) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if m.ReceiveEnabled {
		i--
		if m.ReceiveEnabled {
			dAtA[i] = 1
		} else {
			dAtA[i] = 0
		}
		i--
		dAtA[i] = 0x10
	}
	if m.SendEnabled {
		i--
		if m.SendEnabled {
			dAtA[i] = 1
		} else {
			dAtA[i] = 0
		}
		i--
		dAtA[i] = 0x8
	}
	return len(dAtA) - i, nil
}

func (m *Forwarding) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *Forwarding) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *Forwarding) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if len(m.Hops) > 0 {
		for iNdEx := len(m.Hops) - 1; iNdEx >= 0; iNdEx-- {
			{
				size, err := m.Hops[iNdEx].MarshalToSizedBuffer(dAtA[:i])
				if err != nil {
					return 0, err
				}
				i -= size
				i = encodeVarintTransfer(dAtA, i, uint64(size))
			}
			i--
			dAtA[i] = 0x12
		}
	}
	if m.Unwind {
		i--
		if m.Unwind {
			dAtA[i] = 1
		} else {
			dAtA[i] = 0
		}
		i--
		dAtA[i] = 0x8
	}
	return len(dAtA) - i, nil
}

func (m *Hop) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *Hop) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *Hop) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if len(m.ChannelId) > 0 {
		i -= len(m.ChannelId)
		copy(dAtA[i:], m.ChannelId)
		i = encodeVarintTransfer(dAtA, i, uint64(len(m.ChannelId)))
		i--
		dAtA[i] = 0x12
	}
	if len(m.PortId) > 0 {
		i -= len(m.PortId)
		copy(dAtA[i:], m.PortId)
		i = encodeVarintTransfer(dAtA, i, uint64(len(m.PortId)))
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

func encodeVarintTransfer(dAtA []byte, offset int, v uint64) int {
	offset -= sovTransfer(v)
	base := offset
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return base
}
func (m *Params) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	if m.SendEnabled {
		n += 2
	}
	if m.ReceiveEnabled {
		n += 2
	}
	return n
}

func (m *Forwarding) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	if m.Unwind {
		n += 2
	}
	if len(m.Hops) > 0 {
		for _, e := range m.Hops {
			l = e.Size()
			n += 1 + l + sovTransfer(uint64(l))
		}
	}
	return n
}

func (m *Hop) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.PortId)
	if l > 0 {
		n += 1 + l + sovTransfer(uint64(l))
	}
	l = len(m.ChannelId)
	if l > 0 {
		n += 1 + l + sovTransfer(uint64(l))
	}
	return n
}

func sovTransfer(x uint64) (n int) {
	return (math_bits.Len64(x|1) + 6) / 7
}
func sozTransfer(x uint64) (n int) {
	return sovTransfer(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (m *Params) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowTransfer
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
			return fmt.Errorf("proto: Params: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: Params: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field SendEnabled", wireType)
			}
			var v int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTransfer
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				v |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			m.SendEnabled = bool(v != 0)
		case 2:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field ReceiveEnabled", wireType)
			}
			var v int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTransfer
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				v |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			m.ReceiveEnabled = bool(v != 0)
		default:
			iNdEx = preIndex
			skippy, err := skipTransfer(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthTransfer
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
func (m *Forwarding) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowTransfer
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
			return fmt.Errorf("proto: Forwarding: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: Forwarding: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Unwind", wireType)
			}
			var v int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTransfer
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				v |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			m.Unwind = bool(v != 0)
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Hops", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTransfer
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
				return ErrInvalidLengthTransfer
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthTransfer
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Hops = append(m.Hops, Hop{})
			if err := m.Hops[len(m.Hops)-1].Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipTransfer(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthTransfer
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
func (m *Hop) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowTransfer
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
			return fmt.Errorf("proto: Hop: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: Hop: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field PortId", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTransfer
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
				return ErrInvalidLengthTransfer
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthTransfer
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.PortId = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field ChannelId", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTransfer
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
				return ErrInvalidLengthTransfer
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthTransfer
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.ChannelId = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipTransfer(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthTransfer
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
func skipTransfer(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	depth := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowTransfer
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
					return 0, ErrIntOverflowTransfer
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
					return 0, ErrIntOverflowTransfer
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
				return 0, ErrInvalidLengthTransfer
			}
			iNdEx += length
		case 3:
			depth++
		case 4:
			if depth == 0 {
				return 0, ErrUnexpectedEndOfGroupTransfer
			}
			depth--
		case 5:
			iNdEx += 4
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
		if iNdEx < 0 {
			return 0, ErrInvalidLengthTransfer
		}
		if depth == 0 {
			return iNdEx, nil
		}
	}
	return 0, io.ErrUnexpectedEOF
}

var (
	ErrInvalidLengthTransfer        = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowTransfer          = fmt.Errorf("proto: integer overflow")
	ErrUnexpectedEndOfGroupTransfer = fmt.Errorf("proto: unexpected end of group")
)
