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
	Width     = 2560
	Height    = 1440
	invHeight = 1.0 / Height

	// 1e4 = 25 seconds
	SubPixels     = 1000
	MaxIterations = 100

	viewHeight = 1.2

	horizontalCenter = 0.0
	verticalCenter   = -viewHeight / 2.0

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

func toP(z complex128) (int, int, float64, float64) {
	x := real(z)
	x -= left
	x /= px

	y := imag(z)
	y = top - y
	y /= px

	dx := x - math.Floor(x)
	dy := y - math.Floor(y)

	if x < -1 || x >= Width {
		return -2, -2, 0.0, 0.0
	}
	if y < -1 || y >= Height {
		return -2, -2, 0.0, 0.0
	}
	//return int(x) + int(y)*Width, dx, dy
	return int(math.Floor(x)), int(math.Floor(y)), dx, dy
}

func runCmd(cmd *cobra.Command, _ []string) error {
	// At this point usage information has already been printed if obviously incorrect.
	cmd.SilenceUsage = true
	j := transforms.JuliaN{C: complex(0.09, -0.575), N: 5.0}
	j2 := transforms.JuliaN{C: complex(0.09, -0.575), N: 6.0}

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
			path := make([]complex128, MaxIterations)

			for y := range yChannel {
				ry := low + width*float64(y)*invHeight
				// px is the simulated pixel starting point size, not viewport pixel size.
				px := width / float64(Height)

				//woh := float64(Width) / float64(Height)

				for x := 0; x < Width; x++ {
					rx := low + width*float64(x)*invHeight

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
							} else {
								sz = j2.Next(sz)
							}
							path[iterations] = sz

							iterations++
						}

						if iterations < bIterations {
							for k, z := range path[:iterations] {

								lx, ly, dx, dy := toP(z)
								if lx == -2 {
									continue
								}

								z00 := (1.0 - dx) * (1.0 - dy)
								z01 := dx * (1.0 - dy)
								z10 := (1.0 - dx) * dy
								z11 := dx * dy

								// int(x) + int(y)*Width
								baseBrightness := math.Min(0.1*float64(k), 1.0)
								if lx >= 0 && ly >= 0 {
									p[lx+ly*Width] += baseBrightness * z00
								}
								if lx < Width-1 && ly >= 0 {
									p[lx+1+ly*Width] += baseBrightness * z01
								}
								if lx >= 0 && ly < Height-1 {
									p[lx+(ly+1)*Width] += baseBrightness * z10
								}
								if lx < Width-1 && ly < Height-1 {
									p[lx+1+(ly+1)*Width] += baseBrightness * z11
								}
							}

						}
					}
					paths <- p
				}
			}

			ywg.Done()
		}()
	}

	//iSP := 1.0 / SubPixels
	frequencies := make([]float64, Width*Height)

	brightnessGroup := sync.WaitGroup{}
	brightnessGroup.Add(1)
	go func() {
		for g := range paths {
			for pixel, count := range g {
				frequencies[pixel] += count
			}
		}
		brightnessGroup.Done()
	}()

	ywg.Wait()
	close(paths)
	brightnessGroup.Wait()

	for i, f := range frequencies {
		frequencies[i] = math.Pow(f, 0.2)
	}

	maxHits := 0.0
	for _, h := range frequencies {
		maxHits = math.Max(h, maxHits)

	}
	if maxHits <= 0.0 {
		maxHits = 1.0
	}
	fmt.Println("Max hits", maxHits)
	invMaxHits := 1.0 / maxHits

	img := image.NewRGBA64(image.Rect(0, 0, Width, Height))
	baseWeight := math.MaxUint16 * 2.5 * invMaxHits
	for pixel := 0; pixel < Width*Height; pixel++ {
		x := pixel % Width
		y := pixel / Width

		b := frequencies[pixel] * baseWeight
		g := 0.0
		r := 0.0

		if b > math.MaxUint16 {
			bExcess := b - math.MaxUint16
			b = math.MaxUint16
			g = bExcess
			r = bExcess
		}

		img.Set(x, y, color.RGBA64{
			R: uint16(r),
			G: uint16(g),
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
