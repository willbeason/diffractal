package tree

import (
	"math"
	"math/rand"
)

func RandomBalanced(layers int, r *rand.Rand) *Tree {
	if layers == 0 {
		return nil
	}

	pLeft := r.Float64()*0.6 + 0.2
	angle := math.Pi / 3.0 * (r.Float64()*0.6 + 0.2)

	leftAngle, rightAngle := 0.0, 0.0
	if pLeft < 0.5 {
		leftAngle = angle
		rightAngle = math.Asin((pLeft / (1.0 - pLeft)) * math.Sin(angle))
	} else {
		leftAngle = math.Asin(((1.0 - pLeft) / pLeft) * math.Sin(angle))
		rightAngle = angle
	}

	result := &Tree{
		LeftP:      pLeft,
		LeftAngle:  leftAngle,
		RightAngle: rightAngle,
		Left:       RandomBalanced(layers-1, r),
		Right:      RandomBalanced(layers-1, r),
	}

	return result
}
