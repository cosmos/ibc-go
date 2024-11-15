// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: ibc/core/channel/v2/genesis.proto

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

// GenesisState defines the ibc channel submodule's genesis state.
type GenesisState struct {
	Channels         []IdentifiedChannel `protobuf:"bytes,1,rep,name=channels,proto3,casttype=IdentifiedChannel" json:"channels"`
	Acknowledgements []PacketState       `protobuf:"bytes,2,rep,name=acknowledgements,proto3" json:"acknowledgements"`
	Commitments      []PacketState       `protobuf:"bytes,3,rep,name=commitments,proto3" json:"commitments"`
	Receipts         []PacketState       `protobuf:"bytes,4,rep,name=receipts,proto3" json:"receipts"`
	SendSequences    []PacketSequence    `protobuf:"bytes,5,rep,name=send_sequences,json=sendSequences,proto3" json:"send_sequences"`
}

func (m *GenesisState) Reset()         { *m = GenesisState{} }
func (m *GenesisState) String() string { return proto.CompactTextString(m) }
func (*GenesisState) ProtoMessage()    {}
func (*GenesisState) Descriptor() ([]byte, []int) {
	return fileDescriptor_b5d374f126f051c3, []int{0}
}
func (m *GenesisState) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *GenesisState) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_GenesisState.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *GenesisState) XXX_Merge(src proto.Message) {
	xxx_messageInfo_GenesisState.Merge(m, src)
}
func (m *GenesisState) XXX_Size() int {
	return m.Size()
}
func (m *GenesisState) XXX_DiscardUnknown() {
	xxx_messageInfo_GenesisState.DiscardUnknown(m)
}

var xxx_messageInfo_GenesisState proto.InternalMessageInfo

func (m *GenesisState) GetChannels() []IdentifiedChannel {
	if m != nil {
		return m.Channels
	}
	return nil
}

func (m *GenesisState) GetAcknowledgements() []PacketState {
	if m != nil {
		return m.Acknowledgements
	}
	return nil
}

func (m *GenesisState) GetCommitments() []PacketState {
	if m != nil {
		return m.Commitments
	}
	return nil
}

func (m *GenesisState) GetReceipts() []PacketState {
	if m != nil {
		return m.Receipts
	}
	return nil
}

func (m *GenesisState) GetSendSequences() []PacketSequence {
	if m != nil {
		return m.SendSequences
	}
	return nil
}

// PacketState defines the generic type necessary to retrieve and store
// packet commitments, acknowledgements, and receipts.
// Caller is responsible for knowing the context necessary to interpret this
// state as a commitment, acknowledgement, or a receipt.
type PacketState struct {
	// channel unique identifier.
	ChannelId string `protobuf:"bytes,1,opt,name=channel_id,json=channelId,proto3" json:"channel_id,omitempty"`
	// packet sequence.
	Sequence uint64 `protobuf:"varint,2,opt,name=sequence,proto3" json:"sequence,omitempty"`
	// embedded data that represents packet state.
	Data []byte `protobuf:"bytes,3,opt,name=data,proto3" json:"data,omitempty"`
}

func (m *PacketState) Reset()         { *m = PacketState{} }
func (m *PacketState) String() string { return proto.CompactTextString(m) }
func (*PacketState) ProtoMessage()    {}
func (*PacketState) Descriptor() ([]byte, []int) {
	return fileDescriptor_b5d374f126f051c3, []int{1}
}
func (m *PacketState) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *PacketState) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_PacketState.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *PacketState) XXX_Merge(src proto.Message) {
	xxx_messageInfo_PacketState.Merge(m, src)
}
func (m *PacketState) XXX_Size() int {
	return m.Size()
}
func (m *PacketState) XXX_DiscardUnknown() {
	xxx_messageInfo_PacketState.DiscardUnknown(m)
}

var xxx_messageInfo_PacketState proto.InternalMessageInfo

// PacketSequence defines the genesis type necessary to retrieve and store next send sequences.
type PacketSequence struct {
	// channel unique identifier.
	ChannelId string `protobuf:"bytes,1,opt,name=channel_id,json=channelId,proto3" json:"channel_id,omitempty"`
	// packet sequence
	Sequence uint64 `protobuf:"varint,2,opt,name=sequence,proto3" json:"sequence,omitempty"`
}

