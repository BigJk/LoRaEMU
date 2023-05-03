package emu

import (
	"errors"
	"github.com/BigJk/loraemu/lora"
	"math"
)

type received struct {
	Start int64   `json:"start"`
	Stop  int64   `json:"stop"`
	Gain  float64 `json:"gain"`
}

// Node represents a LoRa device in the emulator.
type Node struct {
	ID     string                 `json:"id"`
	Online bool                   `json:"online"`
	X      float64                `json:"x"`
	Y      float64                `json:"y"`
	Z      float64                `json:"z"`
	TXGain float64                `json:"txGain"`
	RXSens float64                `json:"rxSens"`
	SNR    int                    `json:"snr"`
	Icon   string                 `json:"icon"`
	Meta   map[string]interface{} `json:"meta"`

	receiving    []received
	sendingUntil int64
}

func (n Node) DistanceTo(other Node) float64 {
	return math.Sqrt(math.Pow(n.X-other.X, 2) + math.Pow(n.Y-other.Y, 2) + math.Pow(n.Z-other.Z, 2))
}

func (n Node) PathLoss(other Node, distanceRef float64, gamma float64, freq float64) float64 {
	return lora.LogDistance(n.DistanceTo(other)*1000, distanceRef, gamma, freq)
}

func (n Node) LatLng() (float64, float64) {
	// Offset so that we don't have negative values
	x := n.X + 100
	y := n.Y + 100

	lat := math.Acos(math.Sqrt(x*x+y*y)/6371) * 180 / math.Pi
	lng := math.Atan2(y, x) * 180 / math.Pi

	return lat, lng
}

func (n Node) Valid() error {
	if len(n.ID) == 0 {
		return errors.New("no id")
	}
	return nil
}
