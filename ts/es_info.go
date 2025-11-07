package ts

import "encoding/binary"

const StreamTypeVideoMpeg1 uint8 = 0x01
const StreamTypeVideoMpeg2 uint8 = 0x02
const StreamTypeAudioMpeg1 uint8 = 0x03
const StreamTypeAudioMpeg2 uint8 = 0x04
const StreamTypePrivateSection uint8 = 0x05
const StreamTypePrivateData uint8 = 0x06
const StreamTypeAudioAac uint8 = 0x0f
const StreamTypeAudioAacLatm uint8 = 0x11
const StreamTypeVideoMpeg4 uint8 = 0x10
const StreamTypeMetadata uint8 = 0x15
const StreamTypeVideoH264 uint8 = 0x1b
const StreamTypeVideoHevc uint8 = 0x24
const StreamTypeVideoCavs uint8 = 0x42
const StreamTypeVideoVc1 uint8 = 0xea
const StreamTypeVideoDirac uint8 = 0xd1
const StreamTypeAudioAc3 uint8 = 0x81
const StreamTypeAudioDts uint8 = 0x82
const StreamTypeAudioTrueHD uint8 = 0x83
const StreamTypeAudioEac3 uint8 = 0x87

type ESInfo struct {
	Streams []*Stream
}

type Stream struct {
	StreamType    uint8
	Reserved      uint8
	ElementaryPID uint16
	Reserved2     uint8
	EsInfoLength  uint16
	Descriptors   []*Descriptor
}

func (s *Stream) encode() []byte {
	descriptorSize := uint16(0)
	for _, d := range s.Descriptors {
		descriptorSize += uint16(len(d.encode()))
	}

	s.EsInfoLength = descriptorSize

	buf := make([]byte, 5+s.EsInfoLength)
	buf[0] = s.StreamType

	next16part := uint16(0)
	next16part |= uint16(s.Reserved) & 0x7
	next16part <<= 13
	next16part |= s.ElementaryPID
	binary.BigEndian.PutUint16(buf[1:], next16part)

	next16part = 0
	next16part |= uint16(s.Reserved2) & 0xf
	next16part <<= 12
	next16part |= s.EsInfoLength
	binary.BigEndian.PutUint16(buf[3:], next16part)

	index := 5
	for _, d := range s.Descriptors {
		tmp := d.encode()
		copy(buf[index:], tmp)
		index += len(tmp)
	}

	return buf
}

func (s *Stream) isVideo() bool {
	switch s.StreamType {
	case StreamTypeVideoCavs,
		StreamTypeVideoDirac,
		StreamTypeVideoH264,
		StreamTypeVideoHevc,
		StreamTypeVideoMpeg1,
		StreamTypeVideoMpeg2,
		StreamTypeVideoMpeg4,
		StreamTypeVideoVc1:
		return true
	}

	return false
}

func (s *Stream) isAudio() bool {
	switch s.StreamType {
	case StreamTypeAudioAac,
		StreamTypeAudioAacLatm,
		StreamTypeAudioAc3,
		StreamTypeAudioDts,
		StreamTypeAudioEac3,
		StreamTypeAudioMpeg1,
		StreamTypeAudioMpeg2,
		StreamTypeAudioTrueHD:
		return true
	}

	return false
}

func IsValidStreamTypeId(id uint8) bool {
	switch id {
	case StreamTypeVideoMpeg1,
		StreamTypeVideoMpeg2,
		StreamTypeAudioMpeg1,
		StreamTypeAudioMpeg2,
		StreamTypePrivateSection,
		StreamTypePrivateData,
		StreamTypeAudioAac,
		StreamTypeAudioAacLatm,
		StreamTypeVideoMpeg4,
		StreamTypeMetadata,
		StreamTypeVideoH264,
		StreamTypeVideoHevc,
		StreamTypeVideoCavs,
		StreamTypeVideoVc1,
		StreamTypeVideoDirac,
		StreamTypeAudioAc3,
		StreamTypeAudioDts,
		StreamTypeAudioTrueHD,
		StreamTypeAudioEac3:
		return true
	}

	return false
}

func DecodeESInfo(b []byte) *ESInfo {
	es := &ESInfo{}

	counter := NewCounter[int]()
	for counter.Current() < len(b) {
		s := &Stream{}
		s.StreamType = b[counter.Current()]
		counter.Next()
		next16part := binary.BigEndian.Uint16(b[counter.Current() : counter.Current()+2])
		counter.Seek(2)
		s.Reserved = uint8(next16part >> 13)
		s.ElementaryPID = next16part & 0x1fff
		next16part = binary.BigEndian.Uint16(b[counter.Current() : counter.Current()+2])
		counter.Seek(2)
		s.Reserved2 = uint8(next16part >> 12)
		s.EsInfoLength = next16part & 0x0fff

		s.Descriptors = DecodeDescriptors(b[counter.Current() : counter.Current()+int(s.EsInfoLength)])
		counter.Seek(int(s.EsInfoLength))
		es.Streams = append(es.Streams, s)
	}

	return es
}