func (m *PacketSequence) Reset()         { *m = PacketSequence{} }
func (m *PacketSequence) String() string { return proto.CompactTextString(m) }
func (*PacketSequence) ProtoMessage()    {}
func (*PacketSequence) Descriptor() ([]byte, []int) {
	return fileDescriptor_b5d374f126f051c3, []int{2}
}
func (m *PacketSequence) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *PacketSequence) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_PacketSequence.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *PacketSequence) XXX_Merge(src proto.Message) {
	xxx_messageInfo_PacketSequence.Merge(m, src)
}
func (m *PacketSequence) XXX_Size() int {
	return m.Size()
}
func (m *PacketSequence) XXX_DiscardUnknown() {
	xxx_messageInfo_PacketSequence.DiscardUnknown(m)
}

var xxx_messageInfo_PacketSequence proto.InternalMessageInfo

func (m *PacketSequence) GetChannelId() string {
	if m != nil {
		return m.ChannelId
	}
	return ""
}

func (m *PacketSequence) GetSequence() uint64 {
	if m != nil {
		return m.Sequence
	}
	return 0
}

func init() {
	proto.RegisterType((*GenesisState)(nil), "ibc.core.channel.v2.GenesisState")
	proto.RegisterType((*PacketState)(nil), "ibc.core.channel.v2.PacketState")
	proto.RegisterType((*PacketSequence)(nil), "ibc.core.channel.v2.PacketSequence")
}

func init() { proto.RegisterFile("ibc/core/channel/v2/genesis.proto", fileDescriptor_b5d374f126f051c3) }

