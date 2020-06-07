package framework

import "math/big"

// ArithmeticLogic interface for mpc arithmetic protocols.
type ArithmeticLogic interface {
	Add(a, b Share, aProof, bProof []big.Int) (Share, []big.Int)
	Multiply_f(a Share, b Share) Share
	Multiply_const(a Share, i *big.Int, aProof []big.Int) (Share, []big.Int)
	Comparison(a Share, b Share) Share
}
