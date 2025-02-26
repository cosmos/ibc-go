// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: ibc/applications/transfer/v1/genesis.proto

package types

import (
	fmt "fmt"
	github_com_cosmos_cosmos_sdk_types "github.com/cosmos/cosmos-sdk/types"
	types "github.com/cosmos/cosmos-sdk/types"
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

// GenesisState defines the ibc-transfer genesis state
type GenesisState struct {
	PortId string `protobuf:"bytes,1,opt,name=port_id,json=portId,proto3" json:"port_id,omitempty"`
	Denoms Denoms `protobuf:"bytes,2,rep,name=denoms,proto3,castrepeated=Denoms" json:"denoms"`
	Params Params `protobuf:"bytes,3,opt,name=params,proto3" json:"params"`
	// total_escrowed contains the total amount of tokens escrowed
	// by the transfer module
	TotalEscrowed github_com_cosmos_cosmos_sdk_types.Coins `protobuf:"bytes,4,rep,name=total_escrowed,json=totalEscrowed,proto3,castrepeated=github.com/cosmos/cosmos-sdk/types.Coins" json:"total_escrowed"`
}

func (m *GenesisState) Reset()         { *m = GenesisState{} }
func (m *GenesisState) String() string { return proto.CompactTextString(m) }
func (*GenesisState) ProtoMessage()    {}
func (*GenesisState) Descriptor() ([]byte, []int) {
	return fileDescriptor_a4f788affd5bea89, []int{0}
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

func (m *GenesisState) GetPortId() string {
	if m != nil {
		return m.PortId
	}
	return ""
}

func (m *GenesisState) GetDenoms() Denoms {
	if m != nil {
		return m.Denoms
	}
	return nil
}

func (m *GenesisState) GetParams() Params {
	if m != nil {
		return m.Params
	}
	return Params{}
}

func (m *GenesisState) GetTotalEscrowed() github_com_cosmos_cosmos_sdk_types.Coins {
	if m != nil {
		return m.TotalEscrowed
	}
	return nil
}

func init() {
	proto.RegisterType((*GenesisState)(nil), "ibc.applications.transfer.v1.GenesisState")
}

func init() {
	proto.RegisterFile("ibc/applications/transfer/v1/genesis.proto", fileDescriptor_a4f788affd5bea89)
}

var fileDescriptor_a4f788affd5bea89 = []byte{
	// 374 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x7c, 0x91, 0xc1, 0x4e, 0xe3, 0x30,
	0x10, 0x86, 0x93, 0xb6, 0xca, 0x6a, 0xd3, 0xdd, 0x1e, 0xa2, 0x95, 0x36, 0x5b, 0xad, 0xd2, 0x0a,
	0x38, 0x44, 0xa0, 0xda, 0x4d, 0xb9, 0x70, 0x0e, 0x20, 0x84, 0xb8, 0xa0, 0x70, 0xe3, 0x52, 0x39,
	0x8e, 0x09, 0x56, 0x9b, 0x4c, 0x14, 0xbb, 0x41, 0xbc, 0x05, 0xcf, 0x81, 0x78, 0x90, 0x1e, 0x7b,
	0xe4, 0x04, 0xa8, 0x7d, 0x11, 0x14, 0x27, 0x45, 0x95, 0x90, 0x72, 0xf2, 0x8c, 0xfd, 0xcd, 0x3f,
	0xe3, 0x7f, 0xcc, 0x43, 0x1e, 0x52, 0x4c, 0xb2, 0x6c, 0xce, 0x29, 0x91, 0x1c, 0x52, 0x81, 0x65,
	0x4e, 0x52, 0x71, 0xc7, 0x72, 0x5c, 0x78, 0x38, 0x66, 0x29, 0x13, 0x5c, 0xa0, 0x2c, 0x07, 0x09,
	0xd6, 0x7f, 0x1e, 0x52, 0xb4, 0xcb, 0xa2, 0x2d, 0x8b, 0x0a, 0xaf, 0x7f, 0xd4, 0xa8, 0xf4, 0x45,
	0x2a, 0xa9, 0xbe, 0xdb, 0x0c, 0xc3, 0x8c, 0xa5, 0x35, 0xe9, 0x50, 0x10, 0x09, 0x08, 0x1c, 0x12,
	0xc1, 0x70, 0xe1, 0x85, 0x4c, 0x12, 0x0f, 0x53, 0xe0, 0xdb, 0xf7, 0x3f, 0x31, 0xc4, 0xa0, 0x42,
	0x5c, 0x46, 0xd5, 0xed, 0xde, 0x4b, 0xcb, 0xfc, 0x75, 0x51, 0x0d, 0x7f, 0x23, 0x89, 0x64, 0xd6,
	0x5f, 0xf3, 0x47, 0x06, 0xb9, 0x9c, 0xf2, 0xc8, 0xd6, 0x87, 0xba, 0xfb, 0x33, 0x30, 0xca, 0xf4,
	0x32, 0xb2, 0xae, 0x4c, 0x23, 0x62, 0x29, 0x24, 0xc2, 0x6e, 0x0d, 0xdb, 0x6e, 0x77, 0xb2, 0x8f,
	0x9a, 0x7e, 0x89, 0xce, 0x4a, 0xd6, 0xef, 0x2d, 0xdf, 0x06, 0xda, 0xf3, 0xfb, 0xc0, 0x50, 0xa9,
	0x08, 0x6a, 0x09, 0xcb, 0x37, 0x8d, 0x8c, 0xe4, 0x24, 0x11, 0x76, 0x7b, 0xa8, 0xbb, 0xdd, 0xc9,
	0x41, 0xb3, 0xd8, 0xb5, 0x62, 0xfd, 0x4e, 0xa9, 0x16, 0xd4, 0x95, 0x56, 0x6e, 0xf6, 0x24, 0x48,
	0x32, 0x9f, 0x32, 0x41, 0x73, 0x78, 0x60, 0x91, 0xdd, 0x51, 0x83, 0xfd, 0x43, 0x95, 0x13, 0xa8,
	0x74, 0x02, 0xd5, 0x4e, 0xa0, 0x53, 0xe0, 0xa9, 0x3f, 0xae, 0xc7, 0x71, 0x63, 0x2e, 0xef, 0x17,
	0x21, 0xa2, 0x90, 0xe0, 0xda, 0xb6, 0xea, 0x18, 0x89, 0x68, 0x86, 0xe5, 0x63, 0xc6, 0x84, 0x2a,
	0x10, 0xc1, 0x6f, 0xd5, 0xe2, 0xbc, 0xee, 0xe0, 0x07, 0xcb, 0xb5, 0xa3, 0xaf, 0xd6, 0x8e, 0xfe,
	0xb1, 0x76, 0xf4, 0xa7, 0x8d, 0xa3, 0xad, 0x36, 0x8e, 0xf6, 0xba, 0x71, 0xb4, 0xdb, 0x93, 0xef,
	0x92, 0x3c, 0xa4, 0xa3, 0x18, 0x70, 0xe1, 0x8d, 0x71, 0x02, 0xd1, 0x62, 0xce, 0x44, 0xb9, 0xc9,
	0x9d, 0x0d, 0xaa, 0x46, 0xa1, 0xa1, 0x36, 0x71, 0xfc, 0x19, 0x00, 0x00, 0xff, 0xff, 0x83, 0x55,
	0x07, 0xb2, 0x62, 0x02, 0x00, 0x00,
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
	if len(m.TotalEscrowed) > 0 {
		for iNdEx := len(m.TotalEscrowed) - 1; iNdEx >= 0; iNdEx-- {
			{
				size, err := m.TotalEscrowed[iNdEx].MarshalToSizedBuffer(dAtA[:i])
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
	{
		size, err := m.Params.MarshalToSizedBuffer(dAtA[:i])
		if err != nil {
			return 0, err
		}
		i -= size
		i = encodeVarintGenesis(dAtA, i, uint64(size))
	}
	i--
	dAtA[i] = 0x1a
	if len(m.Denoms) > 0 {
		for iNdEx := len(m.Denoms) - 1; iNdEx >= 0; iNdEx-- {
			{
				size, err := m.Denoms[iNdEx].MarshalToSizedBuffer(dAtA[:i])
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
	if len(m.PortId) > 0 {
		i -= len(m.PortId)
		copy(dAtA[i:], m.PortId)
		i = encodeVarintGenesis(dAtA, i, uint64(len(m.PortId)))
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
	l = len(m.PortId)
	if l > 0 {
		n += 1 + l + sovGenesis(uint64(l))
	}
	if len(m.Denoms) > 0 {
		for _, e := range m.Denoms {
			l = e.Size()
			n += 1 + l + sovGenesis(uint64(l))
		}
	}
	l = m.Params.Size()
	n += 1 + l + sovGenesis(uint64(l))
	if len(m.TotalEscrowed) > 0 {
		for _, e := range m.TotalEscrowed {
			l = e.Size()
			n += 1 + l + sovGenesis(uint64(l))
		}
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
				return fmt.Errorf("proto: wrong wireType = %d for field PortId", wireType)
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
			m.PortId = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Denoms", wireType)
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
			m.Denoms = append(m.Denoms, Denom{})
			if err := m.Denoms[len(m.Denoms)-1].Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 3:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Params", wireType)
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
			if err := m.Params.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 4:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field TotalEscrowed", wireType)
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
			m.TotalEscrowed = append(m.TotalEscrowed, types.Coin{})
			if err := m.TotalEscrowed[len(m.TotalEscrowed)-1].Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
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