var fileDescriptor_b5d374f126f051c3 = []byte{
	// 414 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xa4, 0x92, 0xbf, 0xae, 0xd3, 0x30,
	0x14, 0xc6, 0xe3, 0x9b, 0x80, 0x7a, 0xdd, 0xcb, 0x15, 0x18, 0x86, 0x50, 0x89, 0x34, 0x14, 0x09,
	0x65, 0xb9, 0x31, 0x2a, 0x2c, 0x20, 0xa6, 0x30, 0x40, 0xc5, 0x52, 0x05, 0x89, 0x01, 0x09, 0x55,
	0x89, 0x7d, 0x48, 0xad, 0x36, 0x76, 0xa9, 0xdd, 0x22, 0xde, 0x80, 0x91, 0x47, 0x80, 0xb7, 0xe9,
	0xd8, 0x91, 0xa9, 0x42, 0xed, 0x5b, 0x30, 0xa1, 0xfc, 0x69, 0x55, 0xd4, 0x0a, 0xa9, 0xba, 0x9b,
	0x7d, 0xfc, 0x7d, 0xbf, 0x9f, 0x87, 0x83, 0x1f, 0x8a, 0x94, 0x51, 0xa6, 0xa6, 0x40, 0xd9, 0x30,
	0x91, 0x12, 0xc6, 0x74, 0xde, 0xa5, 0x19, 0x48, 0xd0, 0x42, 0x87, 0x93, 0xa9, 0x32, 0x8a, 0xdc,
	0x15, 0x29, 0x0b, 0x8b, 0x48, 0x58, 0x47, 0xc2, 0x79, 0xb7, 0x75, 0x2f, 0x53, 0x99, 0x2a, 0xdf,
	0x69, 0x71, 0xaa, 0xa2, 0xad, 0xa3, 0xb4, 0x6d, 0xab, 0x8c, 0x74, 0x7e, 0xda, 0xf8, 0xe2, 0x75,
	0xc5, 0x7f, 0x67, 0x12, 0x03, 0xe4, 0x23, 0x6e, 0xd4, 0x09, 0xed, 0x22, 0xdf, 0x0e, 0x9a, 0xdd,
	0xc7, 0xe1, 0x11, 0x63, 0xd8, 0xe3, 0x20, 0x8d, 0xf8, 0x24, 0x80, 0xbf, 0xaa, 0x86, 0xd1, 0xfd,
	0xc5, 0xaa, 0x6d, 0xfd, 0x59, 0xb5, 0xef, 0x1c, 0x3c, 0xc5, 0x3b, 0x24, 0x89, 0xf1, 0xed, 0x84,
	0x8d, 0xa4, 0xfa, 0x32, 0x06, 0x9e, 0x41, 0x0e, 0xd2, 0x68, 0xf7, 0xac, 0xd4, 0xf8, 0x47, 0x35,
	0xfd, 0x84, 0x8d, 0xc0, 0x94, 0x5f, 0x8b, 0x9c, 0x42, 0x10, 0x1f, 0xf4, 0xc9, 0x1b, 0xdc, 0x64,
	0x2a, 0xcf, 0x85, 0xa9, 0x70, 0xf6, 0x49, 0xb8, 0xfd, 0x2a, 0x89, 0x70, 0x63, 0x0a, 0x0c, 0xc4,
	0xc4, 0x68, 0xd7, 0x39, 0x09, 0xb3, 0xeb, 0x91, 0x3e, 0xbe, 0xd4, 0x20, 0xf9, 0x40, 0xc3, 0xe7,
	0x19, 0x48, 0x06, 0xda, 0xbd, 0x51, 0x92, 0x1e, 0xfd, 0x8f, 0x54, 0x67, 0x6b, 0xd8, 0xad, 0x02,
	0xb0, 0x9d, 0xe9, 0x4e, 0x8a, 0x9b, 0x7b, 0x42, 0xf2, 0x00, 0xe3, 0x1a, 0x30, 0x10, 0xdc, 0x45,
	0x3e, 0x0a, 0xce, 0xe3, 0xf3, 0x7a, 0xd2, 0xe3, 0xa4, 0x85, 0x1b, 0x5b, 0xb5, 0x7b, 0xe6, 0xa3,
	0xc0, 0x89, 0x77, 0x77, 0x42, 0xb0, 0xc3, 0x13, 0x93, 0xb8, 0xb6, 0x8f, 0x82, 0x8b, 0xb8, 0x3c,
	0xbf, 0x70, 0xbe, 0xfd, 0x68, 0x5b, 0x9d, 0xb7, 0xf8, 0xf2, 0xdf, 0xaf, 0x5c, 0x43, 0x13, 0xbd,
	0x5f, 0xac, 0x3d, 0xb4, 0x5c, 0x7b, 0xe8, 0xf7, 0xda, 0x43, 0xdf, 0x37, 0x9e, 0xb5, 0xdc, 0x78,
	0xd6, 0xaf, 0x8d, 0x67, 0x7d, 0x78, 0x99, 0x09, 0x33, 0x9c, 0xa5, 0x21, 0x53, 0x39, 0x65, 0x4a,
	0xe7, 0x4a, 0x53, 0x91, 0xb2, 0xab, 0x4c, 0xd1, 0xf9, 0x73, 0x9a, 0x2b, 0x3e, 0x1b, 0x83, 0xae,
	0x36, 0xf6, 0xc9, 0xb3, 0xab, 0xbd, 0xa5, 0x35, 0x5f, 0x27, 0xa0, 0xd3, 0x9b, 0xe5, 0xce, 0x3e,
	0xfd, 0x1b, 0x00, 0x00, 0xff, 0xff, 0x16, 0xa3, 0xcb, 0xa2, 0x26, 0x03, 0x00, 0x00,
}

