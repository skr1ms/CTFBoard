package competition

import "math"

func CalculateDynamicScore(initial, minScore, decay, solves int) int {
	if solves <= 0 {
		return initial
	}
	// First solver gets max points. Decay starts from 2nd solve.
	// solve_count for formula = solves - 1
	solveCount := float64(solves - 1)

	if decay == 0 {
		decay = 1
	}

	// Formula: Initial + ((Min - Initial) / (Decay^2)) * (SolveCount^2)
	// Value drops quadratically until 'decay' solves is reached.

	// If solves >= decay, value is minimum (to avoid values going below min or back up if parabola)
	if solves > decay {
		return minScore
	}

	decayFloat := float64(decay)
	value := float64(initial) + ((float64(minScore)-float64(initial))/(decayFloat*decayFloat))*(solveCount*solveCount)

	score := int(math.Ceil(value))
	if score < minScore {
		return minScore
	}
	return score
}
