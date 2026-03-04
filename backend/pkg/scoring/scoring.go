package scoring

import (
	"context"
	"math"
)

// ApplySolveScore computes the points to record for a solve on a dynamically-scored
// challenge. When dynamic scoring is active (InitialValue > 0 and Decay > 0) it
// recalculates the challenge's current value based on the new solveCount and, if
// the value changed, calls updateFn so the caller can persist the updated score.
//
// Returns the points that should be stored against the solve record.
// When the challenge is not dynamic, currentPoints is returned unchanged.
func ApplySolveScore(
	ctx context.Context,
	initialValue, minValue, decay, currentPoints, solveCount int,
	updateFn func(context.Context, int) error,
) (pointsAtSolve int, err error) {
	if initialValue <= 0 || decay <= 0 {
		return currentPoints, nil
	}
	newPoints := CalculateDynamicScore(initialValue, minValue, decay, solveCount)
	if newPoints == currentPoints {
		return currentPoints, nil
	}
	if err := updateFn(ctx, newPoints); err != nil {
		return 0, err
	}
	return newPoints, nil
}

// CalculateDynamicScore returns the current point value for a challenge
// using the dynamic scoring formula. First solver gets initial points;
// decay is applied from the 2nd solve. When solves > decay, returns minScore.
func CalculateDynamicScore(initial, minScore, decay, solves int) int {
	if initial < minScore {
		initial = minScore
	}
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

	// If solves > decay, value would drop below minimum due to the parabolic formula,
	// so clamp it to minScore. At exactly solves==decay the formula still gives a value
	// slightly above min; that is intentional - min is only guaranteed after decay+1 solves.
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
