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

	// 1e4 = 25 seconds
	SubPixels     = 100
	MaxIterations = 100

	viewHeight = 2

	horizontalCenter = 0.0
	verticalCenter   = 0.0

	bottom = verticalCenter - viewHeight*0.5
	top    = bottom + viewHeight
	left   = horizontalCenter - viewHeight*0.5*float64(Width)/float64(Height)
	//right := left + viewHeight*float64(Width)/float64(Height)

	// px is the real size of each pixel.
	px = viewHeight / float64(Height)
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

const (
	//gIterations = 20
	bIterations = 100
)

func toP(z complex128) int {
	x := real(z)
	x -= left
	x /= px

	y := imag(z)
	y = top - y
	y /= px

	if x < 0 || x >= Width {
		return -1
	}
	if y < 0 || y >= Height {
		return -1
	}
	return int(x) + int(y)*Width
}

func runCmd(cmd *cobra.Command, _ []string) error {
	// At this point usage information has already been printed if obviously incorrect.
	cmd.SilenceUsage = true

	j := transforms.JuliaN{C: complex(0.45, -0.575), N: 6.0}
	l := transforms.Linear{
		Multiply: cmplx.Rect(1.1, 0.3),
		Add:      0.0,
	}

	yChannel := make(chan int)

	go func() {
		for y := 0; y < Height; y++ {
			yChannel <- y
		}
		close(yChannel)
	}()

	paths := make(chan map[int]float64, 10000)

	parallel := runtime.NumCPU()

	low := -4.0
	width := -2.0 * low

	ywg := sync.WaitGroup{}
	ywg.Add(parallel)
	for i := 0; i < parallel; i++ {
		go func() {
			rng := rand.New(rand.NewSource(int64(time.Now().Second())))
			for y := range yChannel {
				ry := low + width*float64(y)/float64(Height)
				px := width / float64(Height)

				path := make([]complex128, MaxIterations)

				//woh := float64(Width) / float64(Height)

				for x := 0; x < Width; x++ {
					rx := low + width*float64(x)/float64(Height)

					p := make(map[int]float64)
					for s := 0; s < SubPixels; s++ {
						// Slightly jitter points.
						sy := ry + px*rng.Float64()
						sx := rx + px*rng.Float64()

						sz := complex(sy, sx)
						//start := sz

						iterations := 0
						for iterations < MaxIterations && cmplx.Abs(sz) < 10 {
							if iterations%2 == 0 {
								sz = j.Next(sz)
								sz = l.Next(sz)
								sz += complex(rng.Float64()*2-1, rng.Float64()*2-1) * 1e-5
							}
							path[iterations] = sz

							iterations++
						}

						if iterations < bIterations {
							for k, z := range path[:iterations] {
								pixel := toP(z)
								if pixel == -1 {
									continue
								}

								p[pixel] += math.Min(0.1*float64(k), 1.0)

							}

						}
					}
					paths <- p
				}
			}

			ywg.Done()
		}()
	}

	iSP := 1.0 / SubPixels
	frequencies := make([]float64, Width*Height)

	brightnessGroup := sync.WaitGroup{}
	brightnessGroup.Add(1)
	go func() {
		for g := range paths {
			for pixel, count := range g {
				frequencies[pixel] += iSP * count
			}
		}
		brightnessGroup.Done()
	}()

	ywg.Wait()
	close(paths)
	brightnessGroup.Wait()

	maxHits := 0.0
	for _, h := range frequencies {
		maxHits = math.Max(h, maxHits)
	}
	if maxHits <= 0.0 {
		maxHits = 1.0
	}
	invMaxHits := 1.0 / maxHits

	img := image.NewRGBA64(image.Rect(0, 0, Width, Height))
	for pixel := 0; pixel < Width*Height; pixel++ {
		x := pixel % Width
		y := pixel / Width

		b := frequencies[pixel] * invMaxHits
		b = math.Pow(b, 0.025)
		//b = 1.0 - b
		if frequencies[pixel] == 0 {
			b = 0.0
		}
		b *= math.MaxUint16

		img.Set(x, y, color.RGBA64{
			R: uint16(b),
			G: uint16(b),
			B: uint16(b),
			A: 0xffff,
		})

	}

	err := os.MkdirAll("out", os.ModePerm)
	if err != nil {
		return err
	}

	f, err := os.Create(fmt.Sprintf("out/%s.png", time.Now().
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
