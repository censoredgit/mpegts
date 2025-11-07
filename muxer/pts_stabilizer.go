package muxer

const NoPts = -1
const tsHz int64 = 90

type PtsStabilizer struct {
	primaryPid   int
	pFirstPts    int64
	pLastPts     int64
	pLastTimeMs  int64
	lastPtsOfPid map[int]int64
}

func NewPtsStabilizer(primaryPid int) *PtsStabilizer {
	return &PtsStabilizer{
		primaryPid:   primaryPid,
		lastPtsOfPid: make(map[int]int64),
	}
}

func (p *PtsStabilizer) ComputePrimaryPts(pts int64, currentTimeMs int64) int64 {
	if p.pFirstPts == 0 {
		p.pFirstPts = pts
	}

	p.pLastPts = pts
	p.pLastTimeMs = currentTimeMs

	return (pts - p.pFirstPts) / 1000 * tsHz
}

func (p *PtsStabilizer) ComputePts(pid int, currentTimeMs int64) int64 {
	pts := int64(0)
	if _, exists := p.lastPtsOfPid[pid]; !exists {
		p.lastPtsOfPid[pid] = 0
	}

	if currentTimeMs > p.pLastTimeMs {
		pts = p.pLastPts + (currentTimeMs - p.pLastTimeMs)
	} else {
		pts = p.pLastPts - (p.pLastTimeMs - currentTimeMs)
	}

	if pts > p.lastPtsOfPid[pid] {
		p.lastPtsOfPid[pid] = pts
	}

	return (pts - p.pFirstPts) / 1000 * tsHz
}
