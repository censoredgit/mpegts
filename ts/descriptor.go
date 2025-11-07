package ts

import "encoding/binary"

const (
	DescriptorTagVideo             = 2
	DescriptorTagAudio             = 3
	DescriptorTagVideoWindow       = 8
	DescriptorTagMpeg4Video        = 27
	DescriptorTagMpeg4Audio        = 28
	DescriptorAvcVideo             = 40
	DescriptorAvcTimingAndHrdVideo = 42
	DescriptorTagIso639Language    = 10
	DescriptorTagSystemClock       = 11
	DescriptorTagMaximumBitrate    = 14
)

type Descriptor struct {
	DescriptorTag    uint8
	DescriptorLength uint8
	*AVCVideoDescriptor
	*AVCTimingAndHRDDescriptor
	Type uint8
}

type AVCVideoDescriptor struct {
	ProfileIdc                    uint8
	ConstraintSet0Flag            bool
	ConstraintSet1Flag            bool
	ConstraintSet2Flag            bool
	ConstraintSet3Flag            bool
	ConstraintSet4Flag            bool
	ConstraintSet5Flag            bool
	AVCCompatibleFlags            uint8
	LevelIdc                      uint8
	AVCStillPresent               bool
	AVC24HourPictureFlag          bool
	FramePackingSEINotPresentFLag bool
	Reserved                      uint8
}

type AVCTimingAndHRDDescriptor struct {
	HrdManagementValidFlag         bool
	Reserved                       uint8
	PictureAndTimingInfoPresent    bool
	Hz90rFlag                      bool
	Reserved2                      uint8
	N                              uint32
	K                              uint32
	NumUnitsInTick                 uint32
	FixedFrameRateFlag             bool
	TemporalPocFlag                bool
	PictureToDisplayConversionFlag bool
	Reserved3                      uint8
}

type VideoDescriptor struct {
	*Descriptor
	MultipleFrameRateFlag     bool
	FrameRateCode             uint8
	MPEG1OnlyFlag             bool
	ConstrainedParameterFlag  bool
	StillPictureFlag          bool
	ProfileAndLevelIndication uint8
	ChromaFormat              uint8
	FrameRateExtensionFlag    bool
	Reserved                  uint8
}

func (d *Descriptor) encode() []byte {
	buf := make([]byte, 2+d.DescriptorLength)
	buf[0] = d.DescriptorTag
	buf[1] = d.DescriptorLength

	switch d.DescriptorTag {
	case DescriptorAvcVideo:
		buf[2] = d.ProfileIdc
		if d.AVCVideoDescriptor.ConstraintSet0Flag {
			buf[3] |= 0x80
		}
		if d.AVCVideoDescriptor.ConstraintSet1Flag {
			buf[3] |= 0x40
		}
		if d.AVCVideoDescriptor.ConstraintSet2Flag {
			buf[3] |= 0x20
		}
		if d.AVCVideoDescriptor.ConstraintSet3Flag {
			buf[3] |= 0x10
		}
		if d.AVCVideoDescriptor.ConstraintSet4Flag {
			buf[3] |= 0x8
		}
		if d.AVCVideoDescriptor.ConstraintSet5Flag {
			buf[3] |= 0x4
		}
		buf[3] |= d.AVCVideoDescriptor.AVCCompatibleFlags & 0x03

		buf[4] = d.AVCVideoDescriptor.LevelIdc

		if d.AVCVideoDescriptor.AVCStillPresent {
			buf[5] |= 0x80
		}
		if d.AVCVideoDescriptor.AVC24HourPictureFlag {
			buf[5] |= 0x40
		}
		if d.AVCVideoDescriptor.FramePackingSEINotPresentFLag {
			buf[5] |= 0x20
		}
		buf[5] |= d.AVCVideoDescriptor.Reserved & 0x1f

	case DescriptorAvcTimingAndHrdVideo:
		counter := NewCounterOffset(2)
		if d.AVCTimingAndHRDDescriptor.HrdManagementValidFlag {
			buf[counter.Current()] |= 0x40
		}
		buf[counter.Current()] |= d.AVCTimingAndHRDDescriptor.Reserved
		buf[counter.Current()] <<= 1
		if d.AVCTimingAndHRDDescriptor.PictureAndTimingInfoPresent {
			buf[counter.Current()] |= 0x1
		}
		counter.Next()

		if d.AVCTimingAndHRDDescriptor.PictureAndTimingInfoPresent {
			if d.AVCTimingAndHRDDescriptor.Hz90rFlag {
				buf[counter.Current()] |= 0x80
			}
			buf[counter.Current()] |= d.AVCTimingAndHRDDescriptor.Reserved2
			counter.Next()
			if !d.AVCTimingAndHRDDescriptor.Hz90rFlag {
				binary.BigEndian.PutUint32(buf[counter.Current():], d.AVCTimingAndHRDDescriptor.N)
				counter.Seek(4)
				binary.BigEndian.PutUint32(buf[counter.Current():], d.AVCTimingAndHRDDescriptor.K)
				counter.Seek(4)
			}
			binary.BigEndian.PutUint32(buf[counter.Current():], d.AVCTimingAndHRDDescriptor.NumUnitsInTick)
			counter.Seek(4)
		}
		if d.AVCTimingAndHRDDescriptor.FixedFrameRateFlag {
			buf[counter.Current()] |= 0x80
		}
		if d.AVCTimingAndHRDDescriptor.TemporalPocFlag {
			buf[counter.Current()] |= 0x40
		}
		if d.AVCTimingAndHRDDescriptor.PictureToDisplayConversionFlag {
			buf[counter.Current()] |= 0x20
		}
		buf[counter.Current()] |= d.AVCTimingAndHRDDescriptor.Reserved3
	}

	return buf
}

