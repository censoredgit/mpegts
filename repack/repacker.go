package repack

import (
	"context"
	"errors"
	"io"
	"mpegts/ts"
)

type RePacker struct {
	in  io.Reader
	out io.Writer
	tsc *ts.Container
}

func NewRePacker(in io.Reader, out io.Writer) *RePacker {
	return &RePacker{
		in:  in,
		out: out,
		tsc: ts.NewContainer(),
	}
}

func (p *RePacker) Run(ctx context.Context) error {
	buf := make([]byte, ts.PacketSize)
	var tsp *ts.Packet

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			_, err := io.ReadFull(p.in, buf)
			if err != nil {
				if errors.Is(err, io.EOF) {
					return nil
				}
				return err
			}
			tsp, err = p.tsc.DecodePacket(buf)
			if err != nil {
				return err
			}

			_, err = p.out.Write(tsp.Encode())
			if err != nil {
				return err
			}
		}
	}
}