func (m *GenesisState) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *GenesisState) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *GenesisState) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if len(m.SendSequences) > 0 {
		for iNdEx := len(m.SendSequences) - 1; iNdEx >= 0; iNdEx-- {
			{
				size, err := m.SendSequences[iNdEx].MarshalToSizedBuffer(dAtA[:i])
				if err != nil {
					return 0, err
				}
				i -= size
				i = encodeVarintGenesis(dAtA, i, uint64(size))
			}
			i--
			dAtA[i] = 0x2a
		}
	}
	if len(m.Receipts) > 0 {
		for iNdEx := len(m.Receipts) - 1; iNdEx >= 0; iNdEx-- {
			{
				size, err := m.Receipts[iNdEx].MarshalToSizedBuffer(dAtA[:i])
				if err != nil {
					return 0, err
				}
				i -= size
				i = encodeVarintGenesis(dAtA, i, uint64(size))
			}
			i--
			dAtA[i] = 0x22
		}
	}
	if len(m.Commitments) > 0 {
		for iNdEx := len(m.Commitments) - 1; iNdEx >= 0; iNdEx-- {
			{
				size, err := m.Commitments[iNdEx].MarshalToSizedBuffer(dAtA[:i])
				if err != nil {
					return 0, err
				}
				i -= size
				i = encodeVarintGenesis(dAtA, i, uint64(size))
			}
			i--
			dAtA[i] = 0x1a
		}
	}
	if len(m.Acknowledgements) > 0 {
		for iNdEx := len(m.Acknowledgements) - 1; iNdEx >= 0; iNdEx-- {
			{
				size, err := m.Acknowledgements[iNdEx].MarshalToSizedBuffer(dAtA[:i])
				if err != nil {
					return 0, err
				}
				i -= size
				i = encodeVarintGenesis(dAtA, i, uint64(size))
			}
			i--
			dAtA[i] = 0x12
		}
	}
	if len(m.Channels) > 0 {
		for iNdEx := len(m.Channels) - 1; iNdEx >= 0; iNdEx-- {
			{
				size, err := m.Channels[iNdEx].MarshalToSizedBuffer(dAtA[:i])
				if err != nil {
					return 0, err
				}
				i -= size
				i = encodeVarintGenesis(dAtA, i, uint64(size))
			}
			i--
			dAtA[i] = 0xa
		}
	}
	return len(dAtA) - i, nil
}

func (m *PacketState) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *PacketState) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *PacketState) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if len(m.Data) > 0 {
		i -= len(m.Data)
		copy(dAtA[i:], m.Data)
		i = encodeVarintGenesis(dAtA, i, uint64(len(m.Data)))
		i--
		dAtA[i] = 0x1a
	}
	if m.Sequence != 0 {
		i = encodeVarintGenesis(dAtA, i, uint64(m.Sequence))
		i--
		dAtA[i] = 0x10
	}
	if len(m.ChannelId) > 0 {
		i -= len(m.ChannelId)
		copy(dAtA[i:], m.ChannelId)
		i = encodeVarintGenesis(dAtA, i, uint64(len(m.ChannelId)))
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

func (m *PacketSequence) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *PacketSequence) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *PacketSequence) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if m.Sequence != 0 {
		i = encodeVarintGenesis(dAtA, i, uint64(m.Sequence))
		i--
		dAtA[i] = 0x10
	}
	if len(m.ChannelId) > 0 {
		i -= len(m.ChannelId)
		copy(dAtA[i:], m.ChannelId)
		i = encodeVarintGenesis(dAtA, i, uint64(len(m.ChannelId)))
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

func encodeVarintGenesis(dAtA []byte, offset int, v uint64) int {
	offset -= sovGenesis(v)
	base := offset
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return base
}
func (m *GenesisState) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	if len(m.Channels) > 0 {
		for _, e := range m.Channels {
			l = e.Size()
			n += 1 + l + sovGenesis(uint64(l))
		}
	}
	if len(m.Acknowledgements) > 0 {
		for _, e := range m.Acknowledgements {
			l = e.Size()
			n += 1 + l + sovGenesis(uint64(l))
		}
	}
	if len(m.Commitments) > 0 {
		for _, e := range m.Commitments {
			l = e.Size()
			n += 1 + l + sovGenesis(uint64(l))
		}
	}
	if len(m.Receipts) > 0 {
		for _, e := range m.Receipts {
			l = e.Size()
			n += 1 + l + sovGenesis(uint64(l))
		}
	}
	if len(m.SendSequences) > 0 {
		for _, e := range m.SendSequences {
			l = e.Size()
			n += 1 + l + sovGenesis(uint64(l))
		}
	}
	return n
}

