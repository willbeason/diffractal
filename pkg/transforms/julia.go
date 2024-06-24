package transforms

import "math/cmplx"

type Julia2 struct {
	C complex128
}

func (j Julia2) Next(z complex128) complex128 {
	return z*z + j.C
}

type JuliaN struct {
	N complex128
	C complex128
}

func (j JuliaN) Next(z complex128) complex128 {
	return cmplx.Pow(z, j.N) + j.C
}
