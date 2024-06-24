package transforms

import (
	"github.com/willbeason/tree-fractal/pkg/geometry"
	"math/rand"
)

// InitialTransform represents the initial distribution of a fractal.
type InitialTransform interface {
	First() geometry.XY
}

// A Transform iterates a passed point.
type Transform interface {
	Next(geometry.XY, *rand.Rand) geometry.XY
}

type TransformProbability struct {
	Transform
	Probability float64
}

type ProbabilisticTransform struct {
	InitialTransform
	Transforms []TransformProbability
}

func (pt ProbabilisticTransform) Next(xy geometry.XY, rng *rand.Rand) geometry.XY {
	p := rng.Float64()

	for _, maxProb := range pt.Transforms {
		if p < maxProb.Probability {
			return maxProb.Next(xy, rng)
		}
	}

	return pt.First()
}

var _ Transform = ProbabilisticTransform{}
