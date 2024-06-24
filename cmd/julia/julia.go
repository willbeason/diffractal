package main

import (
	"context"
	"fmt"
	"github.com/spf13/cobra"
	"github.com/willbeason/tree-fractal/pkg/transforms"
	"image"
	"image/color"
	"image/png"
	"math"
	"math/cmplx"
	"math/rand"
	"os"
	"runtime"
	"sync"
	"time"
)

const (
	Width  = 2560
	Height = 1440

	SubPixels     = 10
	MaxIterations = 1000
)

func mainCmd() *cobra.Command {
	cmd := &cobra.Command{
		Args: cobra.ExactArgs(0),
		RunE: runCmd,
	}

	return cmd
}

type PixelBrightness struct {
	P          int
	Brightness float64
}

func runCmd(cmd *cobra.Command, _ []string) error {
	// At this point usage information has already been printed if obviously incorrect.
	cmd.SilenceUsage = true

	viewHeight := 2.25

	horizontalCenter := 0.0
	verticalCenter := 0.0

	bottom := verticalCenter - viewHeight*0.5
	top := bottom + viewHeight
	left := horizontalCenter - viewHeight*0.5*float64(Width)/float64(Height)
	//right := left + viewHeight*float64(Width)/float64(Height)

	// px is the real size of each pixel.
	px := viewHeight / float64(Height)

	j := transforms.JuliaN{C: complex(0.7, 0.42), N: 6.0}

	yChannel := make(chan int)

	go func() {
		for y := 0; y < Height; y++ {
			yChannel <- y
		}
		close(yChannel)
	}()

	brightChannel := make(chan PixelBrightness, 10000)

	parallel := runtime.NumCPU()

	ywg := sync.WaitGroup{}
	ywg.Add(parallel)
	for i := 0; i < parallel; i++ {
		go func() {
			rng := rand.New(rand.NewSource(int64(time.Now().Second())))
			for y := range yChannel {
				ry := top - px*float64(y)

				for x := 0; x < Width; x++ {
					b := 0.0

					p := x + y*Width
					rx := left + px*float64(x)

					for s := 0; s < SubPixels; s++ {
						// Slightly jitter points.
						sy := ry + px*rng.Float64()
						sx := rx + px*rng.Float64()

						sz := complex(sy, sx)

						iterations := 0
						for iterations < MaxIterations && cmplx.Abs(sz) < math.Pow(math.MaxFloat64, 1.0/6.0) {
							sz = j.Next(sz)
							iterations++
						}

						if iterations >= MaxIterations {
							continue
						}

						pB := float64(iterations) + 1.0 - math.Log(math.Log(cmplx.Abs(sz)))/math.Log(6.0)
						b += pB
					}

					brightChannel <- PixelBrightness{
						P:          p,
						Brightness: b,
					}
				}

			}
			ywg.Done()
		}()
	}

	brightness := make([]float64, Width*Height)
	bwg := sync.WaitGroup{}
	bwg.Add(1)
	go func() {
		for b := range brightChannel {
			brightness[b.P] += b.Brightness
		}
		bwg.Done()
	}()

	ywg.Wait()
	close(brightChannel)
	bwg.Wait()

	maxBrightness := 0.0
	for _, b := range brightness {
		if b > maxBrightness {
			maxBrightness = b
		}
	}

	for i := range brightness {
		brightness[i] /= maxBrightness
	}

	lightBlue := color.RGBA64{
		R: 0x7fff,
		G: 0xafff,
		B: 0xffff,
		A: 0x0,
	}

	img := image.NewRGBA64(image.Rect(0, 0, Width, Height))
	for i, br := range brightness {
		x := i % Width
		y := i / Width

		r := uint16(float64(lightBlue.R) * br)
		g := uint16(float64(lightBlue.G) * br)
		b := uint16(float64(lightBlue.B) * br)

		img.Set(x, y, color.RGBA64{
			R: r,
			G: g,
			B: b,
			A: uint16(0xffff),
		})
	}

	f, err := os.Create(fmt.Sprintf("out-%s.png", time.Now().
		Format("20060102150405")))
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
