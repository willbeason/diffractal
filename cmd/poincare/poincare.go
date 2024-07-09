package main

import (
	"context"
	"fmt"
	"github.com/spf13/cobra"
	"github.com/willbeason/diffeq-go/pkg/equations"
	"github.com/willbeason/diffeq-go/pkg/models"
	"github.com/willbeason/diffeq-go/pkg/solvers/order2"
	"image"
	"image/color"
	"image/png"
	"math"
	"math/rand"
	"os"
	"runtime"
	"sort"
	"sync"
	"time"
)

const (
	Width  = 2560
	Height = 1440

	MinX = -30
	MaxX = 30
	MinY = 0.2
	MaxY = 4

	StartCycles = 10000

	// Cycles = 1e6 => 15 seconds
	Cycles = 1e9

	InvDX = float64(Width) / (MaxX - MinX)
	InvDY = float64(Height) / (MaxY - MinY)

	DX = 1.0 / InvDX
	DY = 1.0 / InvDY
)

var (
	minY  = 100.0
	maxY  = -100.0
	minYP = 100.0
	maxYP = -100.0
)

func toPixel(y, yp float64) int {
	//minY = math.Min(y, minY)
	//maxY = math.Max(y, maxY)
	//minYP = math.Min(yp, minYP)
	//maxYP = math.Max(yp, maxYP)

	if y > MaxX {
		return -1
	}
	if y < MinX {
		return -1
	}
	if yp > MaxY {
		return -1
	}
	if yp < MinY {
		return -1
	}

	px := (y - MinX) * InvDX
	py := (yp - MinY) * InvDY

	return int(py)*Width + int(px)
}

func work(eq equations.SecondOrder, solver order2.Solver, _, y0, yp0, h float64, n int, out chan map[int]int) (float64, float64) {
	y := y0
	yp := yp0

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	result := make(map[int]int)
	for i := 0; i < n; i++ {
		y, yp = order2.Solve(solver, eq, 0.0, y, yp, h, 50)
		result[toPixel(yp, y)]++

		y += (-0.5 + rng.Float64()) * DY * 2
		yp += (-0.5 + rng.Float64()) * DX * 2
	}

	if out != nil {
		out <- result
	}

	return y, yp
}

func reduce(in chan map[int]int, out []int) {
	for i := range in {
		for k, v := range i {
			if k >= len(out) || k < 0 {
				continue
			}
			out[k] += v
		}
	}
}

func runCmd(cmd *cobra.Command, _ []string) error {
	// At this point usage information has already been printed if obviously incorrect.
	cmd.SilenceUsage = true

	spring := models.DuffingOscillator{
		Delta:     0.018,
		Alpha:     0.22,
		Beta:      3.3,
		Gamma:     32.657,
		Frequency: 2.03,
	}

	fmt.Println(spring.Gamma)

	results := make(chan map[int]int, 100000)

	wg := sync.WaitGroup{}
	nWorkers := runtime.NumCPU()
	//nWorkers = 1
	wg.Add(nWorkers)

	for i := 0; i < nWorkers; i++ {
		go func() {
			rng := rand.New(rand.NewSource(time.Now().Unix() + int64(i)))
			y0 := (MinY+MaxY)*0.5 + 10*DY*rng.Float64()
			yp0 := (MinX+MaxX)*0.5 + 10*DX*rng.Float64()

			h := 2 * math.Pi / spring.Frequency
			rk4 := order2.NewRungeKuttaSolver(order2.RK4())
			y0, yp0 = work(spring.Acceleration, rk4, 0.0, y0, yp0, h, StartCycles, nil)

			work(spring.Acceleration, rk4, 0.0, y0, yp0, h, Cycles, results)
			//t0 += h * Cycles
			//for s := 0; s < SimulationsPerWorker; s++ {
			//}

			wg.Done()
		}()
	}

	wg2 := sync.WaitGroup{}
	wg2.Add(1)
	counts := make([]int, Width*Height)

	go func() {
		reduce(results, counts)
		wg2.Done()
	}()

	wg.Wait()
	close(results)
	wg2.Wait()

	img := image.NewRGBA64(image.Rect(0, 0, Width, Height))
	maxCount := 0
	for _, c := range counts {
		if c > maxCount {
			maxCount = c
		}
	}

	heatMap := make(map[int]int)
	for _, c := range counts {
		heatMap[c]++
	}

	var heats []int
	for heat := range heatMap {
		heats = append(heats, heat)
	}
	sort.Ints(heats)

	reverseHeats := make(map[int]int)
	for i, heat := range heats {
		reverseHeats[heat] = i
	}

	fmt.Println(maxCount)
	fmt.Println(minY, maxY, minYP, maxYP)
	for i, c := range counts {

		pHeat := float64(reverseHeats[c]) / float64(len(reverseHeats))

		y := math.MaxUint16 * pHeat * 2
		var col color.RGBA64
		switch {
		case y < 1*math.MaxUint16:
			col = color.RGBA64{
				R: 0,
				G: 0,
				B: uint16(y),
				A: 0xffff,
			}
		case y < 1.5*math.MaxUint16:
			col = color.RGBA64{
				R: uint16(2 * (y - 1*math.MaxUint16)),
				G: 0,
				B: 0xffff,
				A: 0xffff,
			}
		case y < 2*math.MaxUint16:
			col = color.RGBA64{
				R: 0xffff,
				G: uint16(2 * (y - 1.5*math.MaxUint16)),
				B: 0xffff,
				A: 0xffff,
			}
		default:
			col = color.RGBA64{
				R: 0xffff,
				G: 0xffff,
				B: 0xffff,
				A: 0xffff,
			}
		}

		img.Set(i%Width, i/Width, col)
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

func mainCmd() *cobra.Command {
	cmd := &cobra.Command{
		Args: cobra.ExactArgs(0),
		RunE: runCmd,
	}

	return cmd
}

type CellCount struct {
	Cell, Count int
}

func main() {
	ctx := context.Background()

	err := mainCmd().ExecuteContext(ctx)
	if err != nil {
		// At this point the error has already been printed; no need to print again.
		os.Exit(1)
	}
}
