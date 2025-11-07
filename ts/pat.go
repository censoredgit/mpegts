package ts

import (
	"encoding/binary"
)

type PAT struct {
	TableId                uint8
	SectionSyntaxIndicator bool
	ZeroBit                bool
	Reserved               uint8
	SectionLength          uint16
	TransportStreamId      uint16
	Reserved2              uint8
	VersionNumber          uint8
	CurrentNextIndicator   bool
	SectionNumber          uint8
	LastSectionNumber      uint8
	TableData              []*TableData
	Crc32                  uint32
}

type TableData struct {
	ProgramNumber uint16
	Reserved      uint8
	PID           uint16
	IsNetworkPID  bool
}

func NewPAT() *PAT {
	return &PAT{
		TableId: 0x0,
	}
}

func (p *PAT) encode() []byte {
	buf := make([]byte, 3+p.SectionLength)
	buf[0] = p.TableId

	next16part := uint16(0)
	if p.SectionSyntaxIndicator {
		next16part |= 0x1
		next16part <<= 3
	}
	next16part |= uint16(p.Reserved) & 0x3
	next16part <<= 12
	next16part |= p.SectionLength & 0x03ff

	binary.BigEndian.PutUint16(buf[1:], next16part)
	binary.BigEndian.PutUint16(buf[3:], p.TransportStreamId)

	next8part := uint8(0)
	if p.CurrentNextIndicator {
		next8part |= 0x1
	}
	next8part |= (p.VersionNumber << 1) & 0x3E
	next8part |= (p.Reserved2 << 6) & 0xC0
	buf[5] = next8part

	buf[6] = p.SectionNumber
	buf[7] = p.LastSectionNumber

	binary.BigEndian.PutUint16(buf[8:], p.TableData[0].ProgramNumber)
	binary.BigEndian.PutUint16(buf[10:], ((uint16(0)|(uint16(p.TableData[0].Reserved)&0x7))<<13)|p.TableData[0].PID)

	binary.BigEndian.PutUint32(buf[12:], computeCRC32(buf[:12]))

	return buf
}

func DecodePAT(b []byte) *PAT {
	p := &PAT{}
	p.TableId = b[0]

	next32part := binary.BigEndian.Uint32(b[1:5])
	p.SectionSyntaxIndicator = next32part>>31 == 0b1
	p.ZeroBit = false
	p.Reserved = uint8((next32part >> 28) & 0x3)
	p.SectionLength = uint16((next32part & 0xfff0000) >> 16)
	p.TransportStreamId = uint16(next32part & 0x0000ffff)

	next16part := binary.BigEndian.Uint16(b[5:7])
	p.Reserved2 = uint8(next16part >> 14)
	p.VersionNumber = uint8((next16part >> 9) & 0x001f)
	p.CurrentNextIndicator = (next16part>>8)&0x01 == 0x01
	p.SectionNumber = uint8(next16part & 0x00ff)
	p.LastSectionNumber = b[7]

	tableDataCount := (p.SectionLength*8 - 72) / 32
	p.TableData = make([]*TableData, tableDataCount)

	for i := uint16(0); i < tableDataCount; i++ {
		dat := binary.BigEndian.Uint32(b[8+i*4 : 12+i*4])
		tableData := &TableData{}
		tableData.ProgramNumber = uint16(dat >> 16)
		tableData.Reserved = uint8((dat >> 13) & 0x00000007)
		tableData.PID = uint16(dat & 0x00001fff)
		if dat>>16 == 0x0 {
			tableData.IsNetworkPID = true
		} else {
			tableData.IsNetworkPID = false
		}
		p.TableData[i] = tableData
	}
	p.Crc32 = binary.BigEndian.Uint32(b[12+(tableDataCount-1)*4 : (12+(tableDataCount-1)*4)+4])

	return p
}
