package muxer

import (
	"context"
	"errors"
	"mpegts/ts"
	"os"
)

const JmDefaultVideoStreamId = 224
const JmDefaultAudioStreamId = 192

const JmDefaultPMTPid = 4096
const JmDefaultVideoPid = 256
const JmDefaultAudioPid = 250

const (
	JmStreamTypeVideoH264 = iota
	JmStreamTypeVideoHevc
	JmStreamTypeVideoMpeg1
	JmStreamTypeVideoMpeg2
	JmStreamTypeVideoMpeg4
	JmStreamTypeVideoCavs
	JmStreamTypeVideoDirac
	JmStreamTypeVideoVc1
	JmStreamTypeAudioAac
	JmStreamTypeAudioAacLatm
	JmStreamTypeAudioAc3
	JmStreamTypeAudioDts
	JmStreamTypeAudioMpeg1
	JmStreamTypeAudioMpeg2
	JmStreamTypeAudioTrueHD
	JmStreamTypeAudioEac3
)

type jmState uint8

const (
	jmReady jmState = iota
	jmOpened
	jmClosed
)

type JavaAdapter struct {
	destPath string
	pmtPid   int
	pcrPid   uint16
	state    jmState
	streams  []*StreamMeta
	ch       chan *StreamPacket
	closeCh  <-chan struct{}
}

func NewJavaAdapter(destPath string, pmtPid int) *JavaAdapter {
	return &JavaAdapter{
		destPath: destPath,
		streams:  make([]*StreamMeta, 0),
		ch:       make(chan *StreamPacket, 1024),
		pmtPid:   pmtPid,
		state:    jmReady,
	}
}

func (j *JavaAdapter) AddStream(pid int, streamId int, streamTypeId int) error {
	if j.state != jmReady {
		return errors.New("unavailable for current state")
	}

	if uint16(pid) == 0 {
		return errors.New("invalid pid")
	}

	if !ts.IsValidStreamId(uint8(streamId)) {
		return errors.New("invalid stream id")
	}

	_streamTypeId, err := j.toValidStreamType(streamTypeId)
	if err != nil {
		return err
	}

	j.streams = append(j.streams, &StreamMeta{
		Pid:          uint16(pid),
		StreamId:     uint8(streamId),
		StreamTypeId: _streamTypeId,
	})

	return nil
}

func (j *JavaAdapter) Open() error {
	if j.state != jmReady {
		return errors.New("unavailable for current state")
	}

	for _, stream := range j.streams {
		if ts.GetStreamIdType(stream.StreamId) == ts.VideoStream {
			j.pcrPid = stream.Pid
			break
		} else if ts.GetStreamIdType(stream.StreamId) == ts.AudioStream && j.pcrPid == 0 {
			j.pcrPid = stream.Pid
		}
	}

	if j.pcrPid == 0 {
		for _, stream := range j.streams {
			j.pcrPid = stream.Pid
			break
		}
	}

	f, err := os.OpenFile(j.destPath, os.O_CREATE|os.O_WRONLY, os.ModePerm)
	if err != nil {
		return err
	}

	j.closeCh, err = Run(context.Background(), f, uint16(j.pmtPid), j.pcrPid, j.streams, j.ch)
	if err != nil {
		return err
	}

	j.state = jmOpened

	return nil
}

func (j *JavaAdapter) Close() error {
	if j.state != jmOpened {
		return errors.New("unavailable for current state")
	}

	j.state = jmClosed
	close(j.ch)
	<-j.closeCh

	return nil
}

func (j *JavaAdapter) Write(pid int, b []byte, pts int64, isHead bool) error {
	if j.state != jmOpened {
		return errors.New("unavailable for current state")
	}

	nBuf := make([]byte, len(b))
	copy(nBuf, b)

	j.ch <- &StreamPacket{
		Data:   nBuf,
		Pid:    uint16(pid),
		Pts:    pts,
		IsHead: isHead,
	}

	return nil
}

func (j *JavaAdapter) toValidStreamType(streamType int) (uint8, error) {
	switch streamType {
	case JmStreamTypeVideoH264:
		return ts.StreamTypeVideoH264, nil
	case JmStreamTypeVideoHevc:
		return ts.StreamTypeVideoHevc, nil
	case JmStreamTypeVideoMpeg1:
		return ts.StreamTypeVideoMpeg1, nil
	case JmStreamTypeVideoMpeg2:
		return ts.StreamTypeVideoMpeg2, nil
	case JmStreamTypeVideoMpeg4:
		return ts.StreamTypeVideoMpeg4, nil
	case JmStreamTypeVideoCavs:
		return ts.StreamTypeVideoCavs, nil
	case JmStreamTypeVideoDirac:
		return ts.StreamTypeVideoDirac, nil
	case JmStreamTypeVideoVc1:
		return ts.StreamTypeVideoVc1, nil

	case JmStreamTypeAudioAac:
		return ts.StreamTypeAudioAac, nil
	case JmStreamTypeAudioAacLatm:
		return ts.StreamTypeAudioAacLatm, nil
	case JmStreamTypeAudioAc3:
		return ts.StreamTypeAudioAc3, nil
	case JmStreamTypeAudioDts:
		return ts.StreamTypeAudioDts, nil
	case JmStreamTypeAudioMpeg1:
		return ts.StreamTypeAudioMpeg1, nil
	case JmStreamTypeAudioMpeg2:
		return ts.StreamTypeAudioMpeg2, nil
	case JmStreamTypeAudioTrueHD:
		return ts.StreamTypeAudioTrueHD, nil
	case JmStreamTypeAudioEac3:
		return ts.StreamTypeAudioEac3, nil
	default:
		return 0, errors.New("invalid stream type")
	}
}
