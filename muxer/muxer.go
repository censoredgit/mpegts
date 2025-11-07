package muxer

import (
	"bytes"
	"context"
	"errors"
	"io"
	"mpegts/ts"
)

type Muxer struct {
	pmtPid      uint16
	pcrPid      uint16
	destination io.WriteCloser
	pidCounter  map[uint16]uint8
	streams     map[uint16]*StreamMeta
	closeCh     chan struct{}
}

type StreamPacket struct {
	Data   []byte
	Pid    uint16
	Pts    int64
	IsHead bool
}

type StreamMeta struct {
	Pid          uint16
	StreamId     uint8
	StreamTypeId uint8
}

func Run(
	ctx context.Context,
	destination io.WriteCloser,
	pmtPid uint16,
	pcrPid uint16,
	streams []*StreamMeta,
	inputStream <-chan *StreamPacket,
) (<-chan struct{}, error) {
	m := Muxer{}
	m.destination = destination
	m.pmtPid = pmtPid
	m.pcrPid = pcrPid
	m.streams = make(map[uint16]*StreamMeta)
	m.pidCounter = make(map[uint16]uint8)

	if m.pmtPid == 0 {
		return nil, errors.New("invalid pmt pid")
	}

	if len(streams) == 0 {
		return nil, errors.New("no streams")
	}

	isValidPcrPID := false
	for _, sm := range streams {
		if _, exists := m.streams[sm.Pid]; exists {
			return nil, errors.New("duplicate stream")
		}
		stream := *sm

		if !ts.IsValidStreamId(stream.StreamId) {
			return nil, errors.New("invalid stream id")
		}

		if !ts.IsValidStreamTypeId(stream.StreamTypeId) {
			return nil, errors.New("invalid stream type id")
		}

		if m.pcrPid == stream.Pid {
			isValidPcrPID = true
		}

		m.streams[sm.Pid] = &stream
	}

	if !isValidPcrPID {
		return nil, errors.New("invalid pcr pid")
	}

	patBuf := bytes.NewBuffer(m.createPAT())
	_, err := io.Copy(m.destination, patBuf)
	if err != nil {
		return nil, err
	}

	pmtBuf := bytes.NewBuffer(m.createPMT())
	_, err = io.Copy(m.destination, pmtBuf)
	if err != nil {
		return nil, err
	}

	m.closeCh = make(chan struct{})

	go m.process(ctx, inputStream)

	return m.closeCh, nil
}

func (m *Muxer) process(ctx context.Context, streamChannel <-chan *StreamPacket) {
	go func() {
		m.closeCh <- struct{}{}
		close(m.closeCh)
	}()

	var l int
	var err error
	for {
		select {
		case <-ctx.Done():
			return
		case sp, isActive := <-streamChannel:
			if !isActive {
				return
			}

			reader := bytes.NewBuffer(sp.Data)
			buffer := make([]byte, 184)

			if _, existsPid := m.pidCounter[sp.Pid]; !existsPid {
				m.pidCounter[sp.Pid] = 0
			}

			if sp.IsHead {
				adaptPack := m.createAdaptationPacket(sp.Pid, uint64(sp.Pts), sp.Pts != NoPts, sp.IsHead, reader.Len(), m.pidCounter[sp.Pid], m.streams[sp.Pid].StreamId)

				l, err = reader.Read(adaptPack.Payload.PES.Data)
				if l == 0 || err != nil {
					continue
				}

				_, err = m.destination.Write(adaptPack.Encode())
				if err != nil {
					continue
				}
			}

			m.pidCounter[sp.Pid]++
			if m.pidCounter[sp.Pid] == 16 {
				m.pidCounter[sp.Pid] = 0
			}

			for {
				l, err = reader.Read(buffer)
				if l > 0 {
					_, err = m.destination.Write(m.createDataPacket(sp.Pid, buffer, l, m.pidCounter[sp.Pid]))
				}

				if err != nil {
					break
				}

				m.pidCounter[sp.Pid]++
				if m.pidCounter[sp.Pid] == 16 {
					m.pidCounter[sp.Pid] = 0
				}
			}
		}
	}
}

func (m *Muxer) createPAT() []byte {
	packet := ts.Packet{}

	h := &ts.Header{}
	h.SyncByte = 0x47
	h.TransportErrorIndicator = false
	h.PayloadUntilStartIndicator = true
	h.TransportPriority = false
	h.PID = 0
	h.TransportScramblingControl = 0
	h.AdaptationFieldControl = 0x1
	h.ContinuityCounter = 0

	packet.Header = h

	payload := &ts.Payload{}
	payload.PSI = &ts.PSI{}
	payload.PSI.PointerField = 0
	payload.PSI.PointerFillerBytes = 0
	payload.Type = ts.PayloadPSI

	payload.PSI.PAT = ts.NewPAT()
	payload.PSI.PAT.SectionSyntaxIndicator = true
	payload.PSI.PAT.Reserved2 = 0x3
	payload.PSI.PAT.Reserved = 0x3

	payload.PSI.PAT.SectionLength = 13
	payload.PSI.PAT.TransportStreamId = 1
	payload.PSI.PAT.VersionNumber = 0
	payload.PSI.PAT.CurrentNextIndicator = true
	payload.PSI.PAT.SectionNumber = 0x0
	payload.PSI.PAT.LastSectionNumber = 0x0

	payload.PSI.PAT.TableData = make([]*ts.TableData, 1)
	payload.PSI.PAT.TableData[0] = &ts.TableData{
		ProgramNumber: 1,
		Reserved:      0x7,
		PID:           m.pmtPid,
	}

	packet.Payload = payload

	return packet.Encode()
}

