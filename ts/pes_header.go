package ts

type PESHeader struct {
	Marker                 uint8
	ScramblingControl      uint8
	Priority               bool
	DataAlignmentIndicator bool
	Copyright              bool
	OriginalOrCopy         bool
	PTSDTSIndicator        uint8
	ESCRFlag               bool
	ESRateFlag             bool
	DSMTrickModeFLag       bool
	AdditionalCopyInfoFlag bool
	PESCRCFlag             bool
	PESExtensionFlag       bool
	PESHeaderDataLength    uint8
	Data                   *PESHeaderData
}

type PESHeaderData struct {
	PTS uint64
	DTS uint64
}

func (phd *PESHeaderData) encode(pts bool, dts bool) []byte {
	if dts {
		result := make([]byte, len(uintToPts(phd.PTS, dts))+len(uintToDts(phd.DTS)))
		copy(result[:len(uintToPts(phd.PTS, dts))], uintToPts(phd.PTS, dts))
		copy(result[len(uintToPts(phd.DTS, dts)):], uintToDts(phd.DTS))
		return result
	} else if pts {
		return uintToPts(phd.PTS, dts)
	} else {
		return make([]byte, 0)
	}
}
