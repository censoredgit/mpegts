package ts

import "errors"

var ErrUnsupportedPsiTable = errors.New("PSI support only PAT, PMT and Data table")

type PSI struct {
	PointerField       uint8
	PointerFillerBytes uint8
	PMT                *PMT
	PAT                *PAT
	parent             *Payload
}

func (p *PSI) encode() []byte {
	var result []byte

	if p.PAT != nil {
		result = p.PAT.encode()
	} else if p.PMT != nil {
		result = p.PMT.encode()
	}

	res := make([]byte, len(result)+1)
	res[0] = p.PointerField
	copy(res[1:], result)
	return res
}

func isPAT(pid uint16) bool {
	return pid == 0x0000
}

func isCAT(pid uint16) bool {
	return pid == 0x0001
}

func isTSDT(pid uint16) bool {
	return pid == 0x0002
}

func isIPMP(pid uint16) bool {
	return pid == 0x0003
}

func isReservedForFuture(pid uint16) bool {
	return pid >= 0x0004 && pid <= 0x000f
}

func isDVBMetadata(pid uint16) bool {
	return pid >= 0x0010 && pid <= 0x001f
}

func isData(pid uint16) bool {
	return (pid >= 0x0010 && pid <= 0x1ffe) && !isDigiCipher(pid)
}

func isNULL(pid uint16) bool {
	return pid == 0x1fff
}

func isDigiCipher(pid uint16) bool {
	return pid == 0x1ffb
}

func DecodePSI(parent *Payload, payload []byte, pid uint16) (*PSI, error) {
	p := &PSI{}
	p.parent = parent

	p.PointerField = payload[0]

	if p.PointerField != 0x0 {
		p.PointerFillerBytes = p.PointerField * 8
	} else {
		p.PointerFillerBytes = 0
	}

	data := payload[1+p.PointerFillerBytes:]

	if isPAT(pid) {
		p.PAT = DecodePAT(data)
	} else if isData(pid) {
		isPMT := false

		for _, v := range p.parent.parent.container.pmtPIDs {
			if v == pid {
				isPMT = true
				break
			}
		}

		if isPMT {
			p.PMT = DecodePMT(data)
		}

	} else {
		return nil, ErrUnsupportedPsiTable
	}

	return p, nil
}
