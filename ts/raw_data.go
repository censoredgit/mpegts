package ts

type RawData struct {
	parent *Payload
	Data   []byte
}

func NewRawData(parent *Payload, b []byte) *RawData {
	return &RawData{parent, b}
}

func (p *RawData) encode() []byte {
	return p.Data
}
