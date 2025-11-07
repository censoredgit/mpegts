package ts

import (
	"errors"
)

const PacketSize = 188

type Packet struct {
	Header     *Header
	Adaptation *AdaptationField
	Payload    *Payload
	container  *Container
}

func (p *Packet) GetPAT() (*PAT, error) {
	if p.Payload != nil && p.Payload.Type == PayloadPSI {
		if p.Payload.PSI.PAT != nil {
			return p.Payload.PSI.PAT, nil
		}
	}
	return nil, errors.New("pat not exists")
}

func (p *Packet) GetPMT() (*PMT, error) {
	if p.Payload != nil && p.Payload.Type == PayloadPSI {
		if p.Payload.PSI.PMT != nil {
			return p.Payload.PSI.PMT, nil
		}
	}
	return nil, errors.New("pmt not exists")
}

func (p *Packet) Encode() []byte {
	headerOffset := 4

	result := make([]byte, PacketSize)

	copy(result, p.Header.Encode())
	if p.Header.HasAdaptationField() {
		copy(result[headerOffset:], p.Adaptation.Encode())
		if p.Header.AdaptationFieldControl == 0x3 {
			copy(result[headerOffset+len(p.Adaptation.Encode()):], p.Payload.Encode())
		}
	} else {
		copy(result[headerOffset:], p.Payload.Encode())
	}

	return result
}
