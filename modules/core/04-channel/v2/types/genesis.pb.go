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

// GenesisState defines the ibc channel/v2 submodule's genesis state.
type GenesisState struct {
	Acknowledgements []PacketState    `protobuf:"bytes,2,rep,name=acknowledgements,proto3" json:"acknowledgements"`
	Commitments      []PacketState    `protobuf:"bytes,3,rep,name=commitments,proto3" json:"commitments"`
	Receipts         []PacketState    `protobuf:"bytes,4,rep,name=receipts,proto3" json:"receipts"`
	AsyncPackets     []PacketState    `protobuf:"bytes,5,rep,name=async_packets,json=asyncPackets,proto3" json:"async_packets"`
	SendSequences    []PacketSequence `protobuf:"bytes,6,rep,name=send_sequences,json=sendSequences,proto3" json:"send_sequences"`
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

func (m *GenesisState) GetAsyncPackets() []PacketState {
	if m != nil {
		return m.AsyncPackets
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
	// client unique identifier.
	ClientId string `protobuf:"bytes,1,opt,name=client_id,json=clientId,proto3" json:"client_id,omitempty"`
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
	// client unique identifier.
	ClientId string `protobuf:"bytes,1,opt,name=client_id,json=clientId,proto3" json:"client_id,omitempty"`
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

func (m *PacketSequence) GetClientId() string {
	if m != nil {
		return m.ClientId
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
	// 394 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x9c, 0x92, 0x3f, 0x6f, 0xda, 0x40,
	0x18, 0xc6, 0x6d, 0xec, 0x22, 0x38, 0xfe, 0xa8, 0xba, 0x76, 0xb0, 0xa8, 0x64, 0x5c, 0xba, 0x78,
	0xc1, 0x47, 0x69, 0xa7, 0x4a, 0x5d, 0x58, 0x5a, 0xd4, 0x05, 0xb9, 0x43, 0xa5, 0x2e, 0xd4, 0x3e,
	0xbf, 0x32, 0x27, 0xec, 0x3b, 0x97, 0x3b, 0x88, 0xf8, 0x06, 0xd9, 0x92, 0x8f, 0x90, 0x8f, 0xc3,
	0xc8, 0x98, 0x29, 0x8a, 0xe0, 0x8b, 0x44, 0xd8, 0x80, 0x88, 0x12, 0x45, 0x22, 0xdb, 0xeb, 0xd7,
	0xcf, 0xef, 0xf7, 0xde, 0xf0, 0xa0, 0x8f, 0x2c, 0xa4, 0x84, 0x8a, 0x19, 0x10, 0x3a, 0x09, 0x38,
	0x87, 0x84, 0x2c, 0xfa, 0x24, 0x06, 0x0e, 0x92, 0x49, 0x2f, 0x9b, 0x09, 0x25, 0xf0, 0x3b, 0x16,
	0x52, 0x6f, 0x17, 0xf1, 0xf6, 0x11, 0x6f, 0xd1, 0x6f, 0xbd, 0x8f, 0x45, 0x2c, 0xf2, 0xff, 0x64,
	0x37, 0x15, 0xd1, 0xce, 0x95, 0x81, 0xea, 0x3f, 0x0a, 0xf8, 0xb7, 0x0a, 0x14, 0x60, 0x1f, 0xbd,
	0x0d, 0xe8, 0x94, 0x8b, 0x8b, 0x04, 0xa2, 0x18, 0x52, 0xe0, 0x4a, 0x5a, 0x25, 0xc7, 0x70, 0x6b,
	0x7d, 0xc7, 0x7b, 0x46, 0xeb, 0x8d, 0x02, 0x3a, 0x05, 0x95, 0xb3, 0x03, 0x73, 0x75, 0xd7, 0xd6,
	0xfc, 0x27, 0x3c, 0xfe, 0x89, 0x6a, 0x54, 0xa4, 0x29, 0x53, 0x85, 0xce, 0x38, 0x4b, 0x77, 0x8a,
	0xe2, 0x01, 0xaa, 0xcc, 0x80, 0x02, 0xcb, 0x94, 0xb4, 0xcc, 0xb3, 0x34, 0x47, 0x0e, 0xff, 0x42,
	0x8d, 0x40, 0x2e, 0x39, 0x1d, 0x67, 0x79, 0x48, 0x5a, 0x6f, 0xce, 0x12, 0xd5, 0x73, 0xb8, 0xd8,
	0x4b, 0x3c, 0x42, 0x4d, 0x09, 0x3c, 0x1a, 0x4b, 0xf8, 0x3f, 0x07, 0x4e, 0x41, 0x5a, 0xe5, 0xdc,
	0xf6, 0xe9, 0x25, 0xdb, 0x3e, 0xbb, 0x17, 0x36, 0x76, 0x82, 0xc3, 0x4e, 0x76, 0xfe, 0xa1, 0xda,
	0xc9, 0x51, 0xfc, 0x01, 0x55, 0x69, 0xc2, 0x80, 0xab, 0x31, 0x8b, 0x2c, 0xdd, 0xd1, 0xdd, 0xaa,
	0x5f, 0x29, 0x16, 0xc3, 0x08, 0xb7, 0x50, 0xe5, 0x70, 0xd8, 0x2a, 0x39, 0xba, 0x6b, 0xfa, 0xc7,
	0x6f, 0x8c, 0x91, 0x19, 0x05, 0x2a, 0xb0, 0x0c, 0x47, 0x77, 0xeb, 0x7e, 0x3e, 0x7f, 0x33, 0x2f,
	0x6f, 0xda, 0x5a, 0x67, 0x88, 0x9a, 0x8f, 0x1f, 0xf2, 0xea, 0x23, 0x83, 0x3f, 0xab, 0x8d, 0xad,
	0xaf, 0x37, 0xb6, 0x7e, 0xbf, 0xb1, 0xf5, 0xeb, 0xad, 0xad, 0xad, 0xb7, 0xb6, 0x76, 0xbb, 0xb5,
	0xb5, 0xbf, 0xdf, 0x63, 0xa6, 0x26, 0xf3, 0xd0, 0xa3, 0x22, 0x25, 0x54, 0xc8, 0x54, 0x48, 0xc2,
	0x42, 0xda, 0x8d, 0x05, 0x59, 0x7c, 0xee, 0x91, 0x54, 0x44, 0xf3, 0x04, 0x64, 0xd1, 0xe3, 0xde,
	0xd7, 0xee, 0x49, 0x95, 0xd5, 0x32, 0x03, 0x19, 0x96, 0xf3, 0x7a, 0x7e, 0x79, 0x08, 0x00, 0x00,
	0xff, 0xff, 0x9c, 0x0e, 0x36, 0xab, 0xee, 0x02, 0x00, 0x00,
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
			dAtA[i] = 0x32
		}
	}
	if len(m.AsyncPackets) > 0 {
		for iNdEx := len(m.AsyncPackets) - 1; iNdEx >= 0; iNdEx-- {
			{
				size, err := m.AsyncPackets[iNdEx].MarshalToSizedBuffer(dAtA[:i])
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
	if len(m.ClientId) > 0 {
		i -= len(m.ClientId)
		copy(dAtA[i:], m.ClientId)
		i = encodeVarintGenesis(dAtA, i, uint64(len(m.ClientId)))
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
	if len(m.ClientId) > 0 {
		i -= len(m.ClientId)
		copy(dAtA[i:], m.ClientId)
		i = encodeVarintGenesis(dAtA, i, uint64(len(m.ClientId)))
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
	if len(m.AsyncPackets) > 0 {
		for _, e := range m.AsyncPackets {
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
	l = len(m.ClientId)
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
	l = len(m.ClientId)
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
				return fmt.Errorf("proto: wrong wireType = %d for field AsyncPackets", wireType)
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
			m.AsyncPackets = append(m.AsyncPackets, PacketState{})
			if err := m.AsyncPackets[len(m.AsyncPackets)-1].Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 6:
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
				return fmt.Errorf("proto: wrong wireType = %d for field ClientId", wireType)
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
			m.ClientId = string(dAtA[iNdEx:postIndex])
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
				return fmt.Errorf("proto: wrong wireType = %d for field ClientId", wireType)
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
			m.ClientId = string(dAtA[iNdEx:postIndex])
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
