package usecase

import "math"

func CalculateDynamicScore(initial, min, decay, solves int) int {
	if solves == 0 {
		return initial
	}
	if decay == 0 {
		return initial
	}
	score := int(float64(initial-min)/math.Pow(2, float64(solves)/float64(decay))) + min
	if score < min {
		return min
	}
	return score
}
