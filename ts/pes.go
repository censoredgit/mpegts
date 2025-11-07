package ts

import (
	"encoding/binary"
	"errors"
)

const (
	ProgramStreamMap = iota
	PrivateStream1
	PaddingStream
	PrivateStream2
	AudioStream
	VideoStream
	EcmStream
	EmmStream
	DsmccStream
	Stream13522
	TypeAStream
	TypeBStream
	TypeCStream
	TypeDStream
	TypeEStream
	AncillaryStream
	SlPackitizedStream
	FlexMuxStream
	ReservedDataStream
	ProgramStreamDirectory
)

var ErrInvalidPesStreamId = errors.New("invalid PES stream id")
var ErrInvalidPesHeaderMark = errors.New("invalid PES header mark")

type PES struct {
	StartCode    uint32
	StreamId     uint8
	PacketLength uint16
	Header       *PESHeader
	Data         []byte
	parent       *Payload
}

func NewPES(parent *Payload) *PES {
	return &PES{
		parent:    parent,
		Header:    &PESHeader{},
		StartCode: 1,
	}
}

func (p *PES) isVideo() bool {
	return p.getStreamIdType() == VideoStream
}

func (p *PES) isAudio() bool {
	return p.getStreamIdType() == AudioStream
}

func (p *PES) getStreamIdType() int {
	return GetStreamIdType(p.StreamId)
}

func (p *PES) hasHeader() bool {
	if p.getStreamIdType() != ProgramStreamMap &&
		p.getStreamIdType() != PaddingStream &&
		p.getStreamIdType() != PrivateStream2 &&
		p.getStreamIdType() != EcmStream &&
		p.getStreamIdType() != EmmStream &&
		p.getStreamIdType() != ProgramStreamDirectory &&
		p.getStreamIdType() != DsmccStream &&
		p.getStreamIdType() != TypeEStream {
		return true
	}
	return false
}

func (p *PES) encode() []byte {
	buf := make([]byte, PacketSize)
	next32part := uint32(0)

	next32part = p.StartCode
	next32part <<= 8
	next32part |= uint32(p.StreamId) & 0x000000ff
	binary.BigEndian.PutUint32(buf, next32part)

	binary.BigEndian.PutUint16(buf[4:], p.PacketLength)

	length := 7

	if p.hasHeader() {
		buf[6] = 0x80
		buf[6] |= p.Header.ScramblingControl << 4
		if p.Header.Priority {
			buf[6] |= 0x8
		}
		if p.Header.DataAlignmentIndicator {
			buf[6] |= 0x4
		}
		if p.Header.Copyright {
			buf[6] |= 0x2
		}
		if p.Header.OriginalOrCopy {
			buf[6] |= 0x1
		}

		buf[7] = p.Header.PTSDTSIndicator << 6
		if p.Header.ESCRFlag {
			buf[7] |= 0x20
		}
		if p.Header.ESRateFlag {
			buf[7] |= 0x10
		}
		if p.Header.DSMTrickModeFLag {
			buf[7] |= 0x8
		}
		if p.Header.AdditionalCopyInfoFlag {
			buf[7] |= 0x4
		}
		if p.Header.PESCRCFlag {
			buf[7] |= 0x2
		}
		if p.Header.PESExtensionFlag {
			buf[7] |= 0x1
		}

		buf[8] = p.Header.PESHeaderDataLength

		length = 9
		if buf[8] > 0 && p.Header.Data != nil {
			copy(buf[9:], p.Header.Data.encode(p.Header.PTSDTSIndicator == 0x2, p.Header.PTSDTSIndicator == 0x3))
			length += int(p.Header.PESHeaderDataLength)
		}

	}
	if len(p.Data) > 0 {
		copy(buf[length:], p.Data)
		length += len(p.Data)
	}

	return buf[:length]
}

func IsValidStreamId(t uint8) bool {
	switch t {
	case 0b1011_1100,
		0b1011_1101,
		0b1011_1110,
		0b1011_1111,
		0b1111_0000,
		0b1111_0001,
		0b1111_0010,
		0b1111_0011,
		0b1111_0100,
		0b1111_0101,
		0b1111_0110,
		0b1111_0111,
		0b1111_1000,
		0b1111_1001,
		0b1111_1010,
		0b1111_1011,
		0b1111_1111:
		return true
	}

	switch {
	case t >= 0b1100_0000 && t <= 0b1101_1111:
		return true
	case t >= 0b1110_0000 && t <= 0b1110_1111:
		return true
	case t >= 0b1111_1100 && t <= 0b1111_1110:
		return true
	}

	return false
}

