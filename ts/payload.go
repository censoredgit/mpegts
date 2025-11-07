package ts

import "encoding/binary"

const (
	PayloadPSI     = 1
	PayloadPES     = 2
	PayloadRawData = 3
)

type Payload struct {
	PES     *PES
	PSI     *PSI
	RawData *RawData
	Type    int8
	parent  *Packet
}

func NewPayload(parent *Packet) *Payload {
	return &Payload{
		parent: parent,
	}
}

func (p *Payload) Encode() []byte {
	if p.Type == PayloadPSI {
		return p.PSI.encode()
	} else if p.Type == PayloadPES {
		return p.PES.encode()
	} else {
		return p.RawData.encode()
	}
}

func IsPES(p []byte) bool {
	if len(p) > 4 {
		return (binary.BigEndian.Uint32(p[0:4])>>8)&0x000001 != 0
	}
	return false
}

func DecodePayload(parent *Packet, pid uint16, b []byte, onlyData bool) (*Payload, error) {
	p := &Payload{}
	p.Type = 0
	p.parent = parent

	var err error

	if onlyData {
		p.RawData = NewRawData(p, b)
		p.Type = PayloadRawData
	} else if IsPES(b) {
		p.PES, err = DecodePES(p, b)
		p.Type = PayloadPES
	} else if !p.parent.Header.HasAdaptationField() {
		p.PSI, err = DecodePSI(p, b, pid)
		p.Type = PayloadPSI
	}
	return p, err
}
