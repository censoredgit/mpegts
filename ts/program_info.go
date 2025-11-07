package ts

type ProgramInfo struct {
	Descriptors []*Descriptor
}

func DecodeProgramInfo(b []byte) *ProgramInfo {
	return &ProgramInfo{
		Descriptors: DecodeDescriptors(b),
	}
}
