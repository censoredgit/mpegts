package ts

import "encoding/binary"

type PMT struct {
	TableId                uint8
	SectionSyntaxIndicator bool
	ZeroBit                bool
	Reserved               uint8
	SectionLength          uint16
	ProgramNumber          uint16
	Reserved2              uint8
	VersionNumber          uint8
	CurrentNextIndicator   bool
	SectionNumber          uint8
	LastSectionNumber      uint8
	Reserved3              uint8
	PCRPID                 uint16
	Reserved4              uint8
	ProgramInfoLength      uint16
	ProgramInfo            *ProgramInfo
	EsInfo                 *ESInfo
	Crc32                  uint32
}

func NewPMT() *PMT {
	return &PMT{
		TableId: 0x2,
	}
}

func (p *PMT) encode() []byte {
	streams := make([][]byte, len(p.EsInfo.Streams))
	fullLen := 3 + 13

	for i, stream := range p.EsInfo.Streams {
		streams[i] = stream.encode()
		fullLen += len(streams[i])
	}

	p.SectionLength = uint16(fullLen - 3)

	buf := make([]byte, fullLen)
	buf[0] = p.TableId

	next16part := uint16(0)
	if p.SectionSyntaxIndicator {
		next16part |= 0x1
		next16part <<= 1
	}
	next16part <<= 2
	next16part |= uint16(p.Reserved) & 0x3
	next16part <<= 12
	next16part |= p.SectionLength & 0x3ff

	binary.BigEndian.PutUint16(buf[1:], next16part)
	binary.BigEndian.PutUint16(buf[3:], p.ProgramNumber)

	next8part := uint8(0)
	next8part |= p.Reserved2
	next8part <<= 5
	next8part |= p.VersionNumber
	next8part <<= 1
	if p.CurrentNextIndicator {
		next8part |= 0x1
	}
	buf[5] = next8part
	buf[6] = p.SectionNumber
	buf[7] = p.LastSectionNumber

	next16part = 0
	next16part |= uint16(p.Reserved3) & 0x7
	next16part <<= 13
	next16part |= p.PCRPID
	binary.BigEndian.PutUint16(buf[8:], next16part)

	next16part = 0
	next16part |= uint16(p.Reserved4) & 0xf
	next16part <<= 12
	next16part |= p.ProgramInfoLength
	binary.BigEndian.PutUint16(buf[10:], next16part)

	//ES
	counter := NewCounterOffset[int](12)
	for _, stream := range streams {
		copy(buf[counter.Current():], stream)
		counter.Seek(len(stream))
	}

	binary.BigEndian.PutUint32(buf[fullLen-4:], computeCRC32(buf[:fullLen-4]))

	return buf
}

func DecodePMT(b []byte) *PMT {
	p := &PMT{}
	p.TableId = b[0]
	next16Part := binary.BigEndian.Uint16(b[1:3])
	p.SectionSyntaxIndicator = (next16Part>>15)&0x1 != 0
	p.ZeroBit = false
	p.Reserved = uint8((next16Part >> 12) & 0x3)
	p.SectionLength = next16Part & 0x0fff

	next16Part = binary.BigEndian.Uint16(b[3:5])
	p.ProgramNumber = next16Part
	next8part := b[5]
	p.Reserved2 = next8part >> 6
	p.VersionNumber = (next8part >> 1) & 0x1f
	p.CurrentNextIndicator = next8part&0x1 != 0
	p.SectionNumber = b[6]
	p.LastSectionNumber = b[7]

	next16Part = binary.BigEndian.Uint16(b[8:10])
	p.Reserved3 = uint8(next16Part >> 13)
	p.PCRPID = next16Part & 0x1fff

	next16Part = binary.BigEndian.Uint16(b[10:12])
	p.Reserved4 = uint8(next16Part >> 12)
	p.ProgramInfoLength = next16Part & 0x0fff

	p.ProgramInfo = DecodeProgramInfo(b[12 : 12+p.ProgramInfoLength])
	p.EsInfo = DecodeESInfo(b[12+p.ProgramInfoLength : 3+(p.SectionLength-4)])

	p.Crc32 = binary.BigEndian.Uint32(b[3+(p.SectionLength-4) : 3+(p.SectionLength)])

	return p
}
