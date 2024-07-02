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

	SubPixels     = 1
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

	j := transforms.JuliaN{C: complex(0.09, -0.575), N: 5.0}
	j2 := transforms.JuliaN{C: complex(0.09, -0.575), N: 6.0}

	yChannel := make(chan int)

	go func() {
		for y := 0; y < Height; y++ {
			yChannel <- y
		}
		close(yChannel)
	}()

	greenChannel := make(chan []PixelBrightness, 10000)
	blueChannel := make(chan []PixelBrightness, 10000)

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
							blues := make([]PixelBrightness, 0, iterations)
							for _, z := range path[:iterations] {
								pixel := toP(z)
								if pixel == -1 {
									continue
								}

								blues = append(blues, PixelBrightness{
									P:          pixel,
									Brightness: 1.0,
								})

							}

							blueChannel <- blues
						} else {
							greens := make([]PixelBrightness, 0, iterations)
							for _, z := range path[:iterations] {
								pixel := toP(z)
								if pixel == -1 {
									continue
								}

								greens = append(greens, PixelBrightness{
									P:          pixel,
									Brightness: 1.0,
								})
							}

							greenChannel <- greens
						}
					}
				}
			}

			ywg.Done()
		}()
	}

	iSP := 1.0 / SubPixels
	green := make([]float64, Width*Height)

	brightnessGroup := sync.WaitGroup{}
	brightnessGroup.Add(2)
	go func() {
		for g := range greenChannel {
			for _, g2 := range g {
				green[g2.P] += g2.Brightness * iSP
			}
		}
		brightnessGroup.Done()
	}()

	blue := make([]float64, Width*Height)
	go func() {
		for b := range blueChannel {
			for _, b2 := range b {
				blue[b2.P] += b2.Brightness * iSP
			}
		}
		brightnessGroup.Done()
	}()

	ywg.Wait()
	close(greenChannel)
	close(blueChannel)
	brightnessGroup.Wait()

	for i, g := range green {
		green[i] = math.Pow(g, 0.2)
	}
	for i, b := range blue {
		blue[i] = math.Pow(b, 0.2)
	}

	maxGreen := 0.0
	for _, g := range green {
		if g > maxGreen {
			maxGreen = g
		}
	}
	if maxGreen <= 0.0 {
		maxGreen = 1.0
	}

	maxBlue := 0.0
	for _, b := range blue {
		if b > maxBlue {
			maxBlue = b
		}
	}
	if maxBlue <= 0.0 {
		maxBlue = 1.0
	}

	img := image.NewRGBA64(image.Rect(0, 0, Width, Height))
	for pixel := 0; pixel < Width*Height; pixel++ {
		x := pixel % Width
		y := pixel / Width

		b := blue[pixel] * math.MaxUint16 * 2.5 / maxBlue
		g := green[pixel] * math.MaxUint16 * 2.0 / maxGreen
		r := 0.0

		if b > math.MaxUint16 {
			bExcess := b - math.MaxUint16
			b = math.MaxUint16
			g += bExcess
			r = bExcess
			if g > math.MaxUint16 {
				r = g - math.MaxUint16
				g = math.MaxUint16
			}
		} else if g > math.MaxUint16 {
			gExcess := g - math.MaxUint16
			g = math.MaxUint16
			b += gExcess
			r = gExcess
			if b > math.MaxUint16 {
				r = b - math.MaxUint16
				b = math.MaxUint16
			}
		}

		img.Set(x, y, color.RGBA64{
			R: uint16(r),
			G: uint16(g),
			B: uint16(b),
			A: 0xffff,
		})

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
