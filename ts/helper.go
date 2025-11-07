package ts

import (
	"encoding/binary"
)

func ptsToUint(b []byte) uint64 {
	PTS := uint64(0)
	PTS |= uint64(b[0]&0xE) >> 1
	PTS <<= 15
	PTS |= uint64(binary.BigEndian.Uint16(b[1:3])) >> 1
	PTS <<= 15
	PTS |= uint64(binary.BigEndian.Uint16(b[3:5])) >> 1
	return PTS
}

func dtsToUint(b []byte) uint64 {
	PTS := uint64(0)
	PTS |= uint64(b[0]&0xE) >> 1
	PTS <<= 15
	PTS |= uint64(binary.BigEndian.Uint16(b[1:3])) >> 1
	PTS <<= 15
	PTS |= uint64(binary.BigEndian.Uint16(b[3:5])) >> 1
	return PTS
}

func uintToPts(u uint64, hasDTS bool) []byte {
	buf := make([]byte, 5)
	if hasDTS {
		buf[0] = 0b0011_0000 | uint8((u>>29)&0xf)
	} else {
		buf[0] = 0b0010_0000 | uint8((u>>29)&0xf)
	}

	buf[0] |= 0x1
	buf[1] = uint8((u >> 22) & 0xff)
	buf[2] = uint8((u >> 14) & 0xfe)
	buf[2] |= 0x1
	buf[3] = uint8((u >> 7) & 0xff)
	buf[4] = uint8(u&0x7f) << 1
	buf[4] |= 0x1
	return buf[:]
}

func uintToDts(u uint64) []byte {
	buf := make([]byte, 5)
	buf[0] = 0b0001_0000 | uint8((u>>29)&0xf)
	buf[0] |= 0x1
	buf[1] = uint8((u >> 22) & 0xff)
	buf[2] = uint8((u >> 14) & 0xfe)
	buf[2] |= 0x1
	buf[3] = uint8((u >> 7) & 0xff)
	buf[4] = uint8(u&0x7f) << 1
	buf[4] |= 0x1
	return buf[:]
}

func computeCRC32(bs []byte) uint32 {
	o := uint32(0xffffffff)
	for _, b := range bs {
		for i := 0; i < 8; i++ {
			if (o >= uint32(0x80000000)) != (b >= uint8(0x80)) {
				o = (o << 1) ^ 0x04C11DB7
			} else {
				o = o << 1
			}
			b <<= 1
		}
	}
	return o
}

type Counter[T uint8 | uint16 | uint32 | uint64 | int] struct {
	u T
}

func NewCounter[T uint8 | uint16 | uint32 | uint64 | int]() *Counter[T] {
	return NewCounterOffset[T](0)
}

func NewCounterOffset[T uint8 | uint16 | uint32 | uint64 | int](o T) *Counter[T] {
	return &Counter[T]{o}
}

func (c *Counter[T]) Next() T {
	defer c.Add()
	return c.u
}

func (c *Counter[T]) Current() T {
	return c.u
}

func (c *Counter[T]) Add() {
	c.u++
}

func (c *Counter[T]) Seek(l T) {
	c.u += l
}
