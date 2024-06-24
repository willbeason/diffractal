package tree

import (
	"github.com/willbeason/tree-fractal/pkg/geometry"
	"math"
	"math/rand"
)

const (
	// The LengthFactor is how much longer branches are than they are wide, counting the turn.
	// A LengthFactor of 2.0 ensures the branch is at least as long as it is wide after the turn.
	LengthFactor = 1.5
)

// A Tree is really a junction in a fractal.
//
// The recursive structure mimics the actual rendered structures.
type Tree struct {
	// LeftP is the proportion of probability/area dedicated to the left branch.
	// RightP is 1.0 - LeftP.
	LeftP float64

	// LeftAngle is the angle to which the Left branch is angled from Tree.
	// Measured in radians counter-clockwise from the current tree's direction.
	// Should generally be between 0.0 and pi/2 to avoid turns less severe
	// than right angles or shapes looping into themselves; otherwise visual artifacts may occur.
	LeftAngle float64

	// RightAngle is the same as above, but for the Right branch.
	// Measured in radians clockwise.
	RightAngle float64

	// Left and Right are the Tree's branches.
	// If empty, the tree does not continue.
	Left, Right *Tree
}

func (tree *Tree) LeftOrigin() geometry.XY {
	length := LengthFactor - 0.5*tree.LeftAngle*tree.LeftP
	return geometry.XY{
		X: -length * math.Sin(tree.LeftAngle),
		Y: length * math.Cos(tree.LeftAngle),
	}
}

func (tree *Tree) RightOrigin() geometry.XY {
	// Same as above, but mirrored around the split.
	w := 1.0 - tree.LeftP
	length := LengthFactor - 0.5*tree.RightAngle*w
	return geometry.XY{
		X: 1.0 + length*math.Sin(tree.RightAngle) - w*math.Cos(tree.RightAngle),
		Y: length*math.Cos(tree.RightAngle) + w*math.Sin(tree.RightAngle),
	}
}

type Continue int

const (
	None Continue = iota
	Left
	Right
)

// Continue is what the next point to generate should be.
func (tree *Tree) Continue(r *rand.Rand) Continue {
	rContinue := r.Float64()

	if rContinue < tree.LeftP*tree.LeftP {
		if tree.Left != nil {
			return Left
		}
		return None
	} else if rContinue < (1.0-tree.LeftP)*(1.0-tree.LeftP)+tree.LeftP*tree.LeftP {
		if tree.Right != nil {
			return Right
		}
		return None
	}
	return None
}

// RandomPoint returns a random point for the tree, relative to the tree's own
// scaling and angle.
func (tree *Tree) RandomPoint(r *rand.Rand) geometry.XY {
	// isLeft is whether the point will be generated on the left branch.
	isLeft := r.Float64() < tree.LeftP

	angle := tree.LeftAngle
	if !isLeft {
		angle = tree.RightAngle
	}

	width := tree.LeftP
	if !isLeft {
		width = 1.0 - tree.LeftP
	}

	// Since the area of a slice of a circle is half the angle times the radius squared,
	// This is the probability of the point being generated being on the branch's curve.
	totalArea := LengthFactor
	turnArea := 0.5 * angle * width
	turnP := turnArea / totalArea
	isTurn := r.Float64() < turnP

	rWidth := r.Float64()
	rHeight := r.Float64()
	if isTurn {
		// Quadratic distribution favoring the outside.
		rWidth = math.Sqrt(rWidth)
		// Height here is the angle along the circle.
		rHeight *= angle
	} else {
		length := LengthFactor - 0.5*angle*width
		rHeight *= length
	}
	rWidth *= width

	// Default to generating as a left point.
	dx, dy := 0.0, 0.0
	if isTurn {
		dx = rWidth * math.Cos(rHeight)
		dy = rWidth * math.Sin(rHeight)
	} else {
		dx = rWidth*math.Cos(angle) - rHeight*math.Sin(angle)
		dy = rWidth*math.Sin(angle) + rHeight*math.Cos(angle)
	}

	if !isLeft {
		// Mirror about split axis.
		dx = 1.0 - dx
	}

	return geometry.XY{X: dx, Y: dy}
}
