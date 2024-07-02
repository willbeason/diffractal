package transforms

type Linear struct {
	Multiply complex128
	Add      complex128
}

func (l Linear) Next(z complex128) complex128 {
	return z*l.Multiply + l.Add
}
