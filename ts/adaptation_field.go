package ts

type AdaptationField struct {
	AdaptationFieldLength        uint8
	DiscontinuityIndicator       bool
	RndAccessIndicator           bool
	EsPriorityIndicator          bool
	PcrFlag                      bool
	OpcrFlag                     bool
	SplicingPointFlag            bool
	TransportPrivateDataFlag     bool
	AdaptationFieldExtensionFlag bool
	Type                         uint8
	PCR                          [6]byte
	OPCR                         [6]byte
	SpliceCountdown              uint8
	TransportPrivateDataLen      uint8
	TransportPrivateData         []byte
	AdaptationExtension          *AdaptationFieldExtension
	StuffingBytes                []byte
}

func (a *AdaptationField) Encode() []byte {
	buf := make([]byte, 2)

	if a.AdaptationFieldLength > 0 {
		a.AdaptationFieldLength = 1
	}

	if a.DiscontinuityIndicator {
		buf[1] |= 0x80
	}
	if a.RndAccessIndicator {
		buf[1] |= 0x40
	}
	if a.EsPriorityIndicator {
		buf[1] |= 0x20
	}
	if a.PcrFlag {
		buf[1] |= 0x10
	}
	if a.OpcrFlag {
		buf[1] |= 0x8
	}
	if a.SplicingPointFlag {
		buf[1] |= 0x4
	}
	if a.TransportPrivateDataFlag {
		buf[1] |= 0x2
	}
	if a.AdaptationFieldExtensionFlag {
		buf[1] |= 0x1
	}

	if a.PcrFlag {
		buf = append(buf, 0, 0, 0, 0, 0, 0)
		copy(buf[2:], a.PCR[:6])
		a.AdaptationFieldLength += 6
	}

	if a.TransportPrivateDataFlag {
		buf = append(buf, 0)
		buf[a.AdaptationFieldLength+1] = a.TransportPrivateDataLen
		a.AdaptationFieldLength++
		for i := uint8(0); i < a.TransportPrivateDataLen; i++ {
			buf = append(buf, 0)
		}
		copy(buf[a.AdaptationFieldLength+1:], a.TransportPrivateData)
		a.AdaptationFieldLength += a.TransportPrivateDataLen
	}

	if len(a.StuffingBytes) > 0 {
		for i := 0; i < len(a.StuffingBytes); i++ {
			buf = append(buf, 0)
		}
		copy(buf[a.AdaptationFieldLength+1:], a.StuffingBytes)
		a.AdaptationFieldLength += uint8(len(a.StuffingBytes))
	}

	if a.AdaptationFieldLength == 0 {
		return []byte{0}
	}

	buf[0] = a.AdaptationFieldLength

	return buf
}

func DecodeAdaptationField(adfControl uint8, b []byte) *AdaptationField {
	adf := &AdaptationField{}
	adf.Type = adfControl
	adf.AdaptationFieldLength = b[0]

	if adf.AdaptationFieldLength > 0 {
		counter := NewCounter[uint8]()

		adfBuf := b[1 : adf.AdaptationFieldLength+1]

		if len(adfBuf) > 0 {
			adf.DiscontinuityIndicator = adfBuf[counter.Current()]>>7 == 0x1
			adf.RndAccessIndicator = (adfBuf[counter.Current()]>>6)&0x1 == 0x1
			adf.EsPriorityIndicator = (adfBuf[counter.Current()]>>5)&0x1 == 0x1
			adf.PcrFlag = (adfBuf[counter.Current()]>>4)&0x1 == 0x1
			adf.OpcrFlag = (adfBuf[counter.Current()]>>3)&0x1 == 0x1
			adf.SplicingPointFlag = (adfBuf[counter.Current()]>>2)&0x1 == 0x1
			adf.TransportPrivateDataFlag = (adfBuf[counter.Current()]>>1)&0x1 == 0x1
			adf.AdaptationFieldExtensionFlag = adfBuf[counter.Next()]&0x1 == 0x1
		}

		if adf.PcrFlag {
			adf.PCR[0] = adfBuf[counter.Next()]
			adf.PCR[1] = adfBuf[counter.Next()]
			adf.PCR[2] = adfBuf[counter.Next()]
			adf.PCR[3] = adfBuf[counter.Next()]
			adf.PCR[4] = adfBuf[counter.Next()]
			adf.PCR[5] = adfBuf[counter.Next()]
		}

		if adf.OpcrFlag {
			adf.OPCR[0] = adfBuf[counter.Next()]
			adf.OPCR[1] = adfBuf[counter.Next()]
			adf.OPCR[2] = adfBuf[counter.Next()]
			adf.OPCR[3] = adfBuf[counter.Next()]
			adf.OPCR[4] = adfBuf[counter.Next()]
			adf.OPCR[5] = adfBuf[counter.Next()]
		}

		if adf.SplicingPointFlag {
			adf.SpliceCountdown = adfBuf[counter.Next()]
		}

		if adf.TransportPrivateDataFlag {
			adf.TransportPrivateDataLen = adfBuf[counter.Next()]
			adf.TransportPrivateData = adfBuf[counter.Current() : counter.Current()+adf.TransportPrivateDataLen]
			counter.Seek(adf.TransportPrivateDataLen)
		}

		if adf.AdaptationFieldExtensionFlag {
			counter.Next()
			adf.AdaptationExtension = DecodeAdaptationFieldExtension(adfBuf[counter.Current() : counter.Current()+adfBuf[counter.Current()]])
		}

		adf.StuffingBytes = adfBuf[counter.Current():]
	}

	return adf
}