func DecodeDescriptors(b []byte) []*Descriptor {
	descriptors := make([]*Descriptor, 0)

	counter := NewCounter[int]()
	for {
		if counter.Current() >= len(b)-1 {
			break
		}
		d := &Descriptor{}
		d.DescriptorTag = b[counter.Next()]
		d.DescriptorLength = b[counter.Next()]

		switch d.DescriptorTag {
		case DescriptorAvcVideo:
			d.Type = DescriptorAvcVideo
			d.AVCVideoDescriptor = &AVCVideoDescriptor{}
			d.AVCVideoDescriptor.ProfileIdc = b[counter.Next()]
			d.AVCVideoDescriptor.ConstraintSet0Flag = b[counter.Next()]&0x80 != 0
			d.AVCVideoDescriptor.ConstraintSet1Flag = b[counter.Next()]&0x40 != 0
			d.AVCVideoDescriptor.ConstraintSet2Flag = b[counter.Next()]&0x20 != 0
			d.AVCVideoDescriptor.ConstraintSet3Flag = b[counter.Next()]&0x10 != 0
			d.AVCVideoDescriptor.ConstraintSet4Flag = b[counter.Next()]&0x8 != 0
			d.AVCVideoDescriptor.ConstraintSet5Flag = b[counter.Next()]&0x4 != 0
			d.AVCVideoDescriptor.AVCCompatibleFlags = b[counter.Next()] & 0x3
			d.AVCVideoDescriptor.LevelIdc = b[counter.Next()]
			d.AVCVideoDescriptor.AVCStillPresent = b[counter.Next()]&0x80 != 0
			d.AVCVideoDescriptor.AVC24HourPictureFlag = b[counter.Next()]&0x40 != 0
			d.AVCVideoDescriptor.FramePackingSEINotPresentFLag = b[counter.Next()]&0x20 != 0
			d.AVCVideoDescriptor.Reserved = 0
		case DescriptorAvcTimingAndHrdVideo:
			d.Type = DescriptorAvcTimingAndHrdVideo
			d.AVCTimingAndHRDDescriptor = &AVCTimingAndHRDDescriptor{}
			d.AVCTimingAndHRDDescriptor.HrdManagementValidFlag = b[counter.Next()]&0x80 != 0
			d.AVCTimingAndHRDDescriptor.Reserved = (b[counter.Next()] >> 1) & 0x3f
			d.AVCTimingAndHRDDescriptor.PictureAndTimingInfoPresent = b[counter.Next()]&0x1 != 0
			if d.AVCTimingAndHRDDescriptor.PictureAndTimingInfoPresent {
				d.AVCTimingAndHRDDescriptor.Hz90rFlag = b[counter.Next()]&0x80 != 0
				d.AVCTimingAndHRDDescriptor.Reserved2 = b[counter.Next()] & 0x7f
				if d.AVCTimingAndHRDDescriptor.Hz90rFlag {
					d.AVCTimingAndHRDDescriptor.N = binary.BigEndian.Uint32(b[counter.Next() : counter.Current()+4])
					counter.Seek(4)
					d.AVCTimingAndHRDDescriptor.K = binary.BigEndian.Uint32(b[counter.Next() : counter.Current()+4])
					counter.Seek(3)
				}
				d.AVCTimingAndHRDDescriptor.FixedFrameRateFlag = b[counter.Next()]&0x80 != 0
				d.AVCTimingAndHRDDescriptor.TemporalPocFlag = b[counter.Next()]&0x40 != 0
				d.AVCTimingAndHRDDescriptor.PictureToDisplayConversionFlag = b[counter.Next()]&0x20 != 0
				d.AVCTimingAndHRDDescriptor.Reserved3 = b[counter.Next()] & 0x1f
			}
		default:
			counter.Seek(int(d.DescriptorLength) + 1)
		}

		descriptors = append(descriptors, d)
	}

	return descriptors
}
