package lora

import "math"

// PacketConfig represents a LoRa packet configuration that is needed to calculate the airtime.
type PacketConfig struct {
	PayloadLen              float64 `json:"payloadLen"`
	PreambleLen             float64 `json:"preambleLen"`
	SpreadingFactor         float64 `json:"spreadingFactor"`
	BandWidth               float64 `json:"bandWidth"`
	CodingRate              float64 `json:"codingRate"`
	CRC                     bool    `json:"crc"`
	ExplicitHeader          bool    `json:"explicitHeader"`
	LowDataRateOptimization bool    `json:"lowDataRateOptimization"`
}

// PacketConfigDefault represents a default packet config.
var PacketConfigDefault = PacketConfig{
	PayloadLen:              1,
	PreambleLen:             6,
	SpreadingFactor:         7,
	BandWidth:               125,
	CodingRate:              5,
	CRC:                     false,
	ExplicitHeader:          false,
	LowDataRateOptimization: false,
}

func (pc PacketConfig) PayloadValid() bool {
	return pc.PayloadLen >= 1 && pc.PayloadLen < 255
}

func (pc PacketConfig) PreambleValid() bool {
	return pc.PreambleLen >= 6 && pc.PreambleLen <= 655365
}

func (pc PacketConfig) SymbolTime() float64 {
	return math.Pow(2, pc.SpreadingFactor) / pc.BandWidth
}

func (pc PacketConfig) SymbolRate() float64 {
	return 1000 / pc.SymbolTime()
}

func (pc PacketConfig) Throughput() float64 {
	return ((8 * pc.PayloadLen) / pc.TimeTotal()) * 1000
}

func (pc PacketConfig) NPayload() float64 {
	payloadBit := 8 * pc.PayloadLen
	payloadBit -= 4 * pc.SpreadingFactor
	payloadBit += 28

	if pc.CRC {
		payloadBit += 16
	}

	if pc.ExplicitHeader {
		payloadBit += 20
	}

	payloadBit = math.Max(payloadBit, 0)

	bitsPerSymbol := pc.SpreadingFactor
	if pc.LowDataRateOptimization {
		bitsPerSymbol = pc.SpreadingFactor - 2.0
	}

	payloadSymbol := math.Ceil(payloadBit/4/bitsPerSymbol) * pc.CodingRate
	payloadSymbol += 8

	return payloadSymbol
}

func (pc PacketConfig) NPreamble() float64 {
	return pc.PreambleLen + 4.25
}

// TimePayload represents the time the payload alone will be on air.
func (pc PacketConfig) TimePayload() float64 {
	return pc.NPayload() * pc.SymbolTime()
}

// TimePreamble represents the time the preamble alone will be on air.
func (pc PacketConfig) TimePreamble() float64 {
	return pc.NPreamble() * pc.SymbolTime()
}

// TimeTotal represents the total time the packet will be on air.
func (pc PacketConfig) TimeTotal() float64 {
	return pc.TimePreamble() + pc.TimePayload()
}
