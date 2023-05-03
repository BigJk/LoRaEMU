package lora

import "math"

// FSPL represents the Free-space path loss based on a distance and the frequency (in MHz) of a signal.
//
// https://en.wikipedia.org/wiki/Free-space_path_loss
func FSPL(distance float64, freq float64) float64 {
	return 20*math.Log10(distance) + 20*math.Log10(freq) + 32.45
}

// LogDistance represents the log-distance path loss model.
//
// - distance is the distance between the nodes
// - distanceRef is the reference distance in the same unit as distance (usually 1 km (or 1 mile) for a large cell and 1 m to 10 m for a microcell)
// - gamma is the path loss exponent (example values are: free space = 2, urban area = 2.7 - 3.5, obstructed in building = 4 - 6)
// - freq is the frequency at which the signal is sent
//
// https://en.wikipedia.org/wiki/Log-distance_path_loss_model
func LogDistance(distance float64, distanceRef float64, gamma float64, freq float64) float64 {
	return FSPL(distanceRef, freq) + 10*gamma*math.Log10(distance/distanceRef)
}
