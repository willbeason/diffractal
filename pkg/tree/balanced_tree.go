package tree

import "math"

// BalancedConstant returns a perfectly-balanced tree where all branches deviate with the same proportions.
// Angle is the deviation for the smaller branch, so right if pLeft > 0.5.
func BalancedConstant(layers int, angle float64, pLeft float64) *Tree {
	if layers == 0 {
		return nil
	}

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
		Left:       nil,
		Right:      nil,
	}

	children := BalancedConstant(layers-1, angle, pLeft)
	result.Left = children
	result.Right = children

	return result
}
