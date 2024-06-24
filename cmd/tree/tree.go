package main

import (
	"context"
	"fmt"
	"github.com/spf13/cobra"
	"github.com/willbeason/tree-fractal/pkg/geometry"
	"github.com/willbeason/tree-fractal/pkg/tree"
	"image"
	"image/color"
	"image/png"
	"math"
	"math/rand"
	"os"
	"time"
)

func mainCmd() *cobra.Command {
	cmd := &cobra.Command{
		Args: cobra.ExactArgs(0),
		RunE: runCmd,
	}

	return cmd
}

func runCmd(cmd *cobra.Command, _ []string) error {
	// At this point usage information has already been printed if obviously incorrect.
	cmd.SilenceUsage = true

	r := rand.New(rand.NewSource(int64(time.Now().Second())))

	nLayers := 20
	//fractal := tree.RandomBalanced(nLayers, r)
	//fractal := tree.BalancedConstant(nLayers, 0.6, 0.4)
	fractal := tree.Symmetric(nLayers, 0.6)
	fractal = &tree.Tree{
		LeftP:      1.0,
		LeftAngle:  0.0,
		RightAngle: 0.0,
		Left:       fractal,
		Right:      nil,
	}

	width := 2560
	height := 1440

	counts := make([]int, width*height)

	viewScale := 3.0

	horizontalCenter := 0.5
	verticalCenter := 2.5

	bottom := verticalCenter - viewScale
	top := verticalCenter + viewScale
	left := horizontalCenter - viewScale*float64(width)/float64(height)
	right := horizontalCenter + viewScale*float64(width)/float64(height)

	fmt.Println(bottom, top)
	fmt.Println(left, right)

	viewWidth := right - left
	viewHeight := top - bottom

	curNode := fractal
	offset := geometry.XY{X: 0.0, Y: 0.0}
	angle := 0.0
	scale := 1.0

	for p := 0; p < 1e7; p++ {
		xy := curNode.RandomPoint(r)

		xy = Rescale(xy, scale, angle, offset)

		if xy.X < left || xy.X > right {
			panic("X is out of bounds")
		}
		if xy.Y < bottom || xy.Y > top {
			panic(fmt.Sprintf("Y is out of bounds: %.02f", xy.Y))
		}

		px := int(((xy.X - left) / viewWidth) * float64(width))
		py := int(((top - xy.Y) / viewHeight) * float64(height))

		cell := px + py*width
		counts[cell]++

		nextNode := curNode.Continue(r)
		switch nextNode {
		case tree.None:
			scale = 1.0
			angle = 0.0
			offset = geometry.XY{X: 0.0, Y: 0.0}
			curNode = fractal
		case tree.Left:
			dOffset := curNode.LeftOrigin()
			dOffset = Rescale(dOffset, scale, angle, offset)
			offset.X = dOffset.X
			offset.Y = dOffset.Y

			angle += curNode.LeftAngle
			scale *= curNode.LeftP
			curNode = curNode.Left
		case tree.Right:
			dOffset := curNode.RightOrigin()
			dOffset = Rescale(dOffset, scale, angle, offset)
			offset.X = dOffset.X
			offset.Y = dOffset.Y

			angle -= curNode.RightAngle
			scale *= 1.0 - curNode.LeftP
			curNode = curNode.Right
		}
	}

	maxCount := 0
	for _, c := range counts {
		if c > maxCount {
			maxCount = c
		}
	}

	for i, c := range counts {
		counts[i] = c * math.MaxUint16 / maxCount
	}

	img := image.NewGray16(image.Rect(0, 0, width, height))
	for i, c := range counts {
		x := i % width
		y := i / width

		img.Set(x, y, color.Gray16{Y: uint16(c)})
	}

	f, err := os.Create("out.png")
	if err != nil {
		return err
	}

	err = png.Encode(f, img)
	if err != nil {
		return err
	}

	return nil
}

func main() {
	ctx := context.Background()

	err := mainCmd().ExecuteContext(ctx)
	if err != nil {
		// At this point the error has already been printed; no need to print again.
		os.Exit(1)
	}
}

func Rescale(xy geometry.XY, scale float64, angle float64, offset geometry.XY) geometry.XY {
	x := xy.X * scale
	y := xy.Y * scale

	x2 := x*math.Cos(angle) - y*math.Sin(angle) + offset.X
	y2 := x*math.Sin(angle) + y*math.Cos(angle) + offset.Y

	return geometry.XY{X: x2, Y: y2}
}