func (m *Muxer) createPMT() []byte {
	packet := ts.Packet{}

	h := &ts.Header{}
	h.SyncByte = 0x47
	h.TransportErrorIndicator = false
	h.PayloadUntilStartIndicator = true
	h.TransportPriority = false
	h.PID = m.pmtPid
	h.TransportScramblingControl = 0
	h.AdaptationFieldControl = 0x1
	h.ContinuityCounter = 0

	packet.Header = h

	pmt := ts.NewPMT()
	pmt.SectionSyntaxIndicator = true
	pmt.ProgramNumber = 0x1
	pmt.VersionNumber = 0x0
	pmt.CurrentNextIndicator = true
	pmt.SectionNumber = 0
	pmt.LastSectionNumber = 0
	pmt.PCRPID = m.pcrPid
	pmt.ProgramInfoLength = 0
	pmt.Reserved = 3
	pmt.Reserved2 = 3
	pmt.Reserved3 = 7
	pmt.Reserved4 = 15

	esInfo := &ts.ESInfo{}
	esInfo.Streams = make([]*ts.Stream, 0)
	for _, v := range m.streams {
		esInfo.Streams = append(esInfo.Streams, &ts.Stream{
			StreamType:    v.StreamTypeId,
			Reserved:      7,
			ElementaryPID: v.Pid,
			Reserved2:     15,
		})
	}

	pmt.EsInfo = esInfo

	packet.Payload = &ts.Payload{}
	packet.Payload.Type = ts.PayloadPSI
	packet.Payload.PSI = &ts.PSI{}
	packet.Payload.PSI.PointerField = 0
	packet.Payload.PSI.PMT = pmt

	return packet.Encode()
}

func (m *Muxer) createDataPacket(pid uint16, data []byte, dataLen int, counter uint8) []byte {
	packet := ts.Packet{}

	h := ts.Header{}

	h.SyncByte = 0x47
	h.TransportErrorIndicator = false
	h.PayloadUntilStartIndicator = false
	h.TransportPriority = false
	h.PID = pid
	h.TransportScramblingControl = 0
	h.ContinuityCounter = counter

	packet.Header = &h
	packet.Payload = &ts.Payload{}
	packet.Payload.Type = ts.PayloadRawData

	if dataLen < 184 {
		stuffingBytesLen := 184 - dataLen
		h.AdaptationFieldControl = 0x3
		a := &ts.AdaptationField{}
		if stuffingBytesLen == 1 {
			a.AdaptationFieldLength = 0
		} else {
			a.AdaptationFieldLength = 1
			a.StuffingBytes = make([]byte, stuffingBytesLen-2) // exclude adaptation length and header
			for i := 0; i < len(a.StuffingBytes); i++ {
				a.StuffingBytes[i] = 255
			}
		}

		packet.Adaptation = a
		packet.Payload.RawData = ts.NewRawData(packet.Payload, data[:dataLen])
	} else {
		h.AdaptationFieldControl = 0x1
		packet.Payload.RawData = ts.NewRawData(packet.Payload, data)
	}
	return packet.Encode()
}

func (m *Muxer) createAdaptationPacket(pid uint16, pts uint64, hasPTS bool, needPCR bool, buffRemainLen int, counter uint8, streamId uint8) *ts.Packet {
	packet := ts.Packet{}

	h := ts.Header{}

	h.SyncByte = 0x47
	h.TransportErrorIndicator = false
	h.PayloadUntilStartIndicator = true
	h.TransportPriority = false
	h.PID = pid
	h.TransportScramblingControl = 0
	h.AdaptationFieldControl = 0x3
	h.ContinuityCounter = counter

	packet.Header = &h

	packet.Adaptation = &ts.AdaptationField{}
	packet.Adaptation.RndAccessIndicator = false
	packet.Adaptation.AdaptationFieldLength = 1

	packet.Payload = ts.NewPayload(&packet)
	packet.Payload.Type = ts.PayloadPES

	pes := ts.NewPES(packet.Payload)
	pes.StreamId = streamId
	pes.Header = &ts.PESHeader{}
	pes.Header.Marker = 2

	if !hasPTS {
		pes.Header.PTSDTSIndicator = 0x0
		pes.Header.PESHeaderDataLength = 0
		pes.Header.Data = &ts.PESHeaderData{}
	} else {
		pes.Header.PTSDTSIndicator = 0x2
		pes.Header.PESHeaderDataLength = 5
		pes.Header.Data = &ts.PESHeaderData{}
		pes.Header.Data.PTS = pts
	}

	if needPCR && pid == m.pcrPid {
		packet.Adaptation.PcrFlag = true

		pcr := uint64(0)
		if hasPTS && pts > 50 {
			pcr = (pts - 50) << 9
		}

		packet.Adaptation.PCR[0] = uint8((pcr >> 34) & 0xff)
		packet.Adaptation.PCR[1] = uint8((pcr >> 26) & 0xff)
		packet.Adaptation.PCR[2] = uint8((pcr >> 18) & 0xff)
		packet.Adaptation.PCR[3] = uint8((pcr >> 10) & 0xff)
		packet.Adaptation.PCR[4] = uint8(0x7e | ((pcr & (1 << 9)) >> 2) | ((pcr & (1 << 8)) >> 8))
		packet.Adaptation.PCR[5] = uint8(pcr & 0xff)
	}

	packet.Payload.PES = pes

	length := ts.PacketSize - (len(h.Encode()) + len(packet.Adaptation.Encode()) + len(packet.Payload.Encode()))
	if buffRemainLen < length {
		packet.Adaptation.StuffingBytes = make([]byte, length-buffRemainLen)
		for i := 0; i < len(packet.Adaptation.StuffingBytes); i++ {
			packet.Adaptation.StuffingBytes[i] = 255
		}
	}
	pes.Data = make([]byte, length)

	return &packet
}