func (m *PacketState) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.ChannelId)
	if l > 0 {
		n += 1 + l + sovGenesis(uint64(l))
	}
	if m.Sequence != 0 {
		n += 1 + sovGenesis(uint64(m.Sequence))
	}
	l = len(m.Data)
	if l > 0 {
		n += 1 + l + sovGenesis(uint64(l))
	}
	return n
}

func (m *PacketSequence) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.ChannelId)
	if l > 0 {
		n += 1 + l + sovGenesis(uint64(l))
	}
	if m.Sequence != 0 {
		n += 1 + sovGenesis(uint64(m.Sequence))
	}
	return n
}

func sovGenesis(x uint64) (n int) {
	return (math_bits.Len64(x|1) + 6) / 7
}
func sozGenesis(x uint64) (n int) {
	return sovGenesis(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (m *GenesisState) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowGenesis
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
			return fmt.Errorf("proto: GenesisState: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: GenesisState: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Channels", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowGenesis
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
				return ErrInvalidLengthGenesis
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthGenesis
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Channels = append(m.Channels, IdentifiedChannel{})
			if err := m.Channels[len(m.Channels)-1].Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Acknowledgements", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowGenesis
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
				return ErrInvalidLengthGenesis
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthGenesis
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Acknowledgements = append(m.Acknowledgements, PacketState{})
			if err := m.Acknowledgements[len(m.Acknowledgements)-1].Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 3:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Commitments", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowGenesis
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
				return ErrInvalidLengthGenesis
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthGenesis
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Commitments = append(m.Commitments, PacketState{})
			if err := m.Commitments[len(m.Commitments)-1].Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 4:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Receipts", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowGenesis
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
				return ErrInvalidLengthGenesis
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthGenesis
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Receipts = append(m.Receipts, PacketState{})
			if err := m.Receipts[len(m.Receipts)-1].Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 5:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field SendSequences", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowGenesis
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
				return ErrInvalidLengthGenesis
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthGenesis
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.SendSequences = append(m.SendSequences, PacketSequence{})
			if err := m.SendSequences[len(m.SendSequences)-1].Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipGenesis(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthGenesis
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
func (m *PacketState) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowGenesis
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
			return fmt.Errorf("proto: PacketState: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: PacketState: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field ChannelId", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowGenesis
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
				return ErrInvalidLengthGenesis
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthGenesis
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.ChannelId = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 2:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Sequence", wireType)
			}
			m.Sequence = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowGenesis
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
		case 3:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Data", wireType)
			}
			var byteLen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowGenesis
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
				return ErrInvalidLengthGenesis
			}
			postIndex := iNdEx + byteLen
			if postIndex < 0 {
				return ErrInvalidLengthGenesis
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Data = append(m.Data[:0], dAtA[iNdEx:postIndex]...)
			if m.Data == nil {
				m.Data = []byte{}
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipGenesis(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthGenesis
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
func (m *PacketSequence) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowGenesis
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
			return fmt.Errorf("proto: PacketSequence: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: PacketSequence: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field ChannelId", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowGenesis
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
				return ErrInvalidLengthGenesis
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthGenesis
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.ChannelId = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 2:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Sequence", wireType)
			}
			m.Sequence = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowGenesis
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
			skippy, err := skipGenesis(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthGenesis
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
func skipGenesis(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	depth := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowGenesis
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
					return 0, ErrIntOverflowGenesis
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
					return 0, ErrIntOverflowGenesis
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
				return 0, ErrInvalidLengthGenesis
			}
			iNdEx += length
		case 3:
			depth++
		case 4:
			if depth == 0 {
				return 0, ErrUnexpectedEndOfGroupGenesis
			}
			depth--
		case 5:
			iNdEx += 4
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
		if iNdEx < 0 {
			return 0, ErrInvalidLengthGenesis
		}
		if depth == 0 {
			return iNdEx, nil
		}
	}
	return 0, io.ErrUnexpectedEOF
}

var (
	ErrInvalidLengthGenesis        = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowGenesis          = fmt.Errorf("proto: integer overflow")
	ErrUnexpectedEndOfGroupGenesis = fmt.Errorf("proto: unexpected end of group")
)