func GetStreamIdType(streamId uint8) int {
	if streamId == 0b1011_1100 {
		return ProgramStreamMap
	}
	if streamId == 0b1011_1101 {
		return PrivateStream1
	}
	if streamId == 0b1011_1110 {
		return PaddingStream
	}
	if streamId == 0b1011_1111 {
		return PrivateStream2
	}
	if streamId >= 0b1100_0000 && streamId <= 0b1101_1111 {
		return AudioStream
	}
	if streamId >= 0b1110_0000 && streamId <= 0b1110_1111 {
		return VideoStream
	}
	if streamId == 0b1111_0000 {
		return EcmStream
	}
	if streamId == 0b1111_0001 {
		return EmmStream
	}
	if streamId == 0b1111_0010 {
		return DsmccStream
	}
	if streamId == 0b1111_0011 {
		return Stream13522
	}
	if streamId == 0b1111_0100 {
		return TypeAStream
	}
	if streamId == 0b1111_0101 {
		return TypeBStream
	}
	if streamId == 0b1111_0110 {
		return TypeCStream
	}
	if streamId == 0b1111_0111 {
		return TypeDStream
	}
	if streamId == 0b1111_1000 {
		return TypeEStream
	}
	if streamId == 0b1111_1001 {
		return AncillaryStream
	}
	if streamId == 0b1111_1010 {
		return SlPackitizedStream
	}
	if streamId == 0b1111_1011 {
		return FlexMuxStream
	}
	if streamId >= 0b1111_1100 && streamId <= 0b1111_1110 {
		return ReservedDataStream
	}

	return ProgramStreamDirectory
}

func DecodePES(parent *Payload, payload []byte) (*PES, error) {
	p := &PES{}
	p.parent = parent

	next32part := binary.BigEndian.Uint32(payload[0:4])
	p.StartCode = next32part >> 8
	streamId := uint8(next32part & 0x000000ff)
	if !IsValidStreamId(streamId) {
		return nil, ErrInvalidPesStreamId
	}
	p.StreamId = uint8(next32part & 0x000000ff)
	p.PacketLength = binary.BigEndian.Uint16(payload[4:6])

	if p.hasHeader() {
		p.Header = &PESHeader{}
		next16part := binary.BigEndian.Uint16(payload[6:8])
		p.Header.Marker = uint8(next16part >> 14)
		if p.Header.Marker != 0b10 {
			return nil, ErrInvalidPesHeaderMark
		}
		p.Header.ScramblingControl = uint8((next16part >> 12) & 0x3)
		p.Header.Priority = (next16part>>11)&0x1 != 0
		p.Header.DataAlignmentIndicator = (next16part>>10)&0x1 != 0
		p.Header.Copyright = (next16part>>9)&0x1 != 0
		p.Header.OriginalOrCopy = (next16part>>8)&0x1 != 0
		p.Header.PTSDTSIndicator = uint8((next16part >> 6) & 0x3)
		p.Header.ESCRFlag = (next16part>>5)&0x1 != 0
		p.Header.ESRateFlag = (next16part>>4)&0x1 != 0
		p.Header.DSMTrickModeFLag = (next16part>>3)&0x1 != 0
		p.Header.AdditionalCopyInfoFlag = (next16part>>2)&0x1 != 0
		p.Header.PESCRCFlag = (next16part>>1)&0x1 != 0
		p.Header.PESCRCFlag = next16part&0x1 != 0
		p.Header.PESHeaderDataLength = payload[8]
		dataOffset := uint8(9)
		if p.Header.PESHeaderDataLength > 0 {
			p.Header.Data = &PESHeaderData{}
			if p.Header.PTSDTSIndicator == 0x2 {
				ptsData := make([]byte, len(payload[dataOffset:dataOffset+p.Header.PESHeaderDataLength]))
				copy(ptsData, payload[dataOffset:dataOffset+p.Header.PESHeaderDataLength])
				p.Header.Data.PTS = ptsToUint(ptsData)
			} else if p.Header.PTSDTSIndicator == 0x3 {
				ptsDtsData := make([]byte, len(payload[dataOffset:dataOffset+p.Header.PESHeaderDataLength]))
				copy(ptsDtsData, payload[dataOffset:dataOffset+p.Header.PESHeaderDataLength])
				p.Header.Data.PTS = ptsToUint(ptsDtsData[:5])
				p.Header.Data.DTS = dtsToUint(ptsDtsData[5:])
			}

			dataOffset += p.Header.PESHeaderDataLength
		}
		p.Data = payload[dataOffset:]
	}

	return p, nil
}
