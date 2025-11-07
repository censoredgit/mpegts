package ts

import (
	"slices"
)

type Container struct {
	packets         []*Packet
	pmtPIDs         []uint16
	audioStreamPIDs []uint16
	videoStreamPIDs []uint16
}

func NewContainer() *Container {
	return &Container{}
}

func (c *Container) DecodePacket(b []byte) (*Packet, error) {
	ts := &Packet{}
	ts.container = c

	var err error

	ts.Header, err = DecodeHeader(ts, b)
	if err != nil {
		return nil, err
	}

	offset := uint8(0)

	if ts.Header.HasAdaptationField() {
		ts.Adaptation = DecodeAdaptationField(ts.Header.AdaptationFieldControl, b[4+offset:])
	}

	if ts.Header.HasPayload() {
		if ts.Header.HasAdaptationField() {
			ts.Payload, err = DecodePayload(ts, ts.Header.PID, b[5+offset+ts.Adaptation.AdaptationFieldLength:], false)
		} else {
			ts.Payload, err = DecodePayload(ts, ts.Header.PID, b[4+offset:], false)
		}
	} else if ts.Header.IsRawStreamData() {
		if ts.Header.HasAdaptationField() {
			ts.Payload, err = DecodePayload(ts, ts.Header.PID, b[5+offset+ts.Adaptation.AdaptationFieldLength:], true)
		} else {
			ts.Payload, err = DecodePayload(ts, ts.Header.PID, b[4+offset:], true)
		}
	}

	if err != nil {
		return nil, err
	}

	var pat *PAT
	var pmt *PMT

	if pat, err = ts.GetPAT(); err == nil && len(pat.TableData) > 0 {
		for i := 0; i < len(pat.TableData); i++ {
			if !slices.Contains(c.pmtPIDs, pat.TableData[i].PID) {
				c.pmtPIDs = append(c.pmtPIDs, pat.TableData[i].PID)
			}
		}
	}

	if pmt, err = ts.GetPMT(); err == nil {
		for _, stream := range pmt.EsInfo.Streams {
			if stream.isVideo() {
				if !slices.Contains(c.videoStreamPIDs, stream.ElementaryPID) {
					c.videoStreamPIDs = append(c.videoStreamPIDs, stream.ElementaryPID)
				}
			} else if stream.isAudio() {
				if !slices.Contains(c.audioStreamPIDs, stream.ElementaryPID) {
					c.audioStreamPIDs = append(c.audioStreamPIDs, stream.ElementaryPID)
				}
			}
		}
	}

	return ts, nil
}

func (c *Container) addAudioStreamPID(pid uint16) {
	c.audioStreamPIDs = append(c.audioStreamPIDs, pid)
}

func (c *Container) addVideoStreamPID(pid uint16) {
	c.videoStreamPIDs = append(c.videoStreamPIDs, pid)
}
