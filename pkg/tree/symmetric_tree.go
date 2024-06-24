package tree

// Symmetric returns a perfectly-symmetric tree where all branches deviate at the same
// angle.
func Symmetric(layers int, angle float64) *Tree {
	if layers == 0 {
		return nil
	}

	result := &Tree{
		LeftP:      0.5,
		LeftAngle:  angle,
		RightAngle: angle,
		Left:       nil,
		Right:      nil,
	}

	children := Symmetric(layers-1, angle)
	result.Left = children
	result.Right = children

	return result
}
