package ts

type AdaptationFieldExtension struct {
	adaptationExtensionLength uint8
	legalTimeWindowFlag       bool
	piecewiseRateFlag         bool
	seamlessSpliceFlag        bool
	reserved                  uint8
	LTWValidFlag              bool
	LTWOffset                 uint16
	PCWReserved               uint8
	PCWRate                   uint32
	spliceType                uint8
	DTSNextAccessUnit         uint64
}

func DecodeAdaptationFieldExtension(b []byte) *AdaptationFieldExtension {
	return &AdaptationFieldExtension{}
}
