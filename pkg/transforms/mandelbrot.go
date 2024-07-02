package transforms

type Mandelbrot struct {
	C complex128
}

func (m Mandelbrot) Next(z complex128, c complex128) complex128 {
	return z*z + c + m.C
}
