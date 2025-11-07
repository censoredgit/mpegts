package ts

import (
	"encoding/binary"
	"errors"
	"slices"
)

var ErrInvalidHeaderSyncByte = errors.New("invalid header sync byte")
var ErrInvalidInputData = errors.New("invalid input data")

type Header struct {
	SyncByte                   uint8
	TransportErrorIndicator    bool
	PayloadUntilStartIndicator bool
	TransportPriority          bool
	PID                        uint16
	TransportScramblingControl uint8
	AdaptationFieldControl     uint8
	ContinuityCounter          uint8
	parent                     *Packet
}

func (h *Header) HasPayload() bool {
	return h.PayloadUntilStartIndicator && (h.AdaptationFieldControl == 0x1 || h.AdaptationFieldControl == 0x3)
}

func (h *Header) IsRawStreamData() bool {
	return slices.Contains(h.parent.container.videoStreamPIDs, h.PID) || slices.Contains(h.parent.container.audioStreamPIDs, h.PID)
}

func (h *Header) IsPAT() bool {
	return h.parent.Payload != nil && h.parent.Payload.Type == PayloadPSI &&
		h.parent.Payload.PSI.PAT != nil
}

func (h *Header) IsPMT() bool {
	return h.parent.Payload != nil && h.parent.Payload.Type == PayloadPSI &&
		h.parent.Payload.PSI.PMT != nil
}

func (h *Header) HasAdaptationField() bool {
	return h.AdaptationFieldControl == 0x2 || h.AdaptationFieldControl == 0x3
}

func (h *Header) Encode() []byte {
	header := make([]byte, 4)
	counter := NewCounter[uint8]()

	header[counter.Next()] = h.SyncByte

	var indAndPID uint16 = 0
	indAndPID |= h.PID
	if h.TransportErrorIndicator {
		indAndPID |= 0b1000000000000000
	}
	if h.PayloadUntilStartIndicator {
		indAndPID |= 0b100000000000000
	}
	if h.TransportPriority {
		indAndPID |= 0b10000000000000
	}

	var next8part uint8 = 0
	next8part |= h.TransportScramblingControl
	next8part <<= 2
	next8part |= h.AdaptationFieldControl
	next8part <<= 4
	next8part |= h.ContinuityCounter

	binary.BigEndian.PutUint16(header[counter.Current():], indAndPID)
	counter.Seek(2)
	header[counter.Current()] = next8part

	return header
}

func DecodeHeader(parent *Packet, pes []byte) (*Header, error) {
	h := &Header{}
	h.parent = parent

	header32Part := uint32(0)
	if pes[0] == 0x47 {
		header32Part = binary.BigEndian.Uint32(pes[0:4])
		h.SyncByte = uint8(header32Part >> 24)
		if h.SyncByte != 0x47 {
			return nil, ErrInvalidHeaderSyncByte
		}
	} else {
		return nil, ErrInvalidInputData
	}

	h.TransportErrorIndicator = ((header32Part >> 23) & 0x1) == 0x1
	h.PayloadUntilStartIndicator = ((header32Part >> 22) & 0x1) == 0x1
	h.TransportPriority = ((header32Part >> 21) & 0x1) == 0x1
	h.PID = uint16((header32Part >> 8) & 0x001fff)
	h.TransportScramblingControl = uint8((header32Part & 0x000000ff) >> 6)
	h.AdaptationFieldControl = uint8((header32Part & 0x0000003f) >> 4)
	h.ContinuityCounter = uint8(header32Part & 0x0000000f)

	return h, nil
}
