package framework

import "math/big"

// ArithmeticLogic interface for mpc arithmetic protocols.
type ArithmeticLogic interface {
	Add(a Share, b Share) Share
	Multiply_f(a Share, b Share) Share
	Multiply_const(a Share, i *big.Int) Share
	Comparison(a Share, b Share) Share
}
